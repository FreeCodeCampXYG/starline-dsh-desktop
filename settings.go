package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	proxyModeInherit  = "inherit"
	proxyModeCustom   = "custom"
	proxyModeDisabled = "disabled"
)

type Settings struct {
	ProxyMode string `json:"proxyMode"`
	ProxyURL  string `json:"proxyUrl,omitempty"`
}

func defaultSettings() Settings {
	return Settings{ProxyMode: proxyModeInherit}
}

// normalizeSettings 校验界面输入，并把省略协议的常见代理地址补成 HTTP URL。
func normalizeSettings(settings Settings) (Settings, error) {
	settings.ProxyMode = strings.TrimSpace(settings.ProxyMode)
	settings.ProxyURL = strings.TrimSpace(settings.ProxyURL)
	switch settings.ProxyMode {
	case proxyModeInherit, proxyModeDisabled:
		settings.ProxyURL = ""
		return settings, nil
	case proxyModeCustom:
		if settings.ProxyURL == "" {
			return Settings{}, errors.New("自定义代理模式需要填写代理地址")
		}
		if !strings.Contains(settings.ProxyURL, "://") {
			settings.ProxyURL = "http://" + settings.ProxyURL
		}
		parsed, err := url.Parse(settings.ProxyURL)
		if err != nil || parsed.Hostname() == "" {
			return Settings{}, errors.New("代理地址格式无效，例如：http://127.0.0.1:10808")
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return Settings{}, errors.New("当前仅支持 HTTP 或 HTTPS 代理")
		}
		return settings, nil
	default:
		return Settings{}, fmt.Errorf("未知代理模式：%s", settings.ProxyMode)
	}
}

func loadSettings() (Settings, error) {
	path, err := settingsPath()
	if err != nil {
		return defaultSettings(), err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaultSettings(), nil
	}
	if err != nil {
		return defaultSettings(), fmt.Errorf("无法读取配置：%w", err)
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultSettings(), fmt.Errorf("配置文件格式无效：%w", err)
	}
	settings, err = normalizeSettings(settings)
	if err != nil {
		return defaultSettings(), fmt.Errorf("配置内容无效：%w", err)
	}
	return settings, nil
}

// saveSettings 先完整写入临时文件，再替换用户配置，避免留下半份 JSON。
func saveSettings(settings Settings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("无法创建配置目录：%w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), "settings-*.tmp")
	if err != nil {
		return fmt.Errorf("无法创建临时配置：%w", err)
	}
	temporaryPath := file.Name()
	defer os.Remove(temporaryPath)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("无法设置配置权限：%w", err)
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(settings); err != nil {
		_ = file.Close()
		return fmt.Errorf("无法写入配置：%w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("无法关闭配置文件：%w", err)
	}
	// Windows 的 os.Rename 不能直接覆盖已有目标，先移除旧配置再替换。
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("无法替换旧配置：%w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("无法保存配置：%w", err)
	}
	return nil
}

func settingsPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return "", errors.New("无法确定用户配置目录")
	}
	return filepath.Join(base, "starline-dsh-desktop", "settings.json"), nil
}
