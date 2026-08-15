package config

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestNormalizeSettingsCustomProxy(t *testing.T) {
	settings, err := Normalize(Settings{ProxyMode: ProxyModeCustom, ProxyURL: "127.0.0.1:10808"})
	if err != nil {
		t.Fatalf("normalizeSettings() 返回错误：%v", err)
	}
	if settings.ProxyURL != "http://127.0.0.1:10808" {
		t.Fatalf("代理地址未被规范化：%s", settings.ProxyURL)
	}
}

func TestNormalizeSettingsRejectsUnsupportedProxy(t *testing.T) {
	_, err := Normalize(Settings{ProxyMode: ProxyModeCustom, ProxyURL: "socks5://127.0.0.1:10808"})
	if err == nil {
		t.Fatal("预期拒绝不支持的 SOCKS5 代理")
	}
}

func TestNormalizeSettingsClearsUnusedProxyURL(t *testing.T) {
	settings, err := Normalize(Settings{ProxyMode: ProxyModeDisabled, ProxyURL: "http://127.0.0.1:10808"})
	if err != nil {
		t.Fatalf("normalizeSettings() 返回错误：%v", err)
	}
	if settings.ProxyURL != "" {
		t.Fatalf("禁用模式不应保留代理地址：%s", settings.ProxyURL)
	}
}

func TestSettingsRoundTripInChineseAndSpacePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "用户 配置", "代理设置", "settings.json")
	want := Settings{ProxyMode: ProxyModeCustom, ProxyURL: "http://127.0.0.1:10808", DSHVersion: "0.1.0-rc.7"}
	if err := saveFile(path, want); err != nil {
		t.Fatalf("中文路径保存配置失败：%v", err)
	}
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("中文路径读取配置失败：%v", err)
	}
	if got != want {
		t.Fatalf("配置往返不一致：got %#v, want %#v", got, want)
	}
}

func TestSaveFileIfUnchangedRejectsStaleWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	initial := Default()
	updated := Settings{ProxyMode: ProxyModeCustom, ProxyURL: "http://127.0.0.1:10808", DSHVersion: "0.1.0-rc.7"}
	other := Settings{ProxyMode: ProxyModeDisabled}

	if err := saveFile(path, initial); err != nil {
		t.Fatalf("初始配置保存失败：%v", err)
	}
	if err := saveFileIfUnchanged(path, initial, updated); err != nil {
		t.Fatalf("首次条件保存失败：%v", err)
	}
	if err := saveFileIfUnchanged(path, initial, other); !errors.Is(err, ErrConflict) {
		t.Fatalf("预期检测到过期配置，得到：%v", err)
	}
	got, err := loadFile(path)
	if err != nil {
		t.Fatalf("读取最终配置失败：%v", err)
	}
	if got != updated {
		t.Fatalf("过期写入不应覆盖新配置：got %#v, want %#v", got, updated)
	}
}

func TestNormalizeSettingsRejectsFloatingDSHVersion(t *testing.T) {
	_, err := Normalize(Settings{ProxyMode: ProxyModeInherit, DSHVersion: "latest"})
	if err == nil {
		t.Fatal("配置不应接受未固定的 latest DSH 版本")
	}
}
