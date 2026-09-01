package application

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"starline-dsh-desktop/internal/config"
	"starline-dsh-desktop/internal/dshversion"
	"starline-dsh-desktop/internal/launcher"
	"starline-dsh-desktop/internal/updater"
)

const (
	readyEvent        = "dsh:ready"
	progressEvent     = "dsh:progress"
	failedEvent       = "dsh:failed"
	stoppedEvent      = "dsh:stopped"
	windowRevealDelay = 800 * time.Millisecond
	projectURL        = "https://github.com/FreeCodeCampXYG/starline-dsh-desktop"
	desktopReleaseURL = projectURL + "/releases/latest"
)

type App struct {
	ctx               context.Context
	version           string
	dshVersion        string
	defaultDSHVersion string

	mu              sync.Mutex
	settingsMu      sync.Mutex
	process         *launcher.Process
	status          Status
	settings        config.Settings
	settingsLoadErr error
	generation      uint64
	quitRequested   bool
	pendingRollback *dshUpdateRollback
}

type dshUpdateRollback struct {
	generation        uint64
	previousSettings  config.Settings
	previousVersion   string
	attemptedSettings config.Settings
}

type DSHUpdateInfo struct {
	CurrentVersion        string `json:"currentVersion"`
	DefaultVersion        string `json:"defaultVersion"`
	LatestVersion         string `json:"latestVersion"`
	NextVersion           string `json:"nextVersion,omitempty"`
	RuntimeMode           string `json:"runtimeMode"`
	Message               string `json:"message"`
	LatestUpdateAvailable bool   `json:"latestUpdateAvailable"`
	NextUpdateAvailable   bool   `json:"nextUpdateAvailable"`
	CanApply              bool   `json:"canApply"`
	CanReset              bool   `json:"canReset"`
	UsingCustomVersion    bool   `json:"usingCustomVersion"`
}

type Status struct {
	State       string `json:"state"`
	URL         string `json:"url,omitempty"`
	Message     string `json:"message"`
	Detail      string `json:"detail,omitempty"`
	Version     string `json:"version"`
	DSHVersion  string `json:"dshVersion"`
	RuntimeMode string `json:"runtimeMode"`
	Progress    int    `json:"progress,omitempty"`
	Stage       string `json:"stage,omitempty"`
}

