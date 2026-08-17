package launcher

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"starline-dsh-desktop/internal/dshversion"
)

const offlineRuntimeEnv = "DSH_DESKTOP_OFFLINE_ROOT"

type dshCommandSpec struct {
	nodePath    string
	commandPath string
	prefix      []string
	mode        string
	label       string
}

// resolveDSHCommand 只在完整离线运行时不存在时回退到系统 Node/npx。
func resolveDSHCommand(version string) (dshCommandSpec, error) {
	version, err := dshversion.Normalize(version)
	if err != nil {
		return dshCommandSpec{}, err
	}
	root, found, err := findOfflineRuntime()
	if err != nil {
		return dshCommandSpec{}, err
	}
	if found {
		return offlineDSHCommand(root, version)
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		return dshCommandSpec{}, errors.New("未找到 Node.js；请安装 Node.js 22.19+ 或 24+，或改用 offline-full 离线包")
	}
	commandPath, commandPrefix, err := npxCommand(nodePath)
	if err != nil {
		return dshCommandSpec{}, err
	}
	prefix := onlineDSHPrefix(commandPrefix, version)
	return dshCommandSpec{
		nodePath:    nodePath,
		commandPath: commandPath,
		prefix:      prefix,
		mode:        "online",
		label:       " npx",
	}, nil
}

// BundledDSHVersion 返回当前可执行文件旁离线运行时的固定版本，不存在时不报错。
func BundledDSHVersion() (string, bool, error) {
	root, found, err := findOfflineRuntime()
	if err != nil || !found {
		return "", found, err
	}
	version, err := readBundledDSHVersion(root)
	if err != nil {
		return "", true, err
	}
	return version, true, nil
}

// onlineDSHPrefix 固定 DSH 版本，并让 npm 正常重新校验版本元数据与复用内容缓存。
func onlineDSHPrefix(commandPrefix []string, version string) []string {
	return append(append([]string{}, commandPrefix...),
		"--yes",
		"--package=@deepseek-ai/dsh@"+version,
		"dsh",
	)
}

func findOfflineRuntime() (string, bool, error) {
	if configured := strings.TrimSpace(os.Getenv(offlineRuntimeEnv)); configured != "" {
		absolute, err := filepath.Abs(configured)
		if err != nil {
			return "", false, fmt.Errorf("离线运行时路径无效：%w", err)
		}
		return absolute, true, nil
	}

	executable, err := os.Executable()
	if err != nil {
		return "", false, nil
	}
	directory := filepath.Dir(executable)
	candidates := []string{filepath.Join(directory, "offline-runtime")}
	if runtime.GOOS == "darwin" {
		candidates = append(candidates, filepath.Clean(filepath.Join(directory, "..", "Resources", "offline-runtime")))
	}
	for _, candidate := range candidates {
		info, statErr := os.Stat(candidate)
		switch {
		case statErr == nil && info.IsDir():
			return candidate, true, nil
		case statErr == nil:
			return "", false, fmt.Errorf("离线运行时不是目录：%s", candidate)
		case !os.IsNotExist(statErr):
			return "", false, fmt.Errorf("无法检查离线运行时：%w", statErr)
		}
	}
	return "", false, nil
}

func offlineDSHCommand(root, requestedVersion string) (dshCommandSpec, error) {
	bundledVersion, err := readBundledDSHVersion(root)
	if err != nil {
		return dshCommandSpec{}, err
	}
	if bundledVersion != requestedVersion {
		return dshCommandSpec{}, fmt.Errorf("离线运行时 DSH 版本为 %s，但当前请求 %s；请移除版本覆盖或下载匹配的离线包", bundledVersion, requestedVersion)
	}

	nodePath := filepath.Join(root, offlineNodeName())
	dshEntry := filepath.Join(root, "node_modules", "@deepseek-ai", "dsh", "lib", "bin.js")
	for label, path := range map[string]string{"Node": nodePath, "DSH": dshEntry} {
		info, statErr := os.Stat(path)
		if statErr != nil || info.IsDir() {
			return dshCommandSpec{}, fmt.Errorf("离线运行时缺少 %s 文件：%s", label, path)
		}
	}
	return dshCommandSpec{
		nodePath:    nodePath,
		commandPath: nodePath,
		prefix:      []string{dshEntry},
		mode:        "offline",
		label:       "包内 DSH",
	}, nil
}

func readBundledDSHVersion(root string) (string, error) {
	versionPath := filepath.Join(root, "dsh-version.txt")
	versionData, err := os.ReadFile(versionPath)
	if err != nil {
		return "", fmt.Errorf("离线运行时缺少版本文件：%w", err)
	}
	version, err := dshversion.Normalize(string(versionData))
	if err != nil {
		return "", fmt.Errorf("离线运行时版本文件无效：%w", err)
	}
	return version, nil
}

func offlineNodeName() string {
	if runtime.GOOS == "windows" {
		return "node.exe"
	}
	return "node"
}

func checkNodeVersion(nodePath string) error {
	command := exec.Command(nodePath, "--version")
	configureAuxiliaryProcess(command)
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("无法读取 Node.js 版本：%w", err)
	}
	major, minor, ok := parseNodeVersion(string(output))
	if !ok || (major == 22 && minor < 19) || major < 22 || major == 23 {
		return fmt.Errorf("Node.js 版本不受支持：%s；DSH 需要 22.19+ 或 24+", strings.TrimSpace(string(output)))
	}
	return nil
}

func parseNodeVersion(raw string) (int, int, bool) {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(raw), "v"), ".")
	if len(parts) < 2 {
		return 0, 0, false
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	return major, minor, majorErr == nil && minorErr == nil
}

// npxCommand 在 Windows 上绕过 .cmd/.ps1 shim，直接让 Node 执行 npx-cli.js。
func npxCommand(nodePath string) (string, []string, error) {
	if runtime.GOOS == "windows" {
		candidate := filepath.Join(filepath.Dir(nodePath), "node_modules", "npm", "bin", "npx-cli.js")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return nodePath, []string{candidate}, nil
		}
		return "", nil, errors.New("未找到 npm 的 npx-cli.js；请重新安装包含 npm 的 Node.js")
	}
	npxPath, err := exec.LookPath("npx")
	if err != nil {
		return "", nil, errors.New("未找到 npx；请安装包含 npm 的 Node.js")
	}
	return npxPath, nil, nil
}
