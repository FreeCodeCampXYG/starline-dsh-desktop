package launcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseNodeVersion(t *testing.T) {
	tests := []struct {
		input      string
		major      int
		minor      int
		acceptable bool
	}{
		{"v22.19.0\n", 22, 19, true},
		{"v24.7.0", 24, 7, true},
		{"unknown", 0, 0, false},
	}
	for _, test := range tests {
		major, minor, ok := parseNodeVersion(test.input)
		if major != test.major || minor != test.minor || ok != test.acceptable {
			t.Fatalf("parseNodeVersion(%q) = %d, %d, %v", test.input, major, minor, ok)
		}
	}
}

func TestWaitReadyRequiresDSHPageFingerprint(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"DSH 页面", "<!doctype html><title>DeepSeek Harness</title>", false},
		{"其他页面", "<!doctype html><title>Other App</title>", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			ready := make(chan struct{})
			close(ready)
			process := &Process{url: server.URL, urlReady: ready, done: make(chan struct{})}
			err := process.WaitReady(context.Background(), 300*time.Millisecond)
			if (err != nil) != test.wantErr {
				t.Fatalf("WaitReady() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

func TestSafeLoopbackURL(t *testing.T) {
	for _, value := range []string{
		"http://127.0.0.1:3080",
		"http://localhost:4000/path",
		"http://[::1]:8080",
	} {
		if !safeLoopbackURL(value) {
			t.Fatalf("expected safe URL: %s", value)
		}
	}
	for _, value := range []string{
		"https://127.0.0.1:3080",
		"http://example.com:3080",
		"javascript:alert(1)",
	} {
		if safeLoopbackURL(value) {
			t.Fatalf("expected unsafe URL: %s", value)
		}
	}
}

func TestInspectLineAcceptsOnlyDSHLoopbackURL(t *testing.T) {
	process := &Process{urlReady: make(chan struct{})}
	process.inspectLine("dsh web: http://127.0.0.1:41234/")
	if process.URL() != "http://127.0.0.1:41234" {
		t.Fatalf("unexpected URL: %s", process.URL())
	}
}

func TestMergeNoProxyPreservesUserValues(t *testing.T) {
	environment := mergeNoProxy(
		[]string{"PATH=/bin", "no_proxy=example.com,localhost", "HTTP_PROXY=http://proxy"},
		[]string{"127.0.0.1", "localhost", "::1"},
	)
	joined := strings.Join(environment, "\n")
	if strings.Count(strings.ToLower(joined), "no_proxy=") != 1 {
		t.Fatalf("expected exactly one NO_PROXY entry: %s", joined)
	}
	for _, value := range []string{"example.com", "localhost", "127.0.0.1", "::1"} {
		if !strings.Contains(joined, value) {
			t.Fatalf("NO_PROXY does not contain %s: %s", value, joined)
		}
	}
}

func TestChildEnvironmentUsesCustomProxyAndKeepsLoopbackDirect(t *testing.T) {
	environment := childEnvironment(
		[]string{"PATH=/bin", "HTTPS_PROXY=http://old:8080", "NO_PROXY=example.com"},
		"custom",
		"http://127.0.0.1:10808",
	)
	joined := strings.Join(environment, "\n")
	if strings.Contains(joined, "http://old:8080") {
		t.Fatalf("旧代理没有被移除：%s", joined)
	}
	for _, value := range []string{"HTTP_PROXY=http://127.0.0.1:10808", "HTTPS_PROXY=http://127.0.0.1:10808", "example.com", "127.0.0.1", "localhost", "::1"} {
		if !strings.Contains(joined, value) {
			t.Fatalf("环境变量缺少 %s：%s", value, joined)
		}
	}
}

func TestChildEnvironmentCanDisableInheritedProxy(t *testing.T) {
	environment := childEnvironment(
		[]string{"HTTP_PROXY=http://proxy:8080", "ALL_PROXY=socks5://proxy:1080"},
		"disabled",
		"",
	)
	joined := strings.ToLower(strings.Join(environment, "\n"))
	if strings.Contains(joined, "http_proxy=") || strings.Contains(joined, "all_proxy=") {
		t.Fatalf("禁用模式仍然保留代理：%s", joined)
	}
	if !strings.Contains(joined, "no_proxy=127.0.0.1,localhost,::1") {
		t.Fatalf("禁用模式没有保留本地直连：%s", joined)
	}
}