// New 组装 Wails 适配层需要的状态；DSH 版本在启动时解析一次，运行期间保持稳定。
func New(appVersion, defaultDSHVersion string) *App {
	settings, settingsErr := config.Load()
	dshVersion := strings.TrimSpace(defaultDSHVersion)
	resolvedVersion, versionErr := resolveDSHVersion(defaultDSHVersion, settings)
	if versionErr == nil {
		dshVersion = resolvedVersion
	} else if settingsErr == nil {
		settingsErr = versionErr
	}
	return &App{
		version:           appVersion,
		dshVersion:        dshVersion,
		defaultDSHVersion: strings.TrimSpace(defaultDSHVersion),
		settings:          settings,
		settingsLoadErr:   settingsErr,
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
	a.mu.Lock()
	a.ctx = ctx
	a.quitRequested = false
	a.mu.Unlock()
	// 允许前端先在隐藏窗口中完成一次预热；即使前端异常，也会在短暂等待后显示窗口。
	go func() {
		timer := time.NewTimer(windowRevealDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			runtime.WindowShow(ctx)
		}
	}()
	if a.settingsLoadErr != nil {
		a.fail("启动配置无效", a.settingsLoadErr)
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

// BeforeClose 阻止窗口关闭事件直接结束进程；托盘菜单的显式退出会先放行。
func (a *App) BeforeClose(context.Context) bool {
	if allowWindowCloseWithoutTray() {
		return false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return !a.quitRequested
}

// ShowWindow 从托盘恢复主窗口。
func (a *App) ShowWindow() {
	a.mu.Lock()
	ctx := a.ctx
	a.mu.Unlock()
	if ctx != nil {
		runtime.WindowShow(ctx)
	}
}

// RequestQuit 记录显式退出意图，并让 Wails 走标准 shutdown 回收流程。
func (a *App) RequestQuit() {
	a.mu.Lock()
	a.quitRequested = true
	ctx := a.ctx
	a.mu.Unlock()
	if ctx != nil {
		runtime.Quit(ctx)
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
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	a.mu.Lock()
	expected := a.settings
	a.mu.Unlock()
	settings.DSHVersion = expected.DSHVersion
	normalized, err := config.Normalize(settings)
	if err != nil {
		return a.GetStatus(), err
	}
	if err := config.SaveIfUnchanged(expected, normalized); err != nil {
		return a.GetStatus(), err
	}
	a.mu.Lock()
	a.settings = normalized
	a.settingsLoadErr = nil
	a.mu.Unlock()
	return a.restart(), nil
}

// CheckDSHUpdate 自动通过国内镜像直连查询 latest/next，只提示而不修改状态。
func (a *App) CheckDSHUpdate() (DSHUpdateInfo, error) {
	return a.checkDSHUpdate(false)
}

// CheckDSHUpdateManual 由用户主动触发，按当前代理设置查询受信任的 npm registry。
func (a *App) CheckDSHUpdateManual() (DSHUpdateInfo, error) {
	return a.checkDSHUpdate(true)
}

func (a *App) checkDSHUpdate(manual bool) (DSHUpdateInfo, error) {
	a.mu.Lock()
	ctx := a.ctx
	currentVersion := a.dshVersion
	defaultVersion := a.defaultDSHVersion
	settings := a.settings
	runtimeMode := a.status.RuntimeMode
	a.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	var release updater.DSHRelease
	var err error
	if manual {
		release, err = updater.CheckDSHChannels(ctx, currentVersion, settings)
	} else {
		release, err = updater.CheckDSHChannelsAutomatic(ctx, currentVersion)
	}
	if err != nil {
		return DSHUpdateInfo{}, err
	}
	canApply, reason, err := canApplyDSHUpdate()
	if err != nil {
		return DSHUpdateInfo{}, err
	}
	info := DSHUpdateInfo{
		CurrentVersion:        currentVersion,
		DefaultVersion:        defaultVersion,
		LatestVersion:         release.LatestVersion,
		NextVersion:           release.NextVersion,
		RuntimeMode:           runtimeMode,
		LatestUpdateAvailable: release.LatestUpdateAvailable,
		NextUpdateAvailable:   release.NextUpdateAvailable,
		CanApply:              canApply,
		CanReset:              settings.DSHVersion != "",
		UsingCustomVersion:    settings.DSHVersion != "",
	}
	switch {
	case !canApply:
		info.Message = reason
	case release.LatestUpdateAvailable:
		info.Message = "发现新的 npm latest；确认后会保存精确版本、回收当前 DSH 子进程树并重启。"
	case release.NextUpdateAvailable:
		info.Message = "当前已跟上 npm latest；next 预览通道有更新，可自行确认试用。"
	case release.CurrentNewerThanLatest:
		info.Message = "当前 DSH 版本高于 npm latest，不会自动降级。"
	case release.NextVersion == "":
		info.Message = "当前已是 npm latest；官方暂未发布 next 标签。"
	default:
		info.Message = "当前 DSH 已跟上 npm latest 与 next。"
	}
	return info, nil
}

// ApplyDSHUpdate 再次核对指定通道，保存精确版本后回收旧子进程并重启在线运行时。
func (a *App) ApplyDSHUpdate(channel string) (Status, error) {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	a.mu.Lock()
	ctx := a.ctx
	currentVersion := a.dshVersion
	expected := a.settings
	a.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	canApply, reason, err := canApplyDSHUpdate()
	if err != nil {
		return a.GetStatus(), err
	}
	if !canApply {
		return a.GetStatus(), errors.New(reason)
	}
	release, err := updater.CheckDSHChannels(ctx, currentVersion, expected)
	if err != nil {
		return a.GetStatus(), err
	}
	targetVersion, err := selectDSHUpdateTarget(release, channel)
	if err != nil {
		return a.GetStatus(), err
	}
	updated := expected
	updated.DSHVersion = targetVersion
	updated, err = config.Normalize(updated)
	if err != nil {
		return a.GetStatus(), err
	}
	if err := config.SaveIfUnchanged(expected, updated); err != nil {
		return a.GetStatus(), err
	}
	a.mu.Lock()
	a.settings = updated
	a.settingsLoadErr = nil
	a.dshVersion = targetVersion
	a.mu.Unlock()
	return a.restartWithProgressAndRollback(
		"正在更新 DeepSeek Harness…",
		"已保存官方 "+strings.ToLower(strings.TrimSpace(channel))+" 精确版本 "+targetVersion+"，正在切换运行时…",
		20,
		&dshUpdateRollback{
			previousSettings:  expected,
			previousVersion:   currentVersion,
			attemptedSettings: updated,
		},
	), nil
}

// selectDSHUpdateTarget 只接受已知 npm 通道，并禁止无意义降级或重复重启。
func selectDSHUpdateTarget(release updater.DSHRelease, channel string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(channel)) {
	case "latest":
		if !release.LatestUpdateAvailable {
			return "", errors.New("当前版本不低于 npm 官方 latest，无需更新")
		}
		return release.LatestVersion, nil
	case "next":
		if release.NextVersion == "" {
			return "", errors.New("npm 官方当前没有 next 版本")
		}
		if !release.NextUpdateAvailable {
			return "", errors.New("当前版本不低于 npm 官方 next，无需更新")
		}
		return release.NextVersion, nil
	default:
		return "", errors.New("未知的 DSH 更新通道，只支持 latest 或 next")
	}
}

// ResetDSHVersion 清除手动选择，恢复桌面版本内置的兼容 DSH 版本。
func (a *App) ResetDSHVersion() (Status, error) {
	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()

	a.mu.Lock()
	expected := a.settings
	a.mu.Unlock()
	if expected.DSHVersion == "" {
		return a.GetStatus(), errors.New("当前没有保存手动 DSH 版本")
	}
	updated := expected
	updated.DSHVersion = ""
	resolvedVersion, err := resolveDSHVersion(a.defaultDSHVersion, updated)
	if err != nil {
		return a.GetStatus(), err
	}
	if err := config.SaveIfUnchanged(expected, updated); err != nil {
		return a.GetStatus(), err
	}
	a.mu.Lock()
	a.settings = updated
	a.settingsLoadErr = nil
	a.dshVersion = resolvedVersion
	a.mu.Unlock()
	return a.restartWithProgress("正在恢复默认 DSH 版本…", "已恢复 Desktop 内置兼容版本，正在切换运行时…", 20), nil
}

// OpenLogs 使用系统文件管理器打开日志所在目录。
func (a *App) OpenLogs() error {
	return launcher.OpenLogDir()
}

// OpenProjectPage 在系统浏览器中打开项目主页。
func (a *App) OpenProjectPage() error {
	runtime.BrowserOpenURL(a.ctx, projectURL)
	return nil
}

// OpenDesktopReleasePage 供用户手工检查 Desktop 版本和下载对应平台资产。
func (a *App) OpenDesktopReleasePage() error {
	runtime.BrowserOpenURL(a.ctx, desktopReleaseURL)
	return nil
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
func (a *App) launch(generation uint64, settings config.Settings, dshVersion string) {
	workingDir, err := os.Getwd()
	if err != nil {
		a.failIfCurrent(generation, dshVersion, "无法确定工作目录", err)
		return
	}

	process, err := launcher.Start(a.ctx, launcher.Config{
		Version:    dshVersion,
		WorkingDir: workingDir,
		ProxyMode:  settings.ProxyMode,
		ProxyURL:   settings.ProxyURL,
		OnProgress: func(progress launcher.Progress) {
			a.progressIfCurrent(generation, dshVersion, progress)
		},
	})
	if err != nil {
		a.failIfCurrent(generation, dshVersion, "DSH 启动失败", err)
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

	readyTimeout := 5 * time.Minute
	if runtimeMode == "online" {
		readyTimeout = launcher.OnlineStartupTimeout(config.EffectiveOnlineStartupTimeoutSeconds(settings))
	}
	if err := process.WaitReady(a.ctx, readyTimeout); err != nil {
		_ = process.Stop(3 * time.Second)
		a.failIfCurrent(generation, dshVersion, "DSH 未能按时就绪", err)
		return
	}

	status := Status{
		State:       "ready",
		URL:         process.URL(),
		Message:     "DeepSeek Harness 已就绪",
		Version:     a.version,
		DSHVersion:  dshVersion,
		RuntimeMode: runtimeMode,
		Progress:    100,
		Stage:       "DeepSeek Harness 已就绪",
	}
	if !a.setStatusIfCurrent(generation, status) {
		_ = process.Stop(3 * time.Second)
		return
	}
	// Windows WebView2 未能在 alpha.3 的 loopback 认证跳转中回送 session cookie；
	// 仅将当前进程已校验的 URL 交给系统浏览器，避免桌面窗口陷入不可恢复的 401 页面。
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, process.URL())
	}
	a.mu.Lock()
	if a.pendingRollback != nil && a.pendingRollback.generation == generation {
		a.pendingRollback = nil
	}
	a.mu.Unlock()
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
			DSHVersion:  dshVersion,
			RuntimeMode: runtimeMode,
		}
		a.setStatus(stopped)
		runtime.EventsEmit(a.ctx, stoppedEvent, stopped)
	}()
}

// restart 切换启动代次，旧进程的迟到结果不会覆盖新一轮状态。
func (a *App) restart() Status {
	return a.restartWithProgressAndRollback("正在启动 DeepSeek Harness…", "正在初始化桌面运行时…", 5, nil)
}

// restartWithProgress 切换启动代次，并在停止旧进程前发布新一轮的确定阶段。
func (a *App) restartWithProgress(message, stage string, progress int) Status {
	return a.restartWithProgressAndRollback(message, stage, progress, nil)
}

func (a *App) restartWithProgressAndRollback(message, stage string, progress int, rollback *dshUpdateRollback) Status {
	a.mu.Lock()
	a.generation++
	generation := a.generation
	if rollback != nil {
		rollback.generation = generation
	}
	a.pendingRollback = rollback
	process := a.process
	a.process = nil
	settings := a.settings
	dshVersion := a.dshVersion
	a.status = Status{
		State:       "starting",
		Message:     message,
		Version:     a.version,
		DSHVersion:  dshVersion,
		RuntimeMode: "auto",
		Progress:    progress,
		Stage:       stage,
	}
	status := a.status
	ctx := a.ctx
	a.mu.Unlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, progressEvent, status)
	}

	if process != nil {
		_ = process.Stop(3 * time.Second)
	}
	go a.launch(generation, settings, dshVersion)
	return status
}

