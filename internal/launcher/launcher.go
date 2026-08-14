package launcher

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var webURLPattern = regexp.MustCompile(`dsh web:\s+(http://[^\s]+)`)

type Config struct {
	Version    string
	WorkingDir string
	ProxyMode  string
	ProxyURL   string
}

type Process struct {
	cmd         *exec.Cmd
	logPath     string
	runtimeMode string

	mu       sync.RWMutex
	url      string
	urlReady chan struct{}
	urlOnce  sync.Once
	done     chan struct{}
	waitErr  error
	stopOnce sync.Once
}

// Start 优先使用包内离线运行时，否则通过系统 Node 和固定版本 npm 包启动 DSH Web。
func Start(_ context.Context, config Config) (*Process, error) {
	if strings.TrimSpace(config.Version) == "" {
		return nil, errors.New("未指定 DSH 版本")
	}
	if info, err := os.Stat(config.WorkingDir); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("工作目录不可用：%s", config.WorkingDir)
	}

	command, err := resolveDSHCommand(config.Version)
	if err != nil {
		return nil, err
	}
	if err := checkNodeVersion(command.nodePath); err != nil {
		return nil, err
	}

	logPath, logFile, err := createLogFile()
	if err != nil {
		return nil, fmt.Errorf("无法创建日志文件：%w", err)
	}

	process := &Process{
		logPath:     logPath,
		runtimeMode: command.mode,
		urlReady:    make(chan struct{}),
		done:        make(chan struct{}),
	}
	lineSink := &lineWriter{onLine: process.inspectLine}
	writer := io.MultiWriter(logFile, lineSink)
	if command.mode == "offline" {
		_, _ = fmt.Fprintf(logFile, "[%s] Starline DSH Desktop 正在使用包内离线运行时启动 @deepseek-ai/dsh@%s。\n", time.Now().Format(time.RFC3339), config.Version)
	} else {
		_, _ = fmt.Fprintf(logFile, "[%s] Starline DSH Desktop 正在通过 npx 准备 @deepseek-ai/dsh@%s；优先复用 npm 缓存，缓存缺失时才下载。\n", time.Now().Format(time.RFC3339), config.Version)
		_, _ = fmt.Fprintf(logFile, "npm 调试日志目录：%s\n", npmLogDir())
	}
	args := append(append([]string{}, command.prefix...), []string{
		"web",
		"--host",
		"127.0.0.1",
		"--port",
		"0",
	}...)
	cmd := exec.Command(command.commandPath, args...)
	cmd.Dir = config.WorkingDir
	cmd.Env = childEnvironment(os.Environ(), config.ProxyMode, config.ProxyURL)
	cmd.Stdout = writer
	cmd.Stderr = writer
	configureProcess(cmd)
	process.cmd = cmd

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("无法启动%s：%w（日志：%s）", command.label, err, logPath)
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
	p.urlOnce.Do(func() { close(p.urlReady) })
}

func (p *Process) exitError(prefix string) error {
	if p.waitErr != nil {
		return fmt.Errorf("%s：%v（日志：%s）", prefix, p.waitErr, p.logPath)
	}
	return fmt.Errorf("%s（日志：%s）", prefix, p.logPath)
}
