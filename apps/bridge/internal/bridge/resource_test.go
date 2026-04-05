package bridge

import (
	"encoding/base64"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveMobileResourceAllowsWorkspaceFile(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	workspaceDir := filepath.Join(homeDir, "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		t.Fatalf("create workspace dir: %v", err)
	}
	filePath := filepath.Join(workspaceDir, "context.png")
	if err := os.WriteFile(filePath, []byte("png"), 0o644); err != nil {
		t.Fatalf("write workspace file: %v", err)
	}
	if err := saveGlobalStateFile(globalStateFilePath(), map[string]any{
		"active-workspace-roots": []string{workspaceDir},
	}); err != nil {
		t.Fatalf("write global state: %v", err)
	}

	bridge := New(Config{}, log.New(io.Discard, "", 0))
	result := bridge.resolveMobileResource(map[string]any{
		"path": filePath,
	})

	if supported, _ := result["supported"].(bool); !supported {
		t.Fatalf("expected supported result, got %#v", result)
	}
	handle, ok := result["handle"].(map[string]any)
	if !ok {
		t.Fatalf("expected handle payload, got %#v", result)
	}
	if kind, _ := handle["kind"].(string); kind != "image" {
		t.Fatalf("expected image handle, got %#v", handle)
	}
	if resourceURL, _ := result["url"].(string); !strings.HasPrefix(resourceURL, "/codex-mobile/resource/") {
		t.Fatalf("expected resource url, got %#v", result)
	}
}

func TestResolveMobileResourceRejectsNonFileURI(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	bridge := New(Config{}, log.New(io.Discard, "", 0))
	result := bridge.resolveMobileResource(map[string]any{
		"uri": "https://example.com/image.png",
	})

	if supported, _ := result["supported"].(bool); supported {
		t.Fatalf("expected unsupported result, got %#v", result)
	}
	errorMessage, _ := result["error"].(string)
	if !strings.Contains(errorMessage, "file://") {
		t.Fatalf("expected file uri validation error, got %#v", result)
	}
}

func TestUploadPickedMobileFileWritesTempFile(t *testing.T) {
	homeDir := t.TempDir()
	tempDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("TMPDIR", tempDir)

	bridge := New(Config{}, log.New(io.Discard, "", 0))
	result := bridge.uploadPickedMobileFile(map[string]any{
		"name":           "../mobile-photo.png",
		"contentsBase64": base64.StdEncoding.EncodeToString([]byte("picked-image")),
	})

	if errorMessage, _ := result["error"].(string); errorMessage != "" {
		t.Fatalf("unexpected upload error: %#v", result)
	}
	label, _ := result["label"].(string)
	if label != "mobile-photo.png" {
		t.Fatalf("expected sanitized label, got %#v", result)
	}
	path, _ := result["path"].(string)
	fsPath, _ := result["fsPath"].(string)
	if path == "" || fsPath == "" || path != fsPath {
		t.Fatalf("expected desktop file path, got %#v", result)
	}
	if !strings.HasPrefix(path, filepath.Join(tempDir, mobilePickedFilesDirName)+string(filepath.Separator)) {
		t.Fatalf("expected picked file inside tmp dir, got %q", path)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read picked file: %v", err)
	}
	if string(contents) != "picked-image" {
		t.Fatalf("unexpected picked file contents: %q", string(contents))
	}
}

func TestReadFileBinaryAllowsTempFile(t *testing.T) {
	homeDir := t.TempDir()
	tempDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("TMPDIR", tempDir)

	filePath := filepath.Join(tempDir, "preview.png")
	if err := os.WriteFile(filePath, []byte("preview-bytes"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	bridge := New(Config{}, log.New(io.Discard, "", 0))
	result := bridge.readFileBinary(map[string]any{
		"path": filePath,
	})

	contentsBase64, _ := result["contentsBase64"].(string)
	if contentsBase64 == "" {
		t.Fatalf("expected binary contents, got %#v", result)
	}
	contents, err := base64.StdEncoding.DecodeString(contentsBase64)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	if string(contents) != "preview-bytes" {
		t.Fatalf("unexpected binary contents: %q", string(contents))
	}
}

func TestResolveMobileResourceRequiresFullAccessForHomeFile(t *testing.T) {
	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	tempDir := filepath.Join(baseDir, "tmp")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("create home dir: %v", err)
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Setenv("HOME", homeDir)
	t.Setenv("TMPDIR", tempDir)

	filePath := filepath.Join(homeDir, "Desktop", "note.txt")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("create home dir: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("desktop-note"), 0o644); err != nil {
		t.Fatalf("write home file: %v", err)
	}

	bridge := New(Config{}, log.New(io.Discard, "", 0))
	result := bridge.resolveMobileResource(map[string]any{
		"path": filePath,
	})

	if supported, _ := result["supported"].(bool); supported {
		t.Fatalf("expected unsupported result without full access, got %#v", result)
	}

	bridge.performLocalDirectRPC("set-configuration", map[string]any{
		"key":   mobileFullDesktopFileAccessConfigKey,
		"value": true,
	})
	result = bridge.resolveMobileResource(map[string]any{
		"path": filePath,
	})

	if supported, _ := result["supported"].(bool); !supported {
		t.Fatalf("expected supported result with full access, got %#v", result)
	}
}

func TestListMobileDirectoryEntriesHonorsFullAccess(t *testing.T) {
	baseDir := t.TempDir()
	homeDir := filepath.Join(baseDir, "home")
	tempDir := filepath.Join(baseDir, "tmp")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("create home dir: %v", err)
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Setenv("HOME", homeDir)
	t.Setenv("TMPDIR", tempDir)

	downloadsDir := filepath.Join(homeDir, "Downloads")
	if err := os.MkdirAll(filepath.Join(downloadsDir, "Projects"), 0o755); err != nil {
		t.Fatalf("create downloads dirs: %v", err)
	}
	filePath := filepath.Join(downloadsDir, "capture.png")
	if err := os.WriteFile(filePath, []byte("capture"), 0o644); err != nil {
		t.Fatalf("write capture file: %v", err)
	}

	bridge := New(Config{}, log.New(io.Discard, "", 0))
	result := bridge.listMobileDirectoryEntries(map[string]any{
		"directoryPath": downloadsDir,
	})
	if errorMessage, _ := result["error"].(string); errorMessage == "" {
		t.Fatalf("expected full access error, got %#v", result)
	}

	bridge.performLocalDirectRPC("set-configuration", map[string]any{
		"key":   mobileFullDesktopFileAccessConfigKey,
		"value": true,
	})
	result = bridge.listMobileDirectoryEntries(map[string]any{
		"directoryPath": downloadsDir,
	})

	if errorMessage, _ := result["error"].(string); errorMessage != "" {
		t.Fatalf("unexpected directory list error: %#v", result)
	}
	resolvedDownloadsDir, err := filepath.EvalSymlinks(downloadsDir)
	if err != nil {
		t.Fatalf("resolve downloads dir: %v", err)
	}
	if directoryPath, _ := result["directoryPath"].(string); directoryPath != resolvedDownloadsDir {
		t.Fatalf("expected directory path %q, got %#v", resolvedDownloadsDir, result)
	}
	entries, ok := result["entries"].([]any)
	if !ok || len(entries) != 2 {
		t.Fatalf("expected two entries, got %#v", result)
	}
	firstEntry, _ := entries[0].(map[string]any)
	secondEntry, _ := entries[1].(map[string]any)
	if firstEntry["name"] != "Projects" || firstEntry["isDirectory"] != true {
		t.Fatalf("expected directories first, got %#v", entries)
	}
	resolvedProjectsDir, err := filepath.EvalSymlinks(filepath.Join(downloadsDir, "Projects"))
	if err != nil {
		t.Fatalf("resolve projects dir: %v", err)
	}
	if firstEntry["path"] != resolvedProjectsDir {
		t.Fatalf("expected resolved project path %q, got %#v", resolvedProjectsDir, entries)
	}
	if secondEntry["name"] != "capture.png" || secondEntry["isDirectory"] != false {
		t.Fatalf("expected file entry second, got %#v", entries)
	}
	resolvedFilePath, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		t.Fatalf("resolve capture file path: %v", err)
	}
	if secondEntry["path"] != resolvedFilePath {
		t.Fatalf("expected resolved file path %q, got %#v", resolvedFilePath, entries)
	}
}
