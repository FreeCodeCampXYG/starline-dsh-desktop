package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type fileLock struct {
	file *os.File
}

func acquireFileLock(configPath string) (*fileLock, error) {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return nil, fmt.Errorf("无法创建配置锁目录：%w", err)
	}
	lockPath := configPath + ".lock"
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("无法打开配置锁：%w", err)
	}
	if err := lockFile(file); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("配置正在被其他桌面实例更新，请稍后重试：%w", err)
	}
	return &fileLock{file: file}, nil
}

func (lock *fileLock) Close() error {
	if lock == nil || lock.file == nil {
		return nil
	}
	unlockErr := unlockFile(lock.file)
	closeErr := lock.file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
