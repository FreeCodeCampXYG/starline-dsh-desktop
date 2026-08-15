package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckDSHLatestFindsNewerRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"name":"@deepseek-ai/dsh","version":"0.1.0-rc.7"}`))
	}))
	defer server.Close()

	result, err := checkDSHLatest(context.Background(), &http.Client{Timeout: time.Second}, server.URL, "0.1.0-rc.6")
	if err != nil {
		t.Fatalf("检查 DSH 更新失败：%v", err)
	}
	if !result.UpdateAvailable || result.LatestVersion != "0.1.0-rc.7" {
		t.Fatalf("没有识别到新版本：%+v", result)
	}
}

func TestCheckDSHLatestRejectsUnexpectedPackage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"name":"unexpected","version":"0.1.0-rc.7"}`))
	}))
	defer server.Close()

	if _, err := checkDSHLatest(context.Background(), server.Client(), server.URL, "0.1.0-rc.6"); err == nil {
		t.Fatal("不应接受其他 npm 包的版本信息")
	}
}
