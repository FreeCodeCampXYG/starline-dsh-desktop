package launcher

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maxLogFiles = 10

var webURLPattern = regexp.MustCompile(`dsh web:\s+(http://[^\s]+)`)

type Config struct {
	Version    string
	WorkingDir string
	ProxyMode  string
	ProxyURL   string
}

type Process struct {
	cmd     *exec.Cmd
	logPath string

	mu       sync.RWMutex
	url      string
	urlReady chan struct{}
	urlOnce  sync.Once
	done     chan struct{}
	waitErr  error
	stopOnce sync.Once
}

// Start 校验本机 Node 环境，并通过固定版本的 npm 包启动 DSH Web。
func Start(_ context.Context, config Config) (*Process, error) {
	if strings.TrimSpace(config.Version) == "" {
		return nil, errors.New("未指定 DSH 版本")
	}
	if info, err := os.Stat(config.WorkingDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("工作目录不可用：%s", config.WorkingDir)
	}

	nodePath, err := exec.LookPath("node")
	if err != nil {
		return nil, errors.New("未找到 Node.js；DSH 需要 Node.js 22.19+ 或 24+")
	}
	if err := checkNodeVersion(nodePath); err != nil {
		return nil, err
	}
	commandPath, commandPrefix, err := npxCommand(nodePath)
	if err != nil {
		return nil, err
	}

	logPath, logFile, err := createLogFile()
	if err != nil {
		return nil, fmt.Errorf("无法创建日志文件：%w", err)
	}

	process := &Process{
		logPath:  logPath,
		urlReady: make(chan struct{}),
		done:     make(chan struct{}),
	}
	port, err := availableLoopbackPort()
	if err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("无法选择本地监听端口：%w", err)
	}
	process.url = "http://127.0.0.1:" + strconv.Itoa(port)
	process.urlOnce.Do(func() { close(process.urlReady) })
	lineSink := &lineWriter{onLine: process.inspectLine}
	writer := io.MultiWriter(logFile, lineSink)
	_, _ = fmt.Fprintf(
		logFile,
		"[%s] Starline DSH Desktop 正在通过 npx 准备 @deepseek-ai/dsh@%s。首次下载可能需要数分钟。\n",
		time.Now().Format(time.RFC3339),
		config.Version,
	)
	_, _ = fmt.Fprintf(logFile, "npm 调试日志目录：%s\n", npmLogDir())
	args := append(commandPrefix, []string{
		"--yes",
		"--package=@deepseek-ai/dsh@" + config.Version,
		"dsh",
		"web",
		"--host",
		"127.0.0.1",
		"--port",
		strconv.Itoa(port),
	}...)
	cmd := exec.Command(commandPath, args...)
	cmd.Dir = config.WorkingDir
	cmd.Env = childEnvironment(os.Environ(), config.ProxyMode, config.ProxyURL)
	cmd.Stdout = writer
	cmd.Stderr = writer
	configureProcess(cmd)
	process.cmd = cmd

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("无法启动 npx：%w（日志：%s）", err, logPath)
	}

	go func() {
		process.waitErr = cmd.Wait()
		_ = lineSink.Flush()
		_ = logFile.Close()
		close(process.done)
	}()
	return process, nil
}

// WaitReady 使用不经过代理的 HTTP 请求，并校验页面指纹以确认 DSH 已就绪。
func (p *Process) WaitReady(ctx context.Context, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-p.urlReady:
	case <-p.done:
		return p.exitError("DSH 在公布监听地址前退出")
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("等待监听地址超时（日志：%s）", p.logPath)
	}

	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.URL(), nil)
		if err == nil {
			response, requestErr := client.Do(request)
			if requestErr == nil {
				body, readErr := io.ReadAll(io.LimitReader(response.Body, 128*1024))
				_ = response.Body.Close()
				if readErr == nil && response.StatusCode == http.StatusOK && strings.Contains(string(body), "<title>DeepSeek Harness</title>") {
					return nil
				}
			}
		}
		select {
		case <-p.done:
			return p.exitError("DSH 在健康检查期间退出")
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return fmt.Errorf("HTTP 健康检查超时（日志：%s）", p.logPath)
		case <-ticker.C:
		}
	}
}

