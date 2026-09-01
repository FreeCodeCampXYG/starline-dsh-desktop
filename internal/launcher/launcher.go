package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	webURLPattern      = regexp.MustCompile(`dsh web:\s+(http://[^\s]+)`)
	dshWebTitlePattern = regexp.MustCompile(`<title>(?:DeepSeek Harness|DSH Local Build)</title>`)
)

const onlineStartupTimeout = 5 * time.Minute

type Config struct {
	Version    string
	WorkingDir string
	ProxyMode  string
	ProxyURL   string
	OnProgress func(Progress)
}

// Progress 表示可验证的启动阶段；百分比是阶段权重，不伪装成 npm 下载字节进度。
type Progress struct {
	Percent     int
	Stage       string
	RuntimeMode string
}

type Process struct {
	cmd         *exec.Cmd
	logPath     string
	runtimeMode string
	shimDir     string

	mu       sync.RWMutex
	url      string
	urlReady chan struct{}
	urlOnce  sync.Once
	done     chan struct{}
	waitErr  error
	stopOnce sync.Once

	progressMu sync.Mutex
	progress   int
	onProgress func(Progress)
}

// OnlineStartupTimeout 返回在线 npx 运行时的整体启动上限；零值使用宿主默认值。
func OnlineStartupTimeout(seconds int) time.Duration {
	if seconds <= 0 {
		return onlineStartupTimeout
	}
	return time.Duration(seconds) * time.Second
}

// Start 优先使用包内离线运行时，否则通过系统 Node 和固定版本 npm 包启动 DSH Web。
func Start(_ context.Context, config Config) (*Process, error) {
	if strings.TrimSpace(config.Version) == "" {
		return nil, errors.New("未指定 DSH 版本")
	}
	if info, err := os.Stat(config.WorkingDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("工作目录不可用：%s", config.WorkingDir)
	}
	emitProgress(config.OnProgress, 30, "正在检测包内运行时和系统 Node…", "auto")

	command, err := resolveDSHCommand(config.Version)
	if err != nil {
		return nil, err
	}
	modeStage := "已选择系统 Node / npx 在线运行时"
	if command.mode == "offline" {
		modeStage = "已找到并验证包内离线运行时"
	}
	emitProgress(config.OnProgress, 40, modeStage, command.mode)
	if err := checkNodeVersion(command.nodePath); err != nil {
		return nil, err
	}
	emitProgress(config.OnProgress, 48, "Node.js 版本检查通过", command.mode)

	logPath, logFile, err := createLogFile()
	if err != nil {
		return nil, fmt.Errorf("无法创建日志文件：%w", err)
	}
	shimDir, err := prepareDSHCommandShim()
	if err != nil {
		_ = logFile.Close()
		return nil, err
	}
	childBaseEnv, proxyFallbackRoute := childEnvironmentWithFallback(os.Environ(), config.ProxyMode, config.ProxyURL)
	selectedRegistry := npmMirrorRegistry
	if command.mode == "online" {
		for _, candidate := range npmRegistryCandidates() {
			if npmRegistryReachable(candidate, childBaseEnv) {
				selectedRegistry = candidate
				break
			}
		}
	}
	childBaseEnv = withNPMRegistry(childBaseEnv, selectedRegistry)
	childEnv, err := withDSHCommandShim(childBaseEnv, shimDir, command)
	if err != nil {
		_ = os.RemoveAll(shimDir)
		_ = logFile.Close()
		return nil, err
	}
	emitProgress(config.OnProgress, 55, "正在准备日志、代理和命令兼容入口…", command.mode)

	process := &Process{
		logPath:     logPath,
		runtimeMode: command.mode,
		shimDir:     shimDir,
		urlReady:    make(chan struct{}),
		done:        make(chan struct{}),
		progress:    55,
		onProgress:  config.OnProgress,
	}
	lineSink := &lineWriter{onLine: process.inspectLine}
	writer := io.MultiWriter(logFile, lineSink)
	if command.mode == "offline" {
		_, _ = fmt.Fprintf(logFile, "[%s] Starline DSH Desktop 正在使用包内离线运行时启动 @deepseek-ai/dsh@%s。\n", time.Now().Format(time.RFC3339), config.Version)
	} else {
		_, _ = fmt.Fprintf(logFile, "[%s] Starline DSH Desktop 正在通过 npx 准备 @deepseek-ai/dsh@%s；npm 会校验版本元数据并复用可用内容缓存。\n", time.Now().Format(time.RFC3339), config.Version)
		_, _ = fmt.Fprintf(logFile, "npm 调试日志目录：%s\n", npmLogDir())
		_, _ = fmt.Fprintf(logFile, "npm registry：%s（官方优先，失败回退国内镜像）；单次网络等待：%sms；重试次数：%s。\n", npmRegistry(childEnv), npmFetchTimeout, npmFetchRetries)
		if npmRegistry(childEnv) == npmMirrorRegistry {
			_, _ = fmt.Fprintln(logFile, "官方 npm registry 探测失败，已回退国内镜像；如本机代理监听 1080，请在代理设置中填写 http://127.0.0.1:1080 后重试。")
		}
	}
	switch proxyFallbackRoute {
	case proxyFallbackSystem:
		_, _ = fmt.Fprintln(logFile, "自定义代理不可达，已改用系统环境代理和国内 npm 镜像。")
		emitProgress(config.OnProgress, 56, "自定义代理不可达，已改用系统环境代理", command.mode)
	case proxyFallbackDirect:
		_, _ = fmt.Fprintln(logFile, "代理不可达，已切换为国内 npm 镜像直连；DSH 模型/API 请求如需代理，请恢复可用代理后重启。")
		emitProgress(config.OnProgress, 56, "代理不可达，已切换国内镜像直连（模型/API 可能仍需代理）", command.mode)
	}
	_, _ = fmt.Fprintln(logFile, "DSH 命令兼容入口：Agent 执行 dsh plugin 且未指定 --profile 时，默认使用当前 web profile。")
	args := append(append([]string{}, command.prefix...), dshWebArgs()...)
	cmd := exec.Command(command.commandPath, args...)
	cmd.Dir = config.WorkingDir
	cmd.Env = childEnv
	cmd.Stdout = writer
	cmd.Stderr = writer
	configureProcess(cmd)
	process.cmd = cmd

	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(shimDir)
		_ = logFile.Close()
		return nil, fmt.Errorf("无法启动%s：%w（日志：%s）", command.label, err, logPath)
	}
	stage := "包内 DSH 进程已启动，正在等待监听地址…"
	if command.mode == "online" {
		stage = "npx 已启动，正在校验元数据并准备 DSH 依赖…"
	}
	process.reportProgress(65, stage)

	go func() {
		process.waitErr = cmd.Wait()
		_ = lineSink.Flush()
		_ = logFile.Close()
		_ = os.RemoveAll(process.shimDir)
		close(process.done)
	}()
	return process, nil
}

