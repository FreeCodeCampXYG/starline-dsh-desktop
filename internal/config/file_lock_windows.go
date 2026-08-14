//go:build windows

package config

import (
	"os"

	"golang.org/x/sys/windows"
)

func lockFile(file *os.File) error {
	return windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		^uint32(0),
		^uint32(0),
		&windows.Overlapped{},
	)
}

func unlockFile(file *os.File) error {
	return windows.UnlockFileEx(
		windows.Handle(file.Fd()),
		0,
		^uint32(0),
		^uint32(0),
		&windows.Overlapped{},
	)
}
