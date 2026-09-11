// Package instance 维护桌面宿主的单实例所有权。
//
// DSH 的会话写锁是内核级的：同一个会话目录在同一时刻只能有一个写句柄。宿主把 DSH 当作普通
// loopback Web 应用启动，因此两个宿主进程会共用同一份 DSH 用户数据目录，第二个进程 resume
// 会话时会被内核拒绝，前端显示
// `command directory warmup failed: command.list failed: gateway/internal: resume failed for
// session "…": SessionAlreadyOwnedError: session "…" is already owned by an active write handle`。
// 这不是认证或 token 交接问题，只有“同一登录会话只允许一个宿主”才能消除这种争用。
//
// 所有权由内核对象承载而不是文件内容：持有者进程无论正常退出还是被强杀，内核都会在句柄关闭时
// 释放它，因此不存在需要人工判断的陈旧锁。记录文件只保存当前实例的 PID，用于把已有窗口带到前台
// 和人工排查，不参与所有权判定。
package instance

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"starline-dsh-desktop/internal/config"
)

// recordName 是实例记录文件名；它与 settings.json 同级，只保存当前实例的 PID。
const recordName = "instance.lock"

// record 是实例记录的内容；字段保持最小，新增字段必须能容忍旧文件缺失。
type record struct {
	PID int `json:"pid"`
}

// Guard 表示当前进程持有的实例所有权。在 Release 之前，其他宿主不得启动 DSH。
type Guard struct {
	release func() error
	once    sync.Once
}

// Release 释放所有权；内核对象随最后一个句柄关闭而销毁，重复调用是安全的。
func (g *Guard) Release() error {
	if g == nil {
		return nil
	}
	var err error
	g.once.Do(func() {
		err = g.release()
	})
	return err
}

// Acquire 抢占当前用户会话里的桌面宿主所有权。
//
// acquired 为 false 表示已有实例在运行：调用方必须改为激活已有窗口并退出，不能启动第二个 DSH。
// 返回的错误只表示所有权机制本身不可用，与“是否已有实例”无关。
func Acquire() (*Guard, bool, error) {
	path, err := recordPath()
	if err != nil {
		return nil, false, err
	}
	return acquireGuard(instanceName, path)
}

// ActivateExisting 请求已在运行的实例把主窗口带到前台，并在无法交接时说明原因。
// 它只影响第二次双击的体验，不参与所有权判定。
func ActivateExisting(windowTitle string) {
	path, err := recordPath()
	if err != nil {
		return
	}
	activate(path, windowTitle)
}

// activate 优先把已有窗口带回前台；做不到时只说明原因。
func activate(recordPath, windowTitle string) {
	if bringWindowToFront(recordPath, windowTitle) {
		return
	}
	notifyExisting(windowTitle)
}

// notifyExisting 是第二实例唯一可用的反馈通道：此时 Wails 还没有启动，宿主没有自己的界面，
// 而且不允许用阻塞式交互卡住可能来自脚本或自动化的启动请求。
func notifyExisting(windowTitle string) {
	_, _ = fmt.Fprintf(os.Stderr, "%s 已在运行：同一份 DSH 用户数据只允许一个宿主写入会话，本次启动不会新建实例。请使用已有窗口，或从托盘退出它之后重新启动。\n", windowTitle)
}

// recordPath 返回当前用户的实例记录路径。
func recordPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, recordName), nil
}

// prepareRecordDirectory 确保记录文件所在目录存在，并使用用户私有权限。
func prepareRecordDirectory(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("无法创建实例记录目录：%w", err)
	}
	return nil
}

// encodeRecord 把当前 PID 编码成实例记录内容。
func encodeRecord(pid int) ([]byte, error) {
	data, err := json.Marshal(record{PID: pid})
	if err != nil {
		return nil, fmt.Errorf("无法编码实例记录：%w", err)
	}
	return data, nil
}

// writeRecord 覆盖写入实例记录；只有取得所有权的实例会调用它。
func writeRecord(path string, pid int) error {
	if err := prepareRecordDirectory(path); err != nil {
		return err
	}
	data, err := encodeRecord(pid)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("无法写入实例记录：%w", err)
	}
	return nil
}

// readRecordPID 读取已有实例的 PID；记录缺失或损坏时返回错误，调用方按“无法交接”处理。
func readRecordPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	var parsed record
	if err := json.Unmarshal(data, &parsed); err != nil {
		return 0, errors.New("实例记录格式无效")
	}
	if parsed.PID <= 0 {
		return 0, errors.New("实例记录缺少有效 PID")
	}
	return parsed.PID, nil
}