// dshWebArgs 使用上游 alpha.3 的 Web profile 入口；不能继续把旧的 dsh web 子命令当作兼容契约。
// 内嵌 iframe 已负责展示页面，因此禁止上游把同一 URL 交给系统浏览器。
func dshWebArgs() []string {
	return []string{
		"--profile",
		"web",
		"--host",
		"127.0.0.1",
		"--port",
		"0",
		"--no-open",
	}
}

// WaitReady 使用不经过代理的 HTTP 请求，并校验页面指纹以确认 DSH 已就绪。
func (p *Process) WaitReady(ctx context.Context, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-p.urlReady:
		p.reportProgress(92, "已获得监听地址，正在校验本地 DSH 页面…")
	case <-p.done:
		return p.exitError("DSH 在公布监听地址前退出")
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		if p.runtimeMode == "online" {
			// 在线包首次准备可能先下载完整依赖树；超时信息必须指向 npx/网络边界，避免误判为已启动服务的端口故障。
			return fmt.Errorf("在线 npx 在 %s 内未完成依赖准备或公布监听地址；请检查 npm registry/代理，或改用 offline-full 离线包（日志：%s）", timeout, p.logPath)
		}
		return fmt.Errorf("等待监听地址超时（日志：%s）", p.logPath)
	}

	// alpha.3 的 token URL 只能消费一次，健康检查不能先替 WebView 换 cookie，
	// 否则 iframe 再访问原 URL 会收到 authentication required。检查无查询参数的根地址，
	// 将一次性认证 URL 原样留给内嵌页面完成握手。
	pageURL, err := url.Parse(p.URL())
	if err != nil {
		return fmt.Errorf("DSH 监听地址无效：%w", err)
	}
	pageURL.RawQuery = ""
	pageURL.Fragment = ""
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL.String(), nil)
		if err == nil {
			response, requestErr := client.Do(request)
			if requestErr == nil {
				body, readErr := io.ReadAll(io.LimitReader(response.Body, 128*1024))
				_ = response.Body.Close()
				// 未认证根页可能返回 401；这同样证明服务已监听，token 认证交给 WebView。
				if readErr == nil && (response.StatusCode == http.StatusUnauthorized ||
					(response.StatusCode == http.StatusOK && dshWebTitlePattern.Match(body))) {
					p.reportProgress(98, "本地 DSH 页面校验通过")
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

// RuntimeMode 返回当前进程使用 online 还是 offline 运行时。
func (p *Process) RuntimeMode() string {
	return p.runtimeMode
}

func (p *Process) inspectLine(line string) {
	match := webURLPattern.FindStringSubmatch(line)
	if len(match) != 2 || !safeLoopbackURL(match[1]) {
		return
	}
	p.mu.Lock()
	p.url = strings.TrimRight(match[1], "/")
	p.mu.Unlock()
	p.urlOnce.Do(func() {
		p.reportProgress(85, "DSH 已公布安全的本地监听地址")
		close(p.urlReady)
	})
}

func emitProgress(callback func(Progress), percent int, stage, runtimeMode string) {
	if callback == nil {
		return
	}
	callback(Progress{Percent: percent, Stage: stage, RuntimeMode: runtimeMode})
}

// reportProgress 只允许当前进程的阶段百分比单调前进，避免并发输出造成界面倒退。
func (p *Process) reportProgress(percent int, stage string) {
	p.progressMu.Lock()
	if percent <= p.progress {
		p.progressMu.Unlock()
		return
	}
	p.progress = percent
	callback := p.onProgress
	runtimeMode := p.runtimeMode
	p.progressMu.Unlock()
	emitProgress(callback, percent, stage, runtimeMode)
}

func (p *Process) exitError(prefix string) error {
	if p.waitErr != nil {
		return fmt.Errorf("%s：%v（日志：%s）", prefix, p.waitErr, p.logPath)
	}
	return fmt.Errorf("%s（日志：%s）", prefix, p.logPath)
}
