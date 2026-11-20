package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// A menu bar app should never run twice: two status items for the same deck is
// confusing and both would fight over the saved state.
func acquireSingleInstance() error {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	lockDir := filepath.Join(dir, appName)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		return nil // if we cannot create a lock dir, let the app start anyway
	}
	path := filepath.Join(lockDir, "instance.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return fmt.Errorf("already running (close the existing menu bar item first)")
	}
	// Keep the file descriptor open for the lifetime of the process; the lock is
	// released by the kernel when the process exits.
	instanceLock = f
	return nil
}

var instanceLock *os.File
