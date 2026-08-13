package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/menu"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

var version = "dev"

func main() {
	app := NewApp(version)
	err := wails.Run(&options.App{
		Title:             "Starline DSH Desktop",
		Width:             1280,
		Height:            820,
		MinWidth:          900,
		MinHeight:         620,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       false,
		HideWindowOnClose: false,
		BackgroundColour:  &options.RGBA{R: 8, G: 12, B: 20, A: 1},
		AssetServer:       &assetserver.Options{Assets: assets},
		Menu:              applicationMenu(app),
		OnStartup:         app.startup,
		OnShutdown:        app.shutdown,
		Bind:              []interface{}{app},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			WebviewUserDataPath:  "",
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
			About: &mac.AboutInfo{
				Title:   "Starline DSH Desktop",
				Message: "A thin desktop host for DeepSeek Harness.",
			},
		},
		Linux: &linux.Options{
			ProgramName: "Starline DSH Desktop",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// applicationMenu 仅在 macOS 保留符合平台习惯的系统菜单；其他平台使用窗口内菜单。
func applicationMenu(app *App) *menu.Menu {
	if runtime.GOOS != "darwin" {
		return nil
	}
	result := menu.NewMenu()
	result.Append(menu.AppMenu())
	application := result.AddSubmenu("DSH")
	application.AddText("代理与启动设置…", nil, func(*menu.CallbackData) {
		app.emit("shell:open-settings")
	})
	application.AddText("重新启动 DSH", nil, func(*menu.CallbackData) {
		app.Retry()
	})
	application.AddSeparator()
	application.AddText("打开日志目录", nil, func(*menu.CallbackData) {
		_ = app.OpenLogs()
	})
	result.Append(menu.EditMenu())
	result.Append(menu.WindowMenu())
	help := result.AddSubmenu("帮助")
	help.AddText("Starline DSH Desktop 使用帮助", nil, func(*menu.CallbackData) {
		app.emit("shell:open-help")
	})
	help.AddText("在浏览器中打开 DSH", nil, func(*menu.CallbackData) {
		_ = app.OpenInBrowser()
	})
	return result
}