// progressIfCurrent 只接收当前启动代次的单调阶段，旧进程不能覆盖新状态。
func (a *App) progressIfCurrent(generation uint64, dshVersion string, progress launcher.Progress) {
	a.mu.Lock()
	if generation != a.generation || progress.Percent <= a.status.Progress {
		a.mu.Unlock()
		return
	}
	message := a.status.Message
	if message == "" {
		message = "正在启动 DeepSeek Harness…"
	}
	status := Status{
		State:       "starting",
		Message:     message,
		Version:     a.version,
		DSHVersion:  dshVersion,
		RuntimeMode: progress.RuntimeMode,
		Progress:    progress.Percent,
		Stage:       progress.Stage,
	}
	a.status = status
	ctx := a.ctx
	a.mu.Unlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, progressEvent, status)
	}
}

func (a *App) failIfCurrent(generation uint64, dshVersion, message string, err error) {
	if a.rollbackFailedUpdate(generation, dshVersion, err) {
		return
	}
	status := Status{
		State:       "failed",
		Message:     message,
		Detail:      err.Error(),
		Version:     a.version,
		DSHVersion:  dshVersion,
		RuntimeMode: "auto",
	}
	if a.setStatusIfCurrent(generation, status) && a.ctx != nil {
		runtime.EventsEmit(a.ctx, failedEvent, status)
	}
}

