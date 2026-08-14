package application

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"starline-dsh-desktop/internal/config"
	"starline-dsh-desktop/internal/launcher"
)

const (
	readyEvent   = "dsh:ready"
	failedEvent  = "dsh:failed"
	stoppedEvent = "dsh:stopped"
)

type App struct {
	ctx        context.Context
	version    string
	dshVersion string

	mu              sync.Mutex
	process         *launcher.Process
	status          Status
	settings        config.Settings
	settingsLoadErr error
	generation      uint64
}

type Status struct {
	State       string `json:"state"`
	URL         string `json:"url,omitempty"`
	Message     string `json:"message"`
	Detail      string `json:"detail,omitempty"`
	Version     string `json:"version"`
	DSHVersion  string `json:"dshVersion"`
	RuntimeMode string `json:"runtimeMode"`
}

// New 组装 Wails 适配层需要的状态；DSH 版本在启动时解析一次，运行期间保持稳定。
func New(appVersion, defaultDSHVersion string) *App {
	settings, settingsErr := config.Load()
	dshVersion := resolveDSHVersion(defaultDSHVersion)
	return &App{
		version:         appVersion,
		dshVersion:      dshVersion,
		settings:        settings,
		settingsLoadErr: settingsErr,
		status: Status{
			State:       "idle",
			Message:     "等待启动 DeepSeek Harness",
			Version:     appVersion,
			DSHVersion:  dshVersion,
			RuntimeMode: "auto",
		},
	}
}

// Startup 保存 Wails 上下文，并在后台启动 DSH，避免阻塞窗口创建。
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	if a.settingsLoadErr != nil {
		a.fail("代理配置无法读取", a.settingsLoadErr)
		return
	}
	a.restart()
}

// Shutdown 确保桌面窗口退出时同步回收由它启动的 DSH 子进程。
func (a *App) Shutdown(context.Context) {
	a.mu.Lock()
	a.generation++
	process := a.process
	a.process = nil
	a.mu.Unlock()
	if process != nil {
		_ = process.Stop(5 * time.Second)
	}
}

// GetStatus 返回当前宿主状态，供前端首次挂载和重连时恢复界面。
func (a *App) GetStatus() Status {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status
}

// Retry 停止残留进程并重新启动 DSH。
func (a *App) Retry() Status {
	return a.restart()
}

// GetSettings 返回当前代理配置，不向前端暴露配置文件路径。
func (a *App) GetSettings() config.Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings
}

// SaveSettings 持久化代理配置，并立即重启由外壳管理的 DSH 进程。
func (a *App) SaveSettings(settings config.Settings) (Status, error) {
	normalized, err := config.Normalize(settings)
	if err != nil {
		return a.GetStatus(), err
	}
	if err := config.Save(normalized); err != nil {
		return a.GetStatus(), err
	}
	a.mu.Lock()
	a.settings = normalized
	a.settingsLoadErr = nil
	a.mu.Unlock()
	return a.restart(), nil
}

// OpenLogs 使用系统文件管理器打开日志所在目录。
func (a *App) OpenLogs() error {
	return launcher.OpenLogDir()
}

// OpenInBrowser 在默认浏览器中打开当前 DSH 页面，便于排查 WebView 差异。
func (a *App) OpenInBrowser() error {
	status := a.GetStatus()
	if status.URL == "" {
		return nil
	}
	runtime.BrowserOpenURL(a.ctx, status.URL)
	return nil
}

// launch 完成端口选择、子进程启动和 HTTP 就绪探测。
func (a *App) launch(generation uint64, settings config.Settings) {
	workingDir, err := os.Getwd()
	if err != nil {
		a.failIfCurrent(generation, "无法确定工作目录", err)
		return
	}

	process, err := launcher.Start(a.ctx, launcher.Config{
		Version:    a.dshVersion,
		WorkingDir: workingDir,
		ProxyMode:  settings.ProxyMode,
		ProxyURL:   settings.ProxyURL,
	})
	if err != nil {
		a.failIfCurrent(generation, "DSH 启动失败", err)
		return
	}
	runtimeMode := process.RuntimeMode()

	a.mu.Lock()
	if generation != a.generation {
		a.mu.Unlock()
		_ = process.Stop(3 * time.Second)
		return
	}
	a.process = process
	a.mu.Unlock()

	if err := process.WaitReady(a.ctx, 5*time.Minute); err != nil {
		_ = process.Stop(3 * time.Second)
		a.failIfCurrent(generation, "DSH 未能按时就绪", err)
		return
	}

	status := Status{
		State:       "ready",
		URL:         process.URL(),
		Message:     "DeepSeek Harness 已就绪",
		Version:     a.version,
		DSHVersion:  a.dshVersion,
		RuntimeMode: runtimeMode,
	}
	if !a.setStatusIfCurrent(generation, status) {
		_ = process.Stop(3 * time.Second)
		return
	}
	runtime.EventsEmit(a.ctx, readyEvent, status)

	go func() {
		err := process.Wait()
		a.mu.Lock()
		isCurrent := generation == a.generation && a.process == process
		if isCurrent {
			a.process = nil
		}
		a.mu.Unlock()
		if a.ctx.Err() != nil || !isCurrent {
			return
		}
		detail := "DSH 进程已退出"
		if err != nil {
			detail = err.Error()
		}
		stopped := Status{
			State:       "stopped",
			Message:     "DeepSeek Harness 意外停止",
			Detail:      detail,
			Version:     a.version,
			DSHVersion:  a.dshVersion,
			RuntimeMode: runtimeMode,
		}
		a.setStatus(stopped)
		runtime.EventsEmit(a.ctx, stoppedEvent, stopped)
	}()
}

// restart 切换启动代次，旧进程的迟到结果不会覆盖新一轮状态。
func (a *App) restart() Status {
	a.mu.Lock()
	a.generation++
	generation := a.generation
	process := a.process
	a.process = nil
	settings := a.settings
	a.status = Status{
		State:       "starting",
		Message:     "正在启动 DeepSeek Harness…",
		Version:     a.version,
		DSHVersion:  a.dshVersion,
		RuntimeMode: "auto",
	}
	status := a.status
	a.mu.Unlock()

	if process != nil {
		_ = process.Stop(3 * time.Second)
	}
	go a.launch(generation, settings)
	return status
}

func (a *App) failIfCurrent(generation uint64, message string, err error) {
	status := Status{
		State:       "failed",
		Message:     message,
		Detail:      err.Error(),
		Version:     a.version,
		DSHVersion:  a.dshVersion,
		RuntimeMode: "auto",
	}
	if a.setStatusIfCurrent(generation, status) && a.ctx != nil {
		runtime.EventsEmit(a.ctx, failedEvent, status)
	}
}

func (a *App) fail(message string, err error) {
	status := Status{
		State:       "failed",
		Message:     message,
		Detail:      err.Error(),
		Version:     a.version,
		DSHVersion:  a.dshVersion,
		RuntimeMode: "auto",
	}
	a.setStatus(status)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, failedEvent, status)
	}
}

func (a *App) setStatus(status Status) {
	a.mu.Lock()
	a.status = status
	a.mu.Unlock()
}

func (a *App) setStatusIfCurrent(generation uint64, status Status) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	if generation != a.generation {
		return false
	}
	a.status = status
	return true
}

func (a *App) emit(event string) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, event)
	}
}

func resolveDSHVersion(fallback string) string {
	if value := strings.TrimSpace(os.Getenv("DSH_DESKTOP_DSH_VERSION")); value != "" {
		return value
	}
	return fallback
}
