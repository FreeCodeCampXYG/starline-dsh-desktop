//go:build windows

package launcher

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
)

const createNoWindow uint32 = 0x08000000

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | createNoWindow,
	}
}

func configureAuxiliaryProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
}

func stopProcessTree(pid int) error {
	command := exec.Command("taskkill.exe", "/PID", strconv.Itoa(pid), "/T", "/F")
	configureAuxiliaryProcess(command)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("终止 DSH 进程树失败：%w（%s）", err, string(output))
	}
	return nil
}