func (a *App) rollbackFailedUpdate(generation uint64, dshVersion string, cause error) bool {
	a.mu.Lock()
	rollback := a.pendingRollback
	if rollback == nil || rollback.generation != generation || rollback.previousVersion == dshVersion {
		a.mu.Unlock()
		return false
	}
	a.pendingRollback = nil
	ctx := a.ctx
	currentSettings := a.settings
	a.process = nil
	a.mu.Unlock()

	a.settingsMu.Lock()
	defer a.settingsMu.Unlock()
	if currentSettings != rollback.attemptedSettings {
		a.failIfCurrentWithoutRollback(generation, dshVersion, "DSH 更新失败，检测到配置已被其他操作修改", cause)
		return true
	}
	if err := config.SaveIfUnchanged(currentSettings, rollback.previousSettings); err != nil {
		a.failIfCurrentWithoutRollback(generation, dshVersion, "DSH 更新失败且无法恢复旧配置", errors.Join(cause, err))
		return true
	}

	a.mu.Lock()
	a.settings = rollback.previousSettings
	a.settingsLoadErr = nil
	a.dshVersion = rollback.previousVersion
	a.generation++
	newGeneration := a.generation
	a.status = Status{
		State:       "starting",
		Message:     "DSH 更新未完成，正在恢复旧版本…",
		Detail:      cause.Error(),
		Version:     a.version,
		DSHVersion:  rollback.previousVersion,
		RuntimeMode: "auto",
		Progress:    10,
		Stage:       "新版本校验失败，旧版本配置已恢复",
	}
	status := a.status
	a.mu.Unlock()
	if ctx != nil {
		runtime.EventsEmit(ctx, progressEvent, status)
	}
	go a.launch(newGeneration, rollback.previousSettings, rollback.previousVersion)
	return true
}

