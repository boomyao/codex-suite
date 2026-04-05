package bridge

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type desktopEditableState struct {
	Active         bool   `json:"active"`
	Text           string `json:"text"`
	Placeholder    string `json:"placeholder"`
	SelectionStart int    `json:"selectionStart"`
	SelectionEnd   int    `json:"selectionEnd"`
	Role           string `json:"role"`
	Error          string `json:"error"`
}

var desktopAccessibilityHelperBuildMu sync.Mutex

func desktopFocusedEditableState() (desktopEditableState, error) {
	return runDesktopAccessibilityHelper("focused-editable-state")
}

func desktopSetFocusedTextState(text string, selectionStart int, selectionEnd int) (desktopEditableState, error) {
	return runDesktopAccessibilityHelper(
		"set-focused-text-state",
		base64.StdEncoding.EncodeToString([]byte(text)),
		strconv.Itoa(selectionStart),
		strconv.Itoa(selectionEnd),
	)
}

func runDesktopAccessibilityHelper(args ...string) (desktopEditableState, error) {
	binaryPath, err := ensureDesktopAccessibilityHelperBinary()
	if err != nil {
		return desktopEditableState{}, err
	}

	command := exec.Command(binaryPath, args...)
	output, err := command.CombinedOutput()

	var payload desktopEditableState
	if len(output) > 0 {
		if decodeErr := json.Unmarshal(output, &payload); decodeErr != nil {
			return desktopEditableState{}, fmt.Errorf("desktop accessibility helper returned invalid JSON: %w (%s)", decodeErr, strings.TrimSpace(string(output)))
		}
	}

	if err != nil {
		message := strings.TrimSpace(payload.Error)
		if message == "" {
			message = strings.TrimSpace(string(output))
		}
		if message == "" {
			message = err.Error()
		}
		return desktopEditableState{}, errors.New(message)
	}

	return payload, nil
}

func ensureDesktopAccessibilityHelperBinary() (string, error) {
	desktopAccessibilityHelperBuildMu.Lock()
	defer desktopAccessibilityHelperBuildMu.Unlock()

	sourcePath, err := desktopAccessibilityHelperSourcePath()
	if err != nil {
		return "", err
	}
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return "", fmt.Errorf("desktop accessibility helper source is unavailable: %w", err)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve the user cache directory: %w", err)
	}
	binaryPath := filepath.Join(cacheDir, "codex-bridge", "desktop_accessibility_helper")
	if binaryInfo, err := os.Stat(binaryPath); err == nil &&
		binaryInfo.ModTime().After(sourceInfo.ModTime()) {
		return binaryPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to prepare the desktop accessibility helper cache directory: %w", err)
	}

	tmpPath := binaryPath + ".tmp"
	_ = os.Remove(tmpPath)
	command := exec.Command(
		"swiftc",
		sourcePath,
		"-O",
		"-framework", "AppKit",
		"-framework", "ApplicationServices",
		"-o", tmpPath,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to compile the desktop accessibility helper: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	if err := os.Rename(tmpPath, binaryPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to finalize the desktop accessibility helper binary: %w", err)
	}
	return binaryPath, nil
}

func desktopAccessibilityHelperSourcePath() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to locate the desktop accessibility helper source")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	sourcePath := filepath.Join(root, "helpers", "desktop_accessibility_helper.swift")
	if _, err := os.Stat(sourcePath); err != nil {
		return "", err
	}
	return sourcePath, nil
}
