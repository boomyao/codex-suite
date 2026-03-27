package embeddedui

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed all:desktop-webview/**
var desktopWebviewFS embed.FS

const (
	rootDirName = "desktop-webview"
	stampName   = ".codex-bridge-embedded-sha256"
)

func EnsureMaterialized(dataDir string) (string, error) {
	resolvedDataDir, err := filepath.Abs(strings.TrimSpace(dataDir))
	if err != nil {
		return "", err
	}

	destRoot := filepath.Join(resolvedDataDir, "embedded", rootDirName)
	digest, err := digestEmbeddedRoot()
	if err != nil {
		return "", err
	}

	if current, err := os.ReadFile(filepath.Join(destRoot, stampName)); err == nil {
		if strings.TrimSpace(string(current)) == digest && validMaterializedRoot(destRoot) {
			return destRoot, nil
		}
	}

	tempRoot := destRoot + ".tmp"
	_ = os.RemoveAll(tempRoot)
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return "", err
	}

	if err := materializeInto(tempRoot); err != nil {
		_ = os.RemoveAll(tempRoot)
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tempRoot, stampName), []byte(digest+"\n"), 0o644); err != nil {
		_ = os.RemoveAll(tempRoot)
		return "", err
	}

	_ = os.RemoveAll(destRoot)
	if err := os.MkdirAll(filepath.Dir(destRoot), 0o755); err != nil {
		_ = os.RemoveAll(tempRoot)
		return "", err
	}
	if err := os.Rename(tempRoot, destRoot); err != nil {
		_ = os.RemoveAll(tempRoot)
		return "", err
	}

	return destRoot, nil
}

func validMaterializedRoot(root string) bool {
	info, err := os.Stat(filepath.Join(root, "index.html"))
	return err == nil && !info.IsDir()
}

func materializeInto(destRoot string) error {
	return fs.WalkDir(desktopWebviewFS, rootDirName, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(rootDirName, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destRoot, relative)

		if entry.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		sourceFile, err := desktopWebviewFS.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}

		targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, sourceFile); err != nil {
			return err
		}
		return nil
	})
}

func digestEmbeddedRoot() (string, error) {
	var paths []string
	if err := fs.WalkDir(desktopWebviewFS, rootDirName, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		paths = append(paths, path)
		return nil
	}); err != nil {
		return "", err
	}

	sort.Strings(paths)
	hasher := sha256.New()
	for _, path := range paths {
		if _, err := hasher.Write([]byte(path)); err != nil {
			return "", err
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return "", err
		}

		file, err := desktopWebviewFS.Open(path)
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(hasher, file); err != nil {
			file.Close()
			return "", err
		}
		file.Close()

		if _, err := hasher.Write([]byte{0}); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
