package launcher

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxLogFiles = 10

// LogDir 返回用户级日志目录，不把运行日志写入当前项目或应用安装目录。
func LogDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "starline-dsh-desktop", "logs")
}

func npmLogDir() string {
	if cache := strings.TrimSpace(os.Getenv("npm_config_cache")); cache != "" {
		return filepath.Join(cache, "_logs")
	}
	base, err := os.UserCacheDir()
	if err == nil && runtime.GOOS == "windows" {
		return filepath.Join(base, "npm-cache", "_logs")
	}
	if home, homeErr := os.UserHomeDir(); homeErr == nil {
		return filepath.Join(home, ".npm", "_logs")
	}
	return "npm cache/_logs"
}

// OpenLogDir 使用系统文件管理器打开日志目录。
func OpenLogDir() error {
	directory := LogDir()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("explorer.exe", directory)
	case "darwin":
		command = exec.Command("open", directory)
	default:
		command = exec.Command("xdg-open", directory)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("无法打开日志目录：%w", err)
	}
	return nil
}

func createLogFile() (string, *os.File, error) {
	directory := LogDir()
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", nil, err
	}
	_ = pruneLogs(directory)
	path := filepath.Join(directory, "dsh-"+time.Now().Format("20060102-150405.000")+".log")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	return path, file, err
}

func pruneLogs(directory string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	type logEntry struct {
		path string
		time time.Time
	}
	logs := make([]logEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".log") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr == nil {
			logs = append(logs, logEntry{filepath.Join(directory, entry.Name()), info.ModTime()})
		}
	}
	sort.Slice(logs, func(i, j int) bool { return logs[i].time.After(logs[j].time) })
	if len(logs) > maxLogFiles {
		for _, entry := range logs[maxLogFiles:] {
			_ = os.Remove(entry.path)
		}
	}
	return nil
}

type lineWriter struct {
	mu     sync.Mutex
	buffer strings.Builder
	onLine func(string)
}

func (w *lineWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = w.buffer.Write(data)
	scanner := bufio.NewScanner(strings.NewReader(w.buffer.String()))
	lines := make([]string, 0, 4)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) == 0 {
		return len(data), nil
	}
	complete := strings.HasSuffix(w.buffer.String(), "\n")
	limit := len(lines)
	if !complete {
		limit--
	}
	for _, line := range lines[:limit] {
		w.onLine(line)
	}
	w.buffer.Reset()
	if !complete {
		_, _ = w.buffer.WriteString(lines[len(lines)-1])
	}
	return len(data), nil
}

func (w *lineWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buffer.Len() > 0 {
		w.onLine(w.buffer.String())
		w.buffer.Reset()
	}
	return nil
}
