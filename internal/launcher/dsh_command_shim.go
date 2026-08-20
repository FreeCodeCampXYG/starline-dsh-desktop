package launcher

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	dshShimNodePathEnv      = "STARLINE_DSH_SHIM_NODE_PATH"
	dshShimScriptPathEnv    = "STARLINE_DSH_SHIM_SCRIPT_PATH"
	dshShimCommandPathEnv   = "STARLINE_DSH_SHIM_COMMAND_PATH"
	dshShimCommandPrefixEnv = "STARLINE_DSH_SHIM_COMMAND_PREFIX"
	dshActiveProfileEnv     = "STARLINE_DSH_ACTIVE_PROFILE"
	dshWebProfile           = "web"
)

//go:embed dsh-command-shim.mjs
var dshCommandShimScript []byte

// prepareDSHCommandShim 创建当前 DSH 进程专用的命令入口，不修改系统或用户 PATH。
func prepareDSHCommandShim() (string, error) {
	directory, err := os.MkdirTemp("", "starline-dsh-command-")
	if err != nil {
		return "", fmt.Errorf("无法创建 DSH 命令兼容目录：%w", err)
	}
	writeFile := func(name string, content []byte) error {
		return os.WriteFile(filepath.Join(directory, name), content, 0o700)
	}
	if err := writeFile("dsh-command-shim.mjs", dshCommandShimScript); err != nil {
		_ = os.RemoveAll(directory)
		return "", fmt.Errorf("无法写入 DSH 命令兼容入口：%w", err)
	}
	if err := writeFile("dsh", []byte("#!/bin/sh\nexec \"$STARLINE_DSH_SHIM_NODE_PATH\" \"$STARLINE_DSH_SHIM_SCRIPT_PATH\" \"$@\"\n")); err != nil {
		_ = os.RemoveAll(directory)
		return "", fmt.Errorf("无法写入 DSH 命令兼容入口：%w", err)
	}
	if err := writeFile("dsh.cmd", []byte("@echo off\r\n\"%STARLINE_DSH_SHIM_NODE_PATH%\" \"%STARLINE_DSH_SHIM_SCRIPT_PATH%\" %*\r\nexit /b %ERRORLEVEL%\r\n")); err != nil {
		_ = os.RemoveAll(directory)
		return "", fmt.Errorf("无法写入 DSH 命令兼容入口：%w", err)
	}
	if err := writeFile("dsh.ps1", []byte("& $env:STARLINE_DSH_SHIM_NODE_PATH $env:STARLINE_DSH_SHIM_SCRIPT_PATH @args\r\nexit $LASTEXITCODE\r\n")); err != nil {
		_ = os.RemoveAll(directory)
		return "", fmt.Errorf("无法写入 DSH 命令兼容入口：%w", err)
	}
	return directory, nil
}

// withDSHCommandShim 只改写子进程环境，使 Agent 中省略 profile 的 plugin 命令默认作用于 web profile。
func withDSHCommandShim(environment []string, directory string, command dshCommandSpec) ([]string, error) {
	prefixJSON, err := json.Marshal(command.prefix)
	if err != nil {
		return nil, fmt.Errorf("无法编码 DSH 命令参数：%w", err)
	}
	ownedKeys := []string{
		"PATH",
		dshShimNodePathEnv,
		dshShimScriptPathEnv,
		dshShimCommandPathEnv,
		dshShimCommandPrefixEnv,
		dshActiveProfileEnv,
	}
	profileStoreDir := recordedProfileStoreDir(environment, dshWebProfile)
	if profileStoreDir != "" {
		ownedKeys = append(ownedKeys, "npm_config_store_dir")
	}
	filtered := make([]string, 0, len(environment)+5)
	pathValue := ""
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			filtered = append(filtered, entry)
			continue
		}
		if strings.EqualFold(key, "PATH") {
			pathValue = value
			continue
		}
		owned := false
		for _, ownedKey := range ownedKeys[1:] {
			if strings.EqualFold(key, ownedKey) {
				owned = true
				break
			}
		}
		if !owned {
			filtered = append(filtered, entry)
		}
	}
	if pathValue == "" {
		pathValue = directory
	} else {
		pathValue = directory + string(os.PathListSeparator) + pathValue
	}
	result := append(filtered,
		"PATH="+pathValue,
		dshShimNodePathEnv+"="+command.nodePath,
		dshShimScriptPathEnv+"="+filepath.Join(directory, "dsh-command-shim.mjs"),
		dshShimCommandPathEnv+"="+command.commandPath,
		dshShimCommandPrefixEnv+"="+base64.StdEncoding.EncodeToString(prefixJSON),
		dshActiveProfileEnv+"="+dshWebProfile,
	)
	if profileStoreDir != "" {
		result = append(result, "npm_config_store_dir="+profileStoreDir)
	}
	return result, nil
}

// recordedProfileStoreDir 固定现有 Profile 已使用的 pnpm Store，避免全局 pnpm 配置变化导致插件更新失败。
func recordedProfileStoreDir(environment []string, profile string) string {
	dshHome := environmentEntryValue(environment, "DSH_HOME")
	if strings.TrimSpace(dshHome) == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dshHome = filepath.Join(userHome, ".dsh")
	}
	metadataPath := filepath.Join(dshHome, "profiles", profile, "node_modules", ".modules.yaml")
	metadata, err := os.ReadFile(metadataPath)
	if err != nil {
		return ""
	}
	storeDir := parsePNPMStoreDir(metadata)
	if !filepath.IsAbs(storeDir) || strings.ContainsRune(storeDir, '\x00') {
		return ""
	}
	return filepath.Clean(storeDir)
}

func environmentEntryValue(environment []string, wanted string) string {
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(key, wanted) {
			return value
		}
	}
	return ""
}

func parsePNPMStoreDir(metadata []byte) string {
	var document struct {
		StoreDir string `json:"storeDir"`
	}
	if json.Unmarshal(metadata, &document) == nil && strings.TrimSpace(document.StoreDir) != "" {
		return strings.TrimSpace(document.StoreDir)
	}
	for _, line := range strings.Split(string(metadata), "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found || strings.TrimSpace(key) != "storeDir" {
			continue
		}
		return strings.Trim(strings.TrimSpace(value), "\"'")
	}
	return ""
}
