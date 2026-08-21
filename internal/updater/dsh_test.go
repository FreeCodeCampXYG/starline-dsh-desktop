package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"starline-dsh-desktop/internal/config"
)

func TestCheckDSHChannelsFindsLatestAndNext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		if request.Header.Get("Cache-Control") != "no-cache" {
			t.Error("版本检查必须跳过陈旧的中间缓存")
		}
		_, _ = writer.Write([]byte(`{"latest":"0.1.0-rc.7","next":"0.1.0-rc.8"}`))
	}))
	defer server.Close()

	result, err := checkDSHChannels(context.Background(), &http.Client{Timeout: time.Second}, server.URL, "0.1.0-rc.6")
	if err != nil {
		t.Fatalf("检查 DSH 更新失败：%v", err)
	}
	if !result.LatestUpdateAvailable || result.LatestVersion != "0.1.0-rc.7" {
		t.Fatalf("没有识别到 latest：%+v", result)
	}
	if !result.NextUpdateAvailable || result.NextVersion != "0.1.0-rc.8" {
		t.Fatalf("没有识别到 next：%+v", result)
	}
}

func TestCheckDSHChannelsAllowsMissingNext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"latest":"0.1.0-rc.7"}`))
	}))
	defer server.Close()

	result, err := checkDSHChannels(context.Background(), server.Client(), server.URL, "0.1.0-rc.7")
	if err != nil {
		t.Fatalf("缺少 next 时不应让稳定版检查失败：%v", err)
	}
	if result.NextVersion != "" || result.NextUpdateAvailable {
		t.Fatalf("缺少 next 时结果不正确：%+v", result)
	}
}

func TestCheckDSHChannelsRecognizesNextMinorRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"latest":"0.1.1-rc.2","next":"0.1.1-rc.2"}`))
	}))
	defer server.Close()

	result, err := checkDSHChannels(context.Background(), server.Client(), server.URL, "0.1.0-rc.7")
	if err != nil {
		t.Fatalf("检查跨 minor 版本更新失败：%v", err)
	}
	if !result.LatestUpdateAvailable || !result.NextUpdateAvailable || result.LatestVersion != "0.1.1-rc.2" {
		t.Fatalf("没有识别到官方新版本：%+v", result)
	}
}

func TestRegistryClientUsesCustomProxy(t *testing.T) {
	settings := config.Settings{
		ProxyMode: config.ProxyModeCustom,
		ProxyURL:  "http://127.0.0.1:10808",
	}
	client, err := registryClient(settings)
	if err != nil {
		t.Fatalf("创建更新检查客户端失败：%v", err)
	}
	defer client.CloseIdleConnections()
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("更新检查没有使用可审查的 HTTP transport")
	}
	request, err := http.NewRequest(http.MethodGet, registryDistTagsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	proxyURL, err := transport.Proxy(request)
	if err != nil {
		t.Fatalf("解析更新代理失败：%v", err)
	}
	if proxyURL == nil || proxyURL.String() != settings.ProxyURL {
		t.Fatalf("更新检查没有沿用自定义代理：%v", proxyURL)
	}
}

func TestAutomaticRegistryEndpointsUseDirectMirrorFirst(t *testing.T) {
	endpoints := []registryEndpoint{
		{registryMirrorDistTagsURL, config.Settings{ProxyMode: config.ProxyModeDisabled}},
		{registryDistTagsURL, config.Settings{ProxyMode: config.ProxyModeDisabled}},
	}
	if endpoints[0].url != registryMirrorDistTagsURL || endpoints[0].settings.ProxyMode != config.ProxyModeDisabled {
		t.Fatalf("自动检查没有优先直连国内镜像：%+v", endpoints)
	}
}

func TestManualRegistryEndpointsFallBackByNetworkRoute(t *testing.T) {
	endpoints := manualRegistryEndpoints(config.Settings{
		ProxyMode: config.ProxyModeCustom,
		ProxyURL:  "http://127.0.0.1:1080",
	})
	wantModes := []string{
		config.ProxyModeCustom, config.ProxyModeCustom,
		config.ProxyModeInherit, config.ProxyModeInherit,
		config.ProxyModeDisabled, config.ProxyModeDisabled,
	}
	if len(endpoints) != len(wantModes) {
		t.Fatalf("手动更新端点数量 = %d，期望 %d", len(endpoints), len(wantModes))
	}
	for index, wantMode := range wantModes {
		if endpoints[index].settings.ProxyMode != wantMode {
			t.Fatalf("端点 %d 的代理模式 = %q，期望 %q", index, endpoints[index].settings.ProxyMode, wantMode)
		}
		if index%2 == 0 && endpoints[index].url != registryMirrorDistTagsURL {
			t.Fatalf("网络路径 %d 没有优先国内镜像", index/2)
		}
	}
}
