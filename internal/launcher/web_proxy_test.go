package launcher

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestWebProxyBootstrapsSessionForIframeAndAPI(t *testing.T) {
	var mu sync.Mutex
	authCalls := 0
	target := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Has("token") {
			mu.Lock()
			authCalls++
			mu.Unlock()
			http.SetCookie(response, &http.Cookie{Name: "dsh-auth-test", Value: "ok", Path: "/", HttpOnly: true})
			http.Redirect(response, request, "/", http.StatusSeeOther)
			return
		}
		if request.Header.Get("Origin") != "" || request.Header.Get("Sec-Fetch-Site") != "" {
			http.Error(response, "proxy forwarded browser origin", http.StatusForbidden)
			return
		}
		cookie, err := request.Cookie("dsh-auth-test")
		if err != nil || cookie.Value != "ok" {
			http.Error(response, "missing session", http.StatusUnauthorized)
			return
		}
		if request.URL.Path == "/api/status" {
			_, _ = response.Write([]byte(`{"ok":true}`))
			return
		}
		_, _ = response.Write([]byte("<title>DSH Local Build</title>"))
	}))
	defer target.Close()

	proxy, err := newWebProxy(target.URL, target.URL+"/?token=test-token")
	if err != nil {
		t.Fatalf("创建 Web 代理失败：%v", err)
	}
	defer proxy.Close()

	client := &http.Client{}
	var wait sync.WaitGroup
	for index := 0; index < 8; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			request, requestErr := http.NewRequest(http.MethodGet, proxy.URL()+"/", nil)
			if requestErr != nil {
				t.Errorf("创建代理请求失败：%v", requestErr)
				return
			}
			request.Header.Set("Origin", "wails://wails")
			request.Header.Set("Sec-Fetch-Site", "cross-site")
			response, requestErr := client.Do(request)
			if requestErr != nil {
				t.Errorf("访问代理根页面失败：%v", requestErr)
				return
			}
			body, _ := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "DSH Local Build") {
				t.Errorf("代理根页面异常：status=%d body=%q", response.StatusCode, body)
			}
			if len(response.Cookies()) != 0 {
				t.Errorf("代理不应向 iframe 暴露 DSH 会话 Cookie")
			}
		}()
	}
	wait.Wait()

	apiResponse, err := client.Get(proxy.URL() + "/api/status")
	if err != nil {
		t.Fatalf("访问代理 API 失败：%v", err)
	}
	apiBody, _ := io.ReadAll(apiResponse.Body)
	_ = apiResponse.Body.Close()
	if apiResponse.StatusCode != http.StatusOK || string(apiBody) != `{"ok":true}` {
		t.Fatalf("代理 API 异常：status=%d body=%q", apiResponse.StatusCode, apiBody)
	}
	mu.Lock()
	defer mu.Unlock()
	if authCalls != 1 {
		t.Fatalf("token 握手次数 = %d, want 1", authCalls)
	}
}

func TestWebProxyRejectsDifferentAuthorities(t *testing.T) {
	proxy, err := newWebProxy("http://127.0.0.1:3001", "http://127.0.0.1:3002/?token=test")
	if err == nil || proxy != nil {
		t.Fatal("不同 authority 不应创建 Web 代理")
	}
}
