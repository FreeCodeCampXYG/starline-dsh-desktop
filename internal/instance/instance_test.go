package instance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	childFlagEnv   = "STARLINE_DSH_INSTANCE_CHILD"
	childNameEnv   = "STARLINE_DSH_INSTANCE_NAME"
	childRecordEnv = "STARLINE_DSH_INSTANCE_RECORD"
)

// newTestName 返回只属于本次测试进程的实例名，避免与本机真实运行的桌面实例互相干扰。
func newTestName() string {
	return fmt.Sprintf(`Local\starline-dsh-desktop-test-%d`, os.Getpid())
}

// TestAcquireGuardBlocksSeparateProcess 用真实子进程验证所有权是**跨进程**的：
// 这是本次修复的核心性质，同一个进程内两次调用只能证明互斥逻辑，不能证明第二个宿主会被挡住。
func TestAcquireGuardBlocksSeparateProcess(t *testing.T) {
	name := newTestName()
	record := filepath.Join(t.TempDir(), recordName)
	if os.Getenv(childFlagEnv) == "1" {
		guard, acquired, err := acquireGuard(os.Getenv(childNameEnv), os.Getenv(childRecordEnv))
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if acquired {
			_ = guard.Release()
			_, _ = fmt.Fprint(os.Stdout, "RESULT=ACQUIRED")
		} else {
			_, _ = fmt.Fprint(os.Stdout, "RESULT=BLOCKED")
		}
		return
	}

	owner, acquired, err := acquireGuard(name, record)
	if err != nil {
		t.Fatalf("首次获取所有权失败：%v", err)
	}
	if !acquired {
		t.Fatal("空闲状态下首次获取应当成功")
	}
	defer func() { _ = owner.Release() }()

	if output := runChildAcquire(t, name, record); !strings.Contains(output, "RESULT=BLOCKED") {
		t.Fatalf("另一个进程仍在运行时，子进程不应取得所有权：%q", output)
	}
	if err := owner.Release(); err != nil {
		t.Fatalf("释放所有权失败：%v", err)
	}
	if output := runChildAcquire(t, name, record); !strings.Contains(output, "RESULT=ACQUIRED") {
		t.Fatalf("所有权释放后，子进程应当取得所有权：%q", output)
	}
}

// runChildAcquire 重新执行当前测试二进制，在独立进程里尝试取得同一份所有权。
func runChildAcquire(t *testing.T, name, record string) string {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run="+t.Name())
	command.Env = append(os.Environ(),
		childFlagEnv+"=1",
		childNameEnv+"="+name,
		childRecordEnv+"="+record,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("子进程执行失败：%v（%s）", err, output)
	}
	return string(output)
}

func TestAcquireGuardRefusesSecondOwnerUntilRelease(t *testing.T) {
	record := filepath.Join(t.TempDir(), recordName)
	name := newTestName()

	first, acquired, err := acquireGuard(name, record)
	if err != nil {
		t.Fatalf("首次获取所有权失败：%v", err)
	}
	if !acquired {
		t.Fatal("空闲状态下首次获取应当成功")
	}
	defer func() { _ = first.Release() }()

	pid, err := readRecordPID(record)
	if err != nil {
		t.Fatalf("实例记录应可读：%v", err)
	}
	if pid != os.Getpid() {
		t.Fatalf("实例记录 PID = %d, want %d", pid, os.Getpid())
	}

	second, acquired, err := acquireGuard(name, record)
	if err != nil {
		t.Fatalf("第二次获取返回了机制错误：%v", err)
	}
	if acquired {
		_ = second.Release()
		t.Fatal("已有实例持有所有权时不应再次获取成功")
	}

	if err := first.Release(); err != nil {
		t.Fatalf("释放所有权失败：%v", err)
	}
	third, acquired, err := acquireGuard(name, record)
	if err != nil {
		t.Fatalf("释放后重新获取失败：%v", err)
	}
	if !acquired {
		t.Fatal("释放后应当可以重新获取所有权")
	}
	_ = third.Release()
}

func TestReadRecordPIDRejectsMissingRecord(t *testing.T) {
	if _, err := readRecordPID(filepath.Join(t.TempDir(), recordName)); err == nil {
		t.Fatal("实例记录缺失时应当返回错误")
	}
}

func TestActivateExistingWithoutRecord(t *testing.T) {
	// 没有记录文件时只应静默放弃交接，不影响所有权判定。
	activate(filepath.Join(t.TempDir(), recordName), "Starline DSH Desktop")
}
