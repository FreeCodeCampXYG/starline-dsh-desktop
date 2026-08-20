package application

import (
	"runtime"

	"github.com/wailsapp/wails/v2/pkg/menu"
)

// Menu 仅在 macOS 保留符合平台习惯的系统菜单；其他平台使用窗口内菜单。
func Menu(app *App) *menu.Menu {
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
	application.AddText("退出", nil, func(*menu.CallbackData) {
		app.RequestQuit()
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
	help.AddText("检查 Desktop 更新", nil, func(*menu.CallbackData) {
		_ = app.OpenDesktopReleasePage()
	})
	help.AddText("GitHub 项目主页", nil, func(*menu.CallbackData) {
		_ = app.OpenProjectPage()
	})
	help.AddText("在浏览器中打开 DSH", nil, func(*menu.CallbackData) {
		_ = app.OpenInBrowser()
	})
	return result
}
