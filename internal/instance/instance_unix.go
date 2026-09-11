//go:build !windows

package instance

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// acquireGuard 用记录文件上的 flock 表达“当前用户会话已有一个宿主”。
//
// flock 由内核在进程结束时释放，因此崩溃或被强杀的持有者不会留下需要人工清理的锁；记录文件本身
// 保留在磁盘上，内容只用于把窗口带到前台和人工排查。name 在 Unix 上没有对应的内核对象，
// 仅 Windows 使用，这里刻意忽略。
func acquireGuard(_, recordPath string) (*Guard, bool, error) {
	if err := prepareRecordDirectory(recordPath); err != nil {
		return nil, false, err
	}
	file, err := os.OpenFile(recordPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, fmt.Errorf("无法打开实例记录：%w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if isLockBusy(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("无法获取实例锁：%w", err)
	}
	if err := writeRecord(recordPath, os.Getpid()); err != nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
		return nil, false, err
	}
	return &Guard{release: func() error {
		unlockErr := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		closeErr := file.Close()
		if unlockErr != nil {
			return unlockErr
		}
		return closeErr
	}}, true, nil
}

// bringWindowToFront 在 Unix 上总是失败：宿主主窗口由 Wails 交给系统窗口管理器管理，
// 没有跨进程恢复已隐藏窗口的统一手段，因此第二个实例只说明原因并退出，绝不启动第二个 DSH。
func bringWindowToFront(_, _ string) bool {
	return false
}

// isLockBusy 判断 flock 失败是否只是“锁已被占用”。
func isLockBusy(err error) bool {
	return errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN)
}