// Stop 只终止当前宿主启动的进程树，不扫描或影响其他 DSH 实例。
func (p *Process) Stop(timeout time.Duration) error {
	var stopErr error
	p.stopOnce.Do(func() {
		if p.cmd == nil || p.cmd.Process == nil {
			return
		}
		select {
		case <-p.done:
			return
		default:
		}

		stopErr = stopProcessTree(p.cmd.Process.Pid)
		if stopErr != nil {
			_ = p.cmd.Process.Kill()
		}
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case <-p.done:
		case <-timer.C:
			_ = p.cmd.Process.Kill()
			<-p.done
		}
	})
	return stopErr
}

func (p *Process) Wait() error {
	<-p.done
	return p.waitErr
}

func (p *Process) URL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.url
}

func (p *Process) inspectLine(line string) {
	match := webURLPattern.FindStringSubmatch(line)
	if len(match) != 2 || !safeLoopbackURL(match[1]) {
		return
	}
	p.mu.Lock()
	p.url = strings.TrimRight(match[1], "/")
	p.mu.Unlock()
	p.urlOnce.Do(func() { close(p.urlReady) })
}

func (p *Process) exitError(prefix string) error {
	if p.waitErr != nil {
		return fmt.Errorf("%s：%v（日志：%s）", prefix, p.waitErr, p.logPath)
	}
	return fmt.Errorf("%s（日志：%s）", prefix, p.logPath)
}

func safeLoopbackURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "127.0.0.1" || host == "localhost" || host == "::1"
}

// availableLoopbackPort 让操作系统选择高位端口；关闭监听后立即交给 DSH 使用。
func availableLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok || address.Port == 0 {
		return 0, errors.New("操作系统没有返回有效端口")
	}
	return address.Port, nil
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

func childEnvironment(environment []string, proxyMode, proxyURL string) []string {
	if proxyMode == "custom" || proxyMode == "disabled" {
		environment = withoutProxyVariables(environment)
	}
	if proxyMode == "custom" {
		environment = append(
			environment,
			"HTTP_PROXY="+proxyURL,
			"HTTPS_PROXY="+proxyURL,
			"http_proxy="+proxyURL,
			"https_proxy="+proxyURL,
			"npm_config_proxy="+proxyURL,
			"npm_config_https_proxy="+proxyURL,
		)
	}
	return mergeNoProxy(environment, []string{"127.0.0.1", "localhost", "::1"})
}

func withoutProxyVariables(environment []string) []string {
	proxyKeys := map[string]bool{
		"http_proxy":             true,
		"https_proxy":            true,
		"all_proxy":              true,
		"npm_config_proxy":       true,
		"npm_config_https_proxy": true,
		"npm_config_noproxy":     true,
	}
	result := make([]string, 0, len(environment))
	for _, entry := range environment {
		key, _, found := strings.Cut(entry, "=")
		if found && proxyKeys[strings.ToLower(key)] {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func mergeNoProxy(environment []string, required []string) []string {
	result := make([]string, 0, len(environment)+1)
	values := make([]string, 0, len(required)+4)
	seen := make(map[string]bool)
	for _, entry := range environment {
		key, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(key, "NO_PROXY") {
			for _, item := range strings.Split(value, ",") {
				trimmed := strings.TrimSpace(item)
				if trimmed != "" && !seen[strings.ToLower(trimmed)] {
					values = append(values, trimmed)
					seen[strings.ToLower(trimmed)] = true
				}
			}
			continue
		}
		result = append(result, entry)
	}
	for _, item := range required {
		if !seen[strings.ToLower(item)] {
			values = append(values, item)
			seen[strings.ToLower(item)] = true
		}
	}
	return append(result, "NO_PROXY="+strings.Join(values, ","))
}

// LogDir 返回用户级日志目录，不把运行日志写入当前项目。
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
