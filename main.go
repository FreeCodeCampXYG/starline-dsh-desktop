package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"starline-dsh-desktop/internal/application"
	"starline-dsh-desktop/internal/instance"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/windows/icon.ico
var trayIcon []byte

// windowTitle 同时是窗口标题和第二个实例定位已有窗口的依据，只能有一处定义。
const windowTitle = "Starline DSH Desktop"

var (
	version           = "dev"
	defaultDSHVersion = "0.1.5-rc.1"
)

func main() {
	// 单实例判定必须在创建窗口、托盘和 DSH 子进程之前：两个宿主会共用同一份 DSH 用户数据目录，
	// 后启动的那个 resume 会话时会被会话写锁拒绝，前端表现为 SessionAlreadyOwnedError 红条。
	guard, acquired, err := instance.Acquire()
	if err != nil {
		log.Fatal(err)
	}
	if !acquired {
		instance.ActivateExisting(windowTitle)
		return
	}
	defer func() { _ = guard.Release() }()

	app := application.New(version, defaultDSHVersion)
	tray := application.NewTray(app, trayIcon)
	tray.Register()
	defer tray.Quit()
	err = wails.Run(&options.App{
		Title:                    windowTitle,
		Width:                    1280,
		Height:                   820,
		MinWidth:                 900,
		MinHeight:                620,
		DisableResize:            false,
		Fullscreen:               false,
		Frameless:                false,
		StartHidden:              true,
		HideWindowOnClose:        application.HideWindowOnClose(),
		EnableDefaultContextMenu: true,
		BackgroundColour:         &options.RGBA{R: 8, G: 12, B: 20, A: 1},
		AssetServer:              &assetserver.Options{Assets: assets},
		Menu:                     application.Menu(app),
		OnStartup:                app.Startup,
		OnShutdown: func(ctx context.Context) {
			// DSH 子树回收完成就立刻让位，否则“退出后马上重新启动”会被误判为已有实例在运行。
			app.Shutdown(ctx)
			_ = guard.Release()
		},
		OnBeforeClose: app.BeforeClose,
		Bind:          []interface{}{app},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			WebviewUserDataPath:  "",
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
			About: &mac.AboutInfo{
				Title:   "Starline DSH Desktop",
				Message: "A thin desktop host for DeepSeek Harness.\nCopyright (c) 2026 starline and contributors.\nhttps://github.com/FreeCodeCampXYG/starline-dsh-desktop",
				Icon:    appIcon,
			},
		},
		Linux: &linux.Options{
			Icon:        appIcon,
			ProgramName: "Starline DSH Desktop",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
