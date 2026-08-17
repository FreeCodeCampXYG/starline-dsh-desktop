package application

import (
	"context"
	"testing"

	"starline-dsh-desktop/internal/config"
	"starline-dsh-desktop/internal/updater"
)

func TestBeforeClosePlatformPolicy(t *testing.T) {
	app := &App{}
	if allowWindowCloseWithoutTray() {
		if app.BeforeClose(context.Background()) {
			t.Fatal("Unix platforms should allow the native window close")
		}
		return
	}

	if !app.BeforeClose(context.Background()) {
		t.Fatal("Windows should prevent window close before an explicit quit")
	}

	app.RequestQuit()
	if app.BeforeClose(context.Background()) {
		t.Fatal("explicit quit should allow Wails shutdown")
	}
}

func TestResolveDSHVersion(t *testing.T) {
	t.Setenv("DSH_DESKTOP_DSH_VERSION", " 0.1.0-test ")
	got, err := resolveDSHVersion("0.1.0-rc.6", config.Default())
	if err != nil {
		t.Fatalf("解析版本覆盖失败：%v", err)
	}
	if got != "0.1.0-test" {
		t.Fatalf("版本覆盖未规范化：%q", got)
	}
}

func TestResolveDSHVersionUsesFallback(t *testing.T) {
	t.Setenv("DSH_DESKTOP_DSH_VERSION", " ")
	got, err := resolveDSHVersion("0.1.0-rc.6", config.Default())
	if err != nil {
		t.Fatalf("解析默认版本失败：%v", err)
	}
	if got != "0.1.0-rc.6" {
		t.Fatalf("默认版本不一致：%q", got)
	}
}

func TestResolveDSHVersionUsesSavedOnlineVersion(t *testing.T) {
	t.Setenv("DSH_DESKTOP_DSH_VERSION", "")
	settings := config.Default()
	settings.DSHVersion = "0.1.0-rc.7"
	got, err := resolveDSHVersion("0.1.0-rc.6", settings)
	if err != nil {
		t.Fatalf("解析保存的 DSH 版本失败：%v", err)
	}
	if got != "0.1.0-rc.7" {
		t.Fatalf("保存的 DSH 版本未生效：%q", got)
	}
}

func TestSelectDSHUpdateTarget(t *testing.T) {
	release := updater.DSHRelease{
		LatestVersion:         "0.1.0-rc.7",
		NextVersion:           "0.1.0-rc.8",
		LatestUpdateAvailable: true,
		NextUpdateAvailable:   true,
	}
	for _, test := range []struct {
		channel string
		want    string
	}{
		{channel: "latest", want: "0.1.0-rc.7"},
		{channel: "next", want: "0.1.0-rc.8"},
	} {
		got, err := selectDSHUpdateTarget(release, test.channel)
		if err != nil || got != test.want {
			t.Fatalf("selectDSHUpdateTarget(%q) = %q, %v", test.channel, got, err)
		}
	}
	if _, err := selectDSHUpdateTarget(release, "preview"); err == nil {
		t.Fatal("不应接受未知更新通道")
	}
}
