//go:build darwin || linux

package main

import (
	"fmt"
	"syscall"
)

func processExists(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	err := syscall.Kill(pid, 0)
	if err == nil || err == syscall.EPERM {
		return true, nil
	}
	if err == syscall.ESRCH {
		return false, nil
	}
	return false, fmt.Errorf("check process %d: %w", pid, err)
}

func signalBridgeStop(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid %d", pid)
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		if err == syscall.ESRCH {
			return nil
		}
		return fmt.Errorf("signal codex-bridge pid %d: %w", pid, err)
	}
	return nil
}
