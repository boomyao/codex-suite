//go:build !(darwin || linux)

package main

import (
	"fmt"
	"os"
)

func processExists(pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	return false, nil
}

func signalBridgeStop(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid %d", pid)
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find codex-bridge pid %d: %w", pid, err)
	}
	if err := process.Kill(); err != nil {
		return fmt.Errorf("stop codex-bridge pid %d: %w", pid, err)
	}
	return nil
}