func (a *App) failIfCurrentWithoutRollback(generation uint64, dshVersion, message string, err error) {
	status := Status{
		State:       "failed",
		Message:     message,
		Detail:      err.Error(),
		Version:     a.version,
		DSHVersion:  dshVersion,
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

func resolveDSHVersion(fallback string, settings config.Settings) (string, error) {
	if value := strings.TrimSpace(os.Getenv("DSH_DESKTOP_DSH_VERSION")); value != "" {
		return dshversion.Normalize(value)
	}
	bundledVersion, bundled, err := launcher.BundledDSHVersion()
	if err != nil {
		return "", err
	}
	if bundled {
		if settings.DSHVersion != "" {
			savedVersion, err := dshversion.Normalize(settings.DSHVersion)
			if err != nil {
				return "", err
			}
			comparison, err := dshversion.Compare(savedVersion, bundledVersion)
			if err != nil {
				return "", err
			}
			if comparison < 0 {
				// 新离线包已包含更高版本；旧配置通常来自上一版在线回退，优先闭包并保留配置供用户清理，避免再次启动旧 npx。
				return bundledVersion, nil
			}
			// 用户明确选择的新版本仍走系统 Node/npm，失败时可通过“恢复默认”回到包内版本。
			return savedVersion, nil
		}
		return bundledVersion, nil
	}
	if settings.DSHVersion != "" {
		return dshversion.Normalize(settings.DSHVersion)
	}
	return dshversion.Normalize(fallback)
}

func canApplyDSHUpdate() (bool, string, error) {
	if strings.TrimSpace(os.Getenv("DSH_DESKTOP_DSH_VERSION")) != "" {
		return false, "当前版本由 DSH_DESKTOP_DSH_VERSION 环境变量控制，请先移除该覆盖。", nil
	}
	bundledVersion, bundled, err := launcher.BundledDSHVersion()
	if err != nil {
		return false, "", err
	}
	if bundled {
		canUseOnline, err := launcher.CanUseOnlineRuntime()
		if err != nil {
			return false, "", err
		}
		if !canUseOnline {
			return false, "当前是 offline-full 离线包（内置 DSH " + bundledVersion + "），且系统未找到 Node.js/npm；请安装 Node.js 22.19+ 或下载包含新版本的 Starline 离线包。", nil
		}
		return true, "", nil
	}
	return true, "", nil
}
