package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"starline-dsh-desktop/internal/dshversion"
)

const (
	ProxyModeInherit                  = "inherit"
	ProxyModeCustom                   = "custom"
	ProxyModeDisabled                 = "disabled"
	DefaultOnlineStartupTimeoutSeconds = 90
	MinOnlineStartupTimeoutSeconds     = 30
	MaxOnlineStartupTimeoutSeconds     = 600
)

// ErrConflict 表示另一个桌面实例已经修改了配置，当前实例不能覆盖它。
var ErrConflict = errors.New("配置已被其他实例修改，请重新加载后再保存")

type Settings struct {
	ProxyMode                  string `json:"proxyMode"`
	ProxyURL                   string `json:"proxyUrl,omitempty"`
	DSHVersion                 string `json:"dshVersion,omitempty"`
	OnlineStartupTimeoutSeconds int    `json:"onlineStartupTimeoutSeconds,omitempty"`
}

func Default() Settings {
	return Settings{ProxyMode: ProxyModeInherit}
}

// Normalize 校验界面输入，并把省略协议的常见代理地址补成 HTTP URL。
func Normalize(settings Settings) (Settings, error) {
	settings.ProxyMode = strings.TrimSpace(settings.ProxyMode)
	settings.ProxyURL = strings.TrimSpace(settings.ProxyURL)
	settings.DSHVersion = strings.TrimSpace(settings.DSHVersion)
	if settings.OnlineStartupTimeoutSeconds < 0 || settings.OnlineStartupTimeoutSeconds > MaxOnlineStartupTimeoutSeconds {
		return Settings{}, fmt.Errorf("在线启动等待上限必须为 %d-%d 秒", MinOnlineStartupTimeoutSeconds, MaxOnlineStartupTimeoutSeconds)
	}
	if settings.OnlineStartupTimeoutSeconds > 0 && settings.OnlineStartupTimeoutSeconds < MinOnlineStartupTimeoutSeconds {
		return Settings{}, fmt.Errorf("在线启动等待上限不能低于 %d 秒", MinOnlineStartupTimeoutSeconds)
	}
	if settings.DSHVersion != "" {
		version, err := dshversion.Normalize(settings.DSHVersion)
		if err != nil {
			return Settings{}, err
		}
		settings.DSHVersion = version
	}
	switch settings.ProxyMode {
	case ProxyModeInherit, ProxyModeDisabled:
		settings.ProxyURL = ""
		return settings, nil
	case ProxyModeCustom:
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

// EffectiveOnlineStartupTimeoutSeconds 返回在线 npx 启动的有效等待上限；零值兼容旧配置并使用默认值。
func EffectiveOnlineStartupTimeoutSeconds(settings Settings) int {
	if settings.OnlineStartupTimeoutSeconds == 0 {
		return DefaultOnlineStartupTimeoutSeconds
	}
	return settings.OnlineStartupTimeoutSeconds
}

// Load 从当前用户配置目录读取桌面宿主设置。
func Load() (Settings, error) {
	path, err := Path()
	if err != nil {
		return Default(), err
	}
	return loadFile(path)
}

func loadFile(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), fmt.Errorf("无法读取配置：%w", err)
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Default(), fmt.Errorf("配置文件格式无效：%w", err)
	}
	settings, err = Normalize(settings)
	if err != nil {
		return Default(), fmt.Errorf("配置内容无效：%w", err)
	}
	return settings, nil
}

// Save 先完整写入临时文件，再替换用户配置，避免留下半份 JSON。
func Save(settings Settings) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return saveFile(path, settings)
}

// SaveIfUnchanged 仅在磁盘配置仍等于 expected 时保存，避免多开实例静默覆盖彼此的设置。
func SaveIfUnchanged(expected, settings Settings) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return saveFileIfUnchanged(path, expected, settings)
}

func saveFile(path string, settings Settings) error {
	lock, err := acquireFileLock(path)
	if err != nil {
		return err
	}
	defer lock.Close()
	return writeFile(path, settings)
}

func saveFileIfUnchanged(path string, expected, settings Settings) error {
	lock, err := acquireFileLock(path)
	if err != nil {
		return err
	}
	defer lock.Close()

	current, err := loadFile(path)
	if err != nil {
		return err
	}
	if current != expected {
		return ErrConflict
	}
	return writeFile(path, settings)
}

func writeFile(path string, settings Settings) error {
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

// Path 返回当前用户的设置文件路径，不依赖应用安装目录。
func Path() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		return "", errors.New("无法确定用户配置目录")
	}
	return filepath.Join(base, "starline-dsh-desktop", "settings.json"), nil
}
