package launcher

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResolveDSHCommandUsesOfflineRuntime(t *testing.T) {
	root := filepath.Join(t.TempDir(), "中文 安装目录", "offline-runtime")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	version := "0.1.0-rc.6"
	nodePath := filepath.Join(root, offlineNodeName())
	dshPath := filepath.Join(root, "node_modules", "@deepseek-ai", "dsh", "lib", "bin.js")
	if err := os.MkdirAll(filepath.Dir(dshPath), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(root, "dsh-version.txt"): version + "\n",
		nodePath:                               "node",
		dshPath:                                "dsh",
	} {
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv(offlineRuntimeEnv, root)

	command, err := resolveDSHCommand(version)
	if err != nil {
		t.Fatalf("resolveDSHCommand() error = %v", err)
	}
	if command.mode != "offline" || command.nodePath != nodePath || command.commandPath != nodePath {
		t.Fatalf("unexpected offline command: %#v", command)
	}
	if len(command.prefix) != 1 || command.prefix[0] != dshPath {
		t.Fatalf("unexpected offline prefix: %#v", command.prefix)
	}
}

func TestResolveDSHCommandRejectsOfflineVersionMismatch(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "dsh-version.txt"), []byte("0.1.0-rc.5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(offlineRuntimeEnv, root)

	_, err := resolveDSHCommand("0.1.0-rc.6")
	if err == nil || !strings.Contains(err.Error(), "版本") {
		t.Fatalf("expected version mismatch, got %v", err)
	}
}

func TestOnlineDSHPrefixPrefersNPMCache(t *testing.T) {
	prefix := onlineDSHPrefix([]string{"npx-cli.js"}, "0.1.0-rc.6")
	want := []string{
		"npx-cli.js",
		"--yes",
		"--prefer-offline",
		"--package=@deepseek-ai/dsh@0.1.0-rc.6",
		"dsh",
	}
	if strings.Join(prefix, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("onlineDSHPrefix() = %#v, want %#v", prefix, want)
	}
}

func TestOfflineRuntimeStartsWeb(t *testing.T) {
	if os.Getenv("DSH_DESKTOP_RUN_OFFLINE_INTEGRATION") != "1" {
		t.Skip("set DSH_DESKTOP_RUN_OFFLINE_INTEGRATION=1 to run the packaged runtime")
	}
	root := strings.TrimSpace(os.Getenv(offlineRuntimeEnv))
	if root == "" {
		t.Fatal("DSH_DESKTOP_OFFLINE_ROOT is required")
	}
	versionData, err := os.ReadFile(filepath.Join(root, "dsh-version.txt"))
	if err != nil {
		t.Fatal(err)
	}
	workingDir := filepath.Join(t.TempDir(), "中文 工作区")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatal(err)
	}
	process, err := Start(context.Background(), Config{
		Version:    strings.TrimSpace(string(versionData)),
		WorkingDir: workingDir,
		ProxyMode:  "disabled",
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	t.Cleanup(func() {
		_ = process.Stop(5 * time.Second)
	})
	if process.RuntimeMode() != "offline" {
		t.Fatalf("RuntimeMode() = %q, want offline", process.RuntimeMode())
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := process.WaitReady(ctx, 90*time.Second); err != nil {
		t.Fatalf("WaitReady() error = %v", err)
	}
}

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

func TestInspectLineRejectsNonLoopbackAnnouncement(t *testing.T) {
	process := &Process{urlReady: make(chan struct{})}
	process.inspectLine("dsh web: http://example.com:41234/")
	if process.URL() != "" {
		t.Fatalf("unexpected unsafe URL: %s", process.URL())
	}
	select {
	case <-process.urlReady:
		t.Fatal("unsafe URL marked process ready")
	default:
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
