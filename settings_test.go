package main

import "testing"

func TestNormalizeSettingsCustomProxy(t *testing.T) {
	settings, err := normalizeSettings(Settings{ProxyMode: proxyModeCustom, ProxyURL: "127.0.0.1:10808"})
	if err != nil {
		t.Fatalf("normalizeSettings() 返回错误：%v", err)
	}
	if settings.ProxyURL != "http://127.0.0.1:10808" {
		t.Fatalf("代理地址未被规范化：%s", settings.ProxyURL)
	}
}

func TestNormalizeSettingsRejectsUnsupportedProxy(t *testing.T) {
	_, err := normalizeSettings(Settings{ProxyMode: proxyModeCustom, ProxyURL: "socks5://127.0.0.1:10808"})
	if err == nil {
		t.Fatal("预期拒绝不支持的 SOCKS5 代理")
	}
}

func TestNormalizeSettingsClearsUnusedProxyURL(t *testing.T) {
	settings, err := normalizeSettings(Settings{ProxyMode: proxyModeDisabled, ProxyURL: "http://127.0.0.1:10808"})
	if err != nil {
		t.Fatalf("normalizeSettings() 返回错误：%v", err)
	}
	if settings.ProxyURL != "" {
		t.Fatalf("禁用模式不应保留代理地址：%s", settings.ProxyURL)
	}
}
