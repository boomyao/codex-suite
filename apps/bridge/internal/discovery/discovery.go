package discovery

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	runtimepkg "runtime"
	"strings"
	"time"
)

type CurrentRuntime struct {
	Channel     string `json:"channel"`
	Platform    string `json:"platform"`
	Version     string `json:"version"`
	RuntimeRoot string `json:"runtimeRoot"`
}

type Manifest struct {
	Version        string         `json:"version"`
	Platform       string         `json:"platform"`
	ImportedAt     string         `json:"importedAt"`
	DesktopWebview ManifestAsset  `json:"desktopWebview"`
	AppServer      ManifestBinary `json:"appServer"`
	Source         ManifestSource `json:"source"`
}

type ManifestAsset struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type ManifestBinary struct {
	Path   string   `json:"path"`
	SHA256 string   `json:"sha256"`
	Args   []string `json:"args,omitempty"`
}

type ManifestSource struct {
	Type                 string `json:"type"`
	DesktopWebviewSource string `json:"desktopWebviewSource,omitempty"`
	AppServerSource      string `json:"appServerSource,omitempty"`
}

type Options struct {
	DataDir                     string
	PreferredDesktopWebviewRoot string
	PreferredAppServerBin       string
	RequireAppServer            bool
	Logger                      *log.Logger
}

type Result struct {
	DataDir            string
	CurrentFile        string
	RuntimeRoot        string
	ManifestFile       string
	DesktopWebviewRoot string
	AppServerBin       string
	Imported           bool
}

type sourceCandidate struct {
	Path string
	Hash string
}

func EnsureRuntime(options Options) (Result, error) {
	logger := options.Logger
	if logger == nil {
		logger = log.Default()
	}

	dataDir := strings.TrimSpace(options.DataDir)
	if dataDir == "" {
		return Result{}, fmt.Errorf("data dir is required")
	}

	resolvedDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(resolvedDataDir, 0o755); err != nil {
		return Result{}, err
	}

	result := Result{
		DataDir:     resolvedDataDir,
		CurrentFile: filepath.Join(resolvedDataDir, "current.json"),
	}

	preferredWebview, err := describeDesktopWebviewSource(options.PreferredDesktopWebviewRoot)
	if err != nil {
		return result, err
	}
	preferredAppServer, err := describeExecutableSource(options.PreferredAppServerBin)
	if err != nil {
		return result, err
	}

	currentResult, currentManifest, ok := loadCurrentRuntime(result, options.RequireAppServer)
	if ok && currentMatchesPreferred(currentManifest, currentResult, preferredWebview, preferredAppServer, options.RequireAppServer) {
		return currentResult, nil
	}

	if preferredWebview.Path == "" && preferredAppServer.Path == "" {
		if ok {
			return currentResult, nil
		}
		return result, nil
	}
	if options.RequireAppServer && preferredAppServer.Path == "" {
		if ok {
			return currentResult, nil
		}
		return result, fmt.Errorf("no app-server executable available for managed mode")
	}

	imported, err := importRuntime(result, preferredWebview, preferredAppServer)
	if err != nil {
		return result, err
	}

	if imported.Imported {
		logger.Printf("%s [codex-bridge-discovery] imported local runtime %s", nowISO(), imported.RuntimeRoot)
		if imported.DesktopWebviewRoot != "" {
			logger.Printf("%s [codex-bridge-discovery] desktop-webview %s", nowISO(), imported.DesktopWebviewRoot)
		}
		if imported.AppServerBin != "" {
			logger.Printf("%s [codex-bridge-discovery] app-server %s", nowISO(), imported.AppServerBin)
		}
	}

	return imported, nil
}

func loadCurrentRuntime(base Result, requireAppServer bool) (Result, Manifest, bool) {
	raw, err := os.ReadFile(base.CurrentFile)
	if err != nil {
		return Result{}, Manifest{}, false
	}

	var current CurrentRuntime
	if err := json.Unmarshal(raw, &current); err != nil {
		return Result{}, Manifest{}, false
	}

	runtimeRoot := strings.TrimSpace(current.RuntimeRoot)
	if runtimeRoot == "" {
		return Result{}, Manifest{}, false
	}
	resolvedRuntimeRoot, err := filepath.Abs(runtimeRoot)
	if err != nil {
		return Result{}, Manifest{}, false
	}

	manifestPath := filepath.Join(resolvedRuntimeRoot, "manifest.json")
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		return Result{}, Manifest{}, false
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		return Result{}, Manifest{}, false
	}

	result := base
	result.RuntimeRoot = resolvedRuntimeRoot
	result.ManifestFile = manifestPath
	if manifest.DesktopWebview.Path != "" {
		webviewRoot := filepath.Join(resolvedRuntimeRoot, filepath.FromSlash(manifest.DesktopWebview.Path))
		if validDesktopWebviewRoot(webviewRoot) {
			result.DesktopWebviewRoot = webviewRoot
		}
	}
	if manifest.AppServer.Path != "" {
		appServerBin := filepath.Join(resolvedRuntimeRoot, filepath.FromSlash(manifest.AppServer.Path))
		if isExecutable(appServerBin) {
			result.AppServerBin = appServerBin
		}
	}
	if requireAppServer && result.AppServerBin == "" {
		return Result{}, Manifest{}, false
	}
	return result, manifest, true
}

