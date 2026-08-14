//go:build !windows

package application

// Tray 是非 Windows 平台的空实现。Wails v2 与第三方托盘库在 macOS
// 存在 AppDelegate 符号冲突，在无显示的 Linux 构建环境中也会触发 GTK 初始化。
type Tray struct{}

// NewTray 保持主程序的平台无关调用方式。
func NewTray(_ *App, _ []byte) *Tray {
	return &Tray{}
}

// Register 在非 Windows 平台不注册第三方托盘。
func (t *Tray) Register() {}

// Quit 在非 Windows 平台无需清理第三方托盘。
func (t *Tray) Quit() {}

// HideWindowOnClose 让非 Windows 平台的关闭按钮执行正常退出，避免隐藏后无法恢复。
func HideWindowOnClose() bool {
	return false
}

func allowWindowCloseWithoutTray() bool {
	return true
}
