//go:build windows

package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const createNoWindow uint32 = 0x08000000

// job 是宿主唯一的 Job 对象，一次性创建后所有 DSH 子进程都加入其中。
// KILL_ON_JOB_CLOSE 由内核保证：宿主进程无论正常退出、崩溃还是被任务管理器强杀，最后一个 job
// 句柄关闭时内核都会连同后代一起回收，因此不会留下继续占用 DSH 会话写锁的孤儿进程。
var (
	jobOnce sync.Once
	job     windows.Handle
	jobErr  error
)

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

// containProcessTree 把刚启动的 DSH 子进程交给宿主的 Job 对象。加入 Job 的进程会把它之后创建的
// 子进程（npx 下载链、PTY Shell、工具进程）一并带进来，无需按进程名扫描。
func containProcessTree(process *os.Process) error {
	if process == nil {
		return nil
	}
	handle, err := openJobHost()
	if err != nil {
		return err
	}
	candidate, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(process.Pid))
	if err != nil {
		return fmt.Errorf("无法打开 DSH 进程句柄：%w", err)
	}
	defer func() { _ = windows.CloseHandle(candidate) }()
	if err := windows.AssignProcessToJobObject(handle, candidate); err != nil {
		return fmt.Errorf("无法把 DSH 进程加入宿主 Job：%w", err)
	}
	return nil
}

// openJobHost 惰性创建带 KILL_ON_JOB_CLOSE 的 Job 对象，并在本次进程内复用。
func openJobHost() (windows.Handle, error) {
	jobOnce.Do(func() {
		created, err := windows.CreateJobObject(nil, nil)
		if err != nil {
			jobErr = fmt.Errorf("无法创建宿主 Job 对象：%w", err)
			return
		}
		limits := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
			BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
				LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
			},
		}
		if _, err := windows.SetInformationJobObject(
			created,
			windows.JobObjectExtendedLimitInformation,
			uintptr(unsafe.Pointer(&limits)),
			uint32(unsafe.Sizeof(limits)),
		); err != nil {
			_ = windows.CloseHandle(created)
			jobErr = fmt.Errorf("无法设置宿主 Job 的回收限制：%w", err)
			return
		}
		job = created
	})
	return job, jobErr
}

func stopProcessTree(pid int) error {
	command := exec.Command("taskkill.exe", "/PID", strconv.Itoa(pid), "/T", "/F")
	configureAuxiliaryProcess(command)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("终止 DSH 进程树失败：%w（%s）", err, string(output))
	}
	return nil
}
