package application

import (
	"sync"

	"github.com/getlantern/systray"
)

// Tray 管理跨平台通知区域图标，并把菜单动作转交给 Wails App。
type Tray struct {
	app      *App
	icon     []byte
	quitOnce sync.Once
}

// NewTray 创建桌面托盘控制器。
func NewTray(app *App, icon []byte) *Tray {
	return &Tray{app: app, icon: icon}
}

// Register 在 Wails 启动自己的 WebView 事件循环前注册托盘回调。
func (t *Tray) Register() {
	systray.Register(func() {
		systray.SetIcon(t.icon)
		systray.SetTooltip("Starline DSH Desktop")

		show := systray.AddMenuItem("显示窗口", "恢复 Starline DSH Desktop")
		restart := systray.AddMenuItem("重启 DSH", "重新启动 DeepSeek Harness")
		systray.AddSeparator()
		quit := systray.AddMenuItem("退出", "退出桌面端并释放 DSH 进程")

		go func() {
			for {
				select {
				case <-show.ClickedCh:
					t.app.ShowWindow()
				case <-restart.ClickedCh:
					t.app.Retry()
					t.app.ShowWindow()
				case <-quit.ClickedCh:
					t.app.RequestQuit()
				}
			}
		}()
	}, nil)
}

// Quit 移除托盘图标；Wails 结束后由主程序调用，避免留下通知区域句柄。
func (t *Tray) Quit() {
	t.quitOnce.Do(systray.Quit)
}
