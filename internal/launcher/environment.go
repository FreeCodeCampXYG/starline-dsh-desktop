package launcher

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	npmOfficialRegistry     = "https://registry.npmjs.org"
	npmMirrorRegistry       = "https://registry.npmmirror.com"
	npmFetchTimeout         = "10000"
	npmFetchRetries         = "1"
	npmFetchRetryMinTimeout = "1000"
	npmFetchRetryMaxTimeout = "3000"
	proxyProbeTimeout       = 800 * time.Millisecond
	npmRegistryProbeTimeout = 3 * time.Second
)

const (
	proxyFallbackNone   = ""
	proxyFallbackSystem = "system"
	proxyFallbackDirect = "direct"
)

// childEnvironmentWithFallback 按自定义代理、系统环境代理、国内镜像直连的顺序选择网络路径。
// disabled 是用户明确选择的直连模式，不会再尝试任何代理。
func childEnvironmentWithFallback(environment []string, proxyMode, proxyURL string) ([]string, string) {
	systemProxy := systemProxyFromEnvironment(environment)
	switch proxyMode {
	case "custom":
		if proxyEndpointReachable(proxyURL) {
			return childEnvironment(environment, "custom", proxyURL), proxyFallbackNone
		}
		if systemProxy != "" && !strings.EqualFold(systemProxy, proxyURL) && proxyEndpointReachable(systemProxy) {
			return childEnvironment(environment, "custom", systemProxy), proxyFallbackSystem
		}
		return childEnvironment(environment, "disabled", ""), proxyFallbackDirect
	case "inherit":
		if systemProxy != "" && proxyEndpointReachable(systemProxy) {
			return childEnvironment(environment, "custom", systemProxy), proxyFallbackNone
		}
		if systemProxy != "" {
			return childEnvironment(environment, "disabled", ""), proxyFallbackDirect
		}
		return childEnvironment(environment, "disabled", ""), proxyFallbackNone
	default:
		return childEnvironment(environment, "disabled", ""), proxyFallbackNone
	}
}

func systemProxyFromEnvironment(environment []string) string {
	keys := []string{"HTTPS_PROXY", "HTTP_PROXY", "ALL_PROXY", "npm_config_https_proxy", "npm_config_proxy"}
	for _, wanted := range keys {
		for _, entry := range environment {
			key, value, found := strings.Cut(entry, "=")
			if !found || !strings.EqualFold(key, wanted) {
				continue
			}
			parsed, err := url.Parse(strings.TrimSpace(value))
			if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Hostname() != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func proxyEndpointReachable(rawURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Hostname() == "" {
		return false
	}
	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	if _, err := strconv.Atoi(port); err != nil {
		return false
	}
	address := net.JoinHostPort(parsed.Hostname(), port)
	ctx, cancel := context.WithTimeout(context.Background(), proxyProbeTimeout)
	defer cancel()
	connection, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return false
	}
	_ = connection.Close()
	return true
}

func childEnvironment(environment []string, proxyMode, proxyURL string) []string {
	return childEnvironmentWithRegistry(environment, proxyMode, proxyURL, npmMirrorRegistry)
}

// childEnvironmentWithRegistry 为子进程固定本次已选择的 registry；默认代理语义保持不变。
func childEnvironmentWithRegistry(environment []string, proxyMode, proxyURL, registry string) []string {
	if proxyMode == "custom" || proxyMode == "disabled" {
		environment = withoutProxyVariables(environment)
	}
	if proxyMode == "custom" {
		environment = append(
			environment,
			"HTTP_PROXY="+proxyURL,
			"HTTPS_PROXY="+proxyURL,
			"http_proxy="+proxyURL,
			"https_proxy="+proxyURL,
			"npm_config_proxy="+proxyURL,
			"npm_config_https_proxy="+proxyURL,
		)
	}
	environment = withNPMRegistry(environment, registry)
	// 限制 npm 的单次网络等待和重试，避免代理端口失效时 npx 长时间无响应。
	environment = append(environment,
		"npm_config_fetch_timeout="+npmFetchTimeout,
		"npm_config_fetch_retries="+npmFetchRetries,
		"npm_config_fetch_retry_mintimeout="+npmFetchRetryMinTimeout,
		"npm_config_fetch_retry_maxtimeout="+npmFetchRetryMaxTimeout,
		"npm_config_update_notifier=false",
	)
	return mergeNoProxy(environment, []string{"127.0.0.1", "localhost", "::1"})
}

// npmRegistryCandidates 保证官方 npm 是普通启动的第一选择，镜像只作为失败回退。
func npmRegistryCandidates() []string {
	return []string{npmOfficialRegistry, npmMirrorRegistry}
}

// npmRegistryReachable 只探测受信任 registry 的 dist-tags 端点，不下载包体。
func npmRegistryReachable(registry string, environment []string) bool {
	client := &http.Client{Timeout: npmRegistryProbeTimeout}
	client.Transport = &http.Transport{Proxy: proxyFromEnvironment(environment)}
	request, err := http.NewRequest(http.MethodGet, registry+"/-/package/@deepseek-ai%2Fdsh/dist-tags", nil)
	if err != nil {
		return false
	}
	response, err := client.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	return response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices
}

func proxyFromEnvironment(environment []string) func(*http.Request) (*url.URL, error) {
	return func(request *http.Request) (*url.URL, error) {
		for _, key := range []string{"HTTPS_PROXY", "HTTP_PROXY", "ALL_PROXY"} {
			for _, entry := range environment {
				name, value, found := strings.Cut(entry, "=")
				if found && strings.EqualFold(name, key) && strings.TrimSpace(value) != "" {
					parsed, err := url.Parse(strings.TrimSpace(value))
					if err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") {
						return parsed, nil
					}
				}
			}
		}
		return nil, nil
	}
}

func withNPMRegistry(environment []string, registry string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		key, _, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(key, "npm_config_registry") {
			continue
		}
		result = append(result, entry)
	}
	return append(result, "npm_config_registry="+registry)
}

func npmRegistry(environment []string) string {
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(key, "npm_config_registry") {
			if parsed, err := url.Parse(strings.TrimSpace(value)); err == nil && parsed.Scheme != "" && parsed.Hostname() != "" {
				return strings.TrimRight(parsed.String(), "/")
			}
		}
	}
	return "https://registry.npmjs.org"
}

func withoutProxyVariables(environment []string) []string {
	proxyKeys := map[string]bool{
		"http_proxy":             true,
		"https_proxy":            true,
		"all_proxy":              true,
		"npm_config_proxy":       true,
		"npm_config_https_proxy": true,
		"npm_config_noproxy":     true,
	}
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, found := strings.Cut(entry, "=")
		if found && proxyKeys[strings.ToLower(key)] {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func mergeNoProxy(environment []string, required []string) []string {
	result := make([]string, 0, len(environment)+1)
	values := make([]string, 0, len(required)+4)
	seen := make(map[string]bool)
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(key, "NO_PROXY") {
			for _, item := range strings.Split(value, ",") {
				trimmed := strings.TrimSpace(item)
				if trimmed != "" && !seen[strings.ToLower(trimmed)] {
					values = append(values, trimmed)
					seen[strings.ToLower(trimmed)] = true
				}
			}
			continue
		}
		result = append(result, entry)
	}
	for _, item := range required {
		if !seen[strings.ToLower(item)] {
			values = append(values, item)
			seen[strings.ToLower(item)] = true
		}
	}
	return append(result, "NO_PROXY="+strings.Join(values, ","))
}
