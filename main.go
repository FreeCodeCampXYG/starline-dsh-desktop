package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"starline-dsh-desktop/internal/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/windows/icon.ico
var trayIcon []byte

var (
	version           = "dev"
	defaultDSHVersion = "0.1.0-rc.6"
)

func main() {
	app := application.New(version, defaultDSHVersion)
	tray := application.NewTray(app, trayIcon)
	tray.Register()
	defer tray.Quit()
	err := wails.Run(&options.App{
		Title:             "Starline DSH Desktop",
		Width:             1280,
		Height:            820,
		MinWidth:          900,
		MinHeight:         620,
		DisableResize:     false,
		Fullscreen:        false,
		Frameless:         false,
		StartHidden:       true,
		HideWindowOnClose: application.HideWindowOnClose(),
		BackgroundColour:  &options.RGBA{R: 8, G: 12, B: 20, A: 1},
		AssetServer:       &assetserver.Options{Assets: assets},
		Menu:              application.Menu(app),
		OnStartup:         app.Startup,
		OnShutdown:        app.Shutdown,
		OnBeforeClose:     app.BeforeClose,
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
