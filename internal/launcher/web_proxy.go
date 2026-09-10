package launcher

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const webProxyStartupTimeout = 10 * time.Second

type webProxy struct {
	server    *http.Server
	listener  net.Listener
	publicURL *url.URL
	targetURL *url.URL
	authURL   *url.URL
	client    *http.Client
	jar       http.CookieJar
	reverse   *httputil.ReverseProxy

	bootstrapOnce sync.Once
	bootstrapErr  error
	closeOnce     sync.Once
}

func newWebProxy(targetRaw, authRaw string) (*webProxy, error) {
	target, err := parseProxyURL(targetRaw)
	if err != nil {
		return nil, err
	}
	authURL, err := parseProxyURL(authRaw)
	if err != nil {
		return nil, err
	}
	if target.Scheme != authURL.Scheme || target.Host != authURL.Host {
		return nil, errors.New("DSH 认证地址与监听地址不属于同一 loopback authority")
	}
	target.RawQuery = ""
	target.Fragment = ""
	if authURL.Query().Get("token") == "" {
		return nil, errors.New("DSH 认证地址缺少 token")
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("无法创建 DSH 会话 CookieJar：%w", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("无法启动内嵌 Web 代理：%w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	publicURL := &url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(port))}
	transport := &http.Transport{Proxy: nil}
	proxy := &webProxy{
		listener:  listener,
		publicURL: publicURL,
		targetURL: target,
		authURL:   authURL,
		jar:       jar,
		client: &http.Client{
			Jar:       jar,
			Transport: transport,
			Timeout:   webProxyStartupTimeout,
		},
	}
	proxy.reverse = &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(target)
			request.Out.Host = target.Host
			// DSH 的 API Host/Origin fence 只应看到代理已验证的目标 authority；
			// 转发 WebView 的跨源标记会被上游当成越权请求，且不会增加认证安全性。
			for _, header := range []string{"Cookie", "Origin", "Referer", "Sec-Fetch-Site", "Sec-Fetch-Mode", "Sec-Fetch-Dest"} {
				request.Out.Header.Del(header)
			}
			for _, cookie := range jar.Cookies(request.Out.URL) {
				request.Out.AddCookie(cookie)
			}
		},
		ModifyResponse: func(response *http.Response) error {
			// Cookie 只保存在 Go 的内存 Jar；不把 DSH authority cookie 暴露给 iframe。
			response.Header.Del("Set-Cookie")
			if location := response.Header.Get("Location"); location != "" {
				if rewritten := rewriteProxyLocation(location, target, publicURL); rewritten != "" {
					response.Header.Set("Location", rewritten)
				}
			}
			return nil
		},
	}
	proxy.server = &http.Server{Handler: http.HandlerFunc(proxy.serveHTTP)}
	go func() {
		_ = proxy.server.Serve(listener)
	}()
	return proxy, nil
}

func parseProxyURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !safeLoopbackURL(parsed.String()) || parsed.Host == "" {
		return nil, fmt.Errorf("内嵌 Web 代理只接受有效的 loopback HTTP 地址：%q", raw)
	}
	return parsed, nil
}

func (p *webProxy) serveHTTP(response http.ResponseWriter, request *http.Request) {
	// 并发首访只能完成一次 token 握手，失败原因保留给所有等待者，避免重复消费或竞争认证状态。
	p.bootstrapOnce.Do(func() {
		p.bootstrapErr = p.bootstrap()
	})
	if p.bootstrapErr != nil {
		http.Error(response, "DSH Web 认证代理初始化失败", http.StatusBadGateway)
		return
	}
	p.reverse.ServeHTTP(response, request)
}

func (p *webProxy) bootstrap() error {
	request, err := http.NewRequest(http.MethodGet, p.authURL.String(), nil)
	if err != nil {
		return fmt.Errorf("无法创建 DSH 认证请求：%w", err)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("DSH token 握手失败：%w", err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("DSH token 握手返回 HTTP %d", response.StatusCode)
	}
	if len(p.jar.Cookies(p.targetURL)) == 0 {
		return errors.New("DSH token 握手未生成会话 Cookie")
	}
	return nil
}

func rewriteProxyLocation(raw string, target, public *url.URL) string {
	location, err := url.Parse(raw)
	if err != nil || location.Host != target.Host || location.Scheme != target.Scheme {
		return raw
	}
	location.Scheme = public.Scheme
	location.Host = public.Host
	return location.String()
}

func (p *webProxy) URL() string {
	return strings.TrimRight(p.publicURL.String(), "/")
}

func (p *webProxy) Close() error {
	var closeErr error
	p.closeOnce.Do(func() {
		closeErr = p.server.Close()
		if closeErr != nil {
			closeErr = p.listener.Close()
		}
	})
	return closeErr
}
