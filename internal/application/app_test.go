package application

import (
	"context"
	"testing"
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
	if got := resolveDSHVersion("fallback"); got != "0.1.0-test" {
		t.Fatalf("版本覆盖未规范化：%q", got)
	}
}

func TestResolveDSHVersionUsesFallback(t *testing.T) {
	t.Setenv("DSH_DESKTOP_DSH_VERSION", " ")
	if got := resolveDSHVersion("0.1.0-rc.6"); got != "0.1.0-rc.6" {
		t.Fatalf("默认版本不一致：%q", got)
	}
}
