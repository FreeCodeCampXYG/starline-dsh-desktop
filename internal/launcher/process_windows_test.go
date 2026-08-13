//go:build windows

package launcher

import (
	"os/exec"
	"testing"
)

func TestAuxiliaryProcessIsHidden(t *testing.T) {
	command := exec.Command("cmd.exe", "/c", "exit", "0")
	configureAuxiliaryProcess(command)
	if command.SysProcAttr == nil || !command.SysProcAttr.HideWindow {
		t.Fatal("辅助命令没有启用隐藏窗口")
	}
	if command.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatal("辅助命令缺少 CREATE_NO_WINDOW")
	}
}
