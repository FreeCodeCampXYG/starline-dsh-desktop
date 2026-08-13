//go:build !windows

package launcher

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
)

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func configureAuxiliaryProcess(*exec.Cmd) {}

func stopProcessTree(pid int) error {
	err := syscall.Kill(-pid, syscall.SIGTERM)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	time.Sleep(150 * time.Millisecond)
	return nil
}