func currentMatchesPreferred(manifest Manifest, current Result, preferredWebview sourceCandidate, preferredAppServer sourceCandidate, requireAppServer bool) bool {
	if requireAppServer && current.AppServerBin == "" {
		return false
	}

	if preferredWebview.Path != "" {
		if current.DesktopWebviewRoot == "" {
			return false
		}
		if preferredWebview.Hash != "" && manifest.DesktopWebview.SHA256 != "" && manifest.DesktopWebview.SHA256 != preferredWebview.Hash {
			return false
		}
	}

	if preferredAppServer.Path != "" {
		if current.AppServerBin == "" {
			return false
		}
		if preferredAppServer.Hash != "" && manifest.AppServer.SHA256 != "" && manifest.AppServer.SHA256 != preferredAppServer.Hash {
			return false
		}
	}

	return true
}

func importRuntime(base Result, webview sourceCandidate, appServer sourceCandidate) (Result, error) {
	platform := runtimePlatform()
	version := "local-" + time.Now().UTC().Format("20060102T150405Z")
	runtimeRoot := filepath.Join(base.DataDir, "runtimes", platform, version)
	tempRoot := runtimeRoot + ".tmp"
	_ = os.RemoveAll(tempRoot)
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return base, err
	}

	manifest := Manifest{
		Version:    version,
		Platform:   platform,
		ImportedAt: time.Now().UTC().Format(time.RFC3339),
		Source: ManifestSource{
			Type: "local-import",
		},
	}

	result := base
	result.RuntimeRoot = runtimeRoot
	result.ManifestFile = filepath.Join(runtimeRoot, "manifest.json")
	result.Imported = true

	if webview.Path != "" {
		dst := filepath.Join(tempRoot, "desktop-webview")
		if err := copyDirectory(webview.Path, dst); err != nil {
			return base, err
		}
		manifest.DesktopWebview = ManifestAsset{
			Path:   filepath.ToSlash("desktop-webview"),
			SHA256: webview.Hash,
		}
		manifest.Source.DesktopWebviewSource = webview.Path
		result.DesktopWebviewRoot = filepath.Join(runtimeRoot, "desktop-webview")
	}

	if appServer.Path != "" {
		binName := filepath.Base(appServer.Path)
		if binName == "." || binName == string(filepath.Separator) || binName == "" {
			binName = "codex"
		}
		dst := filepath.Join(tempRoot, "bin", binName)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return base, err
		}
		if err := copyFile(appServer.Path, dst); err != nil {
			return base, err
		}
		manifest.AppServer = ManifestBinary{
			Path:   filepath.ToSlash(filepath.Join("bin", binName)),
			SHA256: appServer.Hash,
		}
		manifest.Source.AppServerSource = appServer.Path
		result.AppServerBin = filepath.Join(runtimeRoot, "bin", binName)
	}

	if err := writeJSONAtomic(filepath.Join(tempRoot, "manifest.json"), manifest); err != nil {
		return base, err
	}
	if err := os.MkdirAll(filepath.Dir(runtimeRoot), 0o755); err != nil {
		return base, err
	}
	if err := os.Rename(tempRoot, runtimeRoot); err != nil {
		return base, err
	}

	current := CurrentRuntime{
		Channel:     "local",
		Platform:    platform,
		Version:     version,
		RuntimeRoot: runtimeRoot,
	}
	if err := writeJSONAtomic(base.CurrentFile, current); err != nil {
		return base, err
	}
	return result, nil
}

func describeDesktopWebviewSource(root string) (sourceCandidate, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return sourceCandidate{}, nil
	}
	resolvedRoot, err := filepath.Abs(root)
	if err != nil {
		return sourceCandidate{}, err
	}
	if !validDesktopWebviewRoot(resolvedRoot) {
		return sourceCandidate{}, nil
	}
	hash, err := hashDirectory(resolvedRoot)
	if err != nil {
		return sourceCandidate{}, err
	}
	return sourceCandidate{
		Path: resolvedRoot,
		Hash: hash,
	}, nil
}

func describeExecutableSource(path string) (sourceCandidate, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return sourceCandidate{}, nil
	}
	resolvedPath, err := filepath.Abs(path)
	if err != nil {
		return sourceCandidate{}, err
	}
	if !isExecutable(resolvedPath) {
		return sourceCandidate{}, nil
	}
	hash, err := hashFile(resolvedPath)
	if err != nil {
		return sourceCandidate{}, err
	}
	return sourceCandidate{
		Path: resolvedPath,
		Hash: hash,
	}, nil
}

func validDesktopWebviewRoot(root string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return false
	}
	indexPath := filepath.Join(root, "index.html")
	indexInfo, err := os.Stat(indexPath)
	return err == nil && !indexInfo.IsDir()
}

func copyDirectory(sourceRoot string, destinationRoot string) error {
	return filepath.WalkDir(sourceRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(destinationRoot, relativePath)

		if entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(targetPath, info.Mode().Perm())
		}

		if entry.Type()&os.ModeSymlink != 0 {
			resolvedPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			return copyFile(resolvedPath, targetPath)
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(sourcePath string, destinationPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return err
	}

	targetFile, err := os.OpenFile(destinationPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return err
	}
	return targetFile.Chmod(info.Mode().Perm())
}

func hashDirectory(root string) (string, error) {
	hasher := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if _, err := hasher.Write([]byte(filepath.ToSlash(relativePath))); err != nil {
			return err
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		if _, err := io.Copy(hasher, file); err != nil {
			return err
		}
		if _, err := hasher.Write([]byte{0}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func writeJSONAtomic(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')

	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, body, 0o644); err != nil {
		return err
	}
	return os.Rename(tempPath, path)
}

func runtimePlatform() string {
	return runtimepkg.GOOS + "-" + runtimepkg.GOARCH
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Mode()&0o111 != 0
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
