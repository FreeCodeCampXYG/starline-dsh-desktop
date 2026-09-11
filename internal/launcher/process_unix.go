//go:build !windows

package launcher

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func configureAuxiliaryProcess(*exec.Cmd) {}

// containProcessTree 在 Unix 上是空实现：Linux 的 PDEATHSIG 在 macOS 上不存在，而 DSH 的会话写锁
// 由内核在进程退出时释放，因此这里沿用既有的“关闭时回收自己启动的进程组”边界，不引入平台分支。
func containProcessTree(*os.Process) error {
	return nil
}

func stopProcessTree(pid int) error {
	err := syscall.Kill(-pid, syscall.SIGTERM)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	time.Sleep(150 * time.Millisecond)
	return nil
}
