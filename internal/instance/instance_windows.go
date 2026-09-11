//go:build windows

package instance

import (
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// instanceName 位于 Local\ 命名空间：只在当前 Windows 登录会话内唯一。这与 DSH 的会话写锁按登录
// 会话隔离的语义一致，不同 Windows 用户可以各自运行一个宿主，互不干扰。
const instanceName = `Local\starline-dsh-desktop-host`

const showWindowRestore = 9

// user32 的窗口句柄调用不在 x/sys/windows 里；这里只需两个入口，不值得为此引入第三方依赖。
var (
	user32DLL           = windows.NewLazySystemDLL("user32.dll")
	findWindowW         = user32DLL.NewProc("FindWindowW")
	showWindow          = user32DLL.NewProc("ShowWindow")
	setForegroundWindow = user32DLL.NewProc("SetForegroundWindow")
)

// acquireGuard 用命名内核互斥体表达“当前登录会话已有一个宿主”。
//
// 所有权就是对象存在性本身：不等待、不夺取，也不需要初始所有权。持有者进程结束时内核关闭句柄并
// 销毁对象，因此命中 ERROR_ALREADY_EXISTS 一定意味着另一个**存活**的宿主，而不是陈旧的锁文件。
// name 由调用方给出，使测试可以使用独立名称，不与真实运行中的桌面实例互相干扰。
func acquireGuard(name, recordPath string) (*Guard, bool, error) {
	namePointer, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, fmt.Errorf("实例名无效：%w", err)
	}
	handle, err := windows.CreateMutex(nil, false, namePointer)
	if err == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(handle)
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("无法创建实例互斥体：%w", err)
	}
	if err := writeRecord(recordPath, os.Getpid()); err != nil {
		_ = windows.CloseHandle(handle)
		return nil, false, err
	}
	return &Guard{release: func() error {
		return windows.CloseHandle(handle)
	}}, true, nil
}

// bringWindowToFront 用“窗口标题 + 记录中的 PID”定位已有宿主的主窗口，避免误操作同名的其他窗口。
// 托盘模式下窗口可能处于隐藏状态，FindWindow 同样能找到它。
func bringWindowToFront(recordPath, windowTitle string) bool {
	pid, err := readRecordPID(recordPath)
	if err != nil {
		return false
	}
	titlePointer, err := windows.UTF16PtrFromString(windowTitle)
	if err != nil {
		return false
	}
	window, _, _ := findWindowW.Call(0, uintptr(unsafe.Pointer(titlePointer)))
	if window == 0 {
		return false
	}
	var owner uint32
	if _, err := windows.GetWindowThreadProcessId(windows.HWND(window), &owner); err != nil || owner != uint32(pid) {
		return false
	}
	_, _, _ = showWindow.Call(window, showWindowRestore)
	// 前台切换受 Windows 前台锁定策略限制可能失败，但窗口已经可见，不再额外打扰用户。
	_, _, _ = setForegroundWindow.Call(window)
	return true
}
