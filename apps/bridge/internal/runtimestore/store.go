package runtimestore

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	runtimepkg "runtime"
	"strings"
	"time"
)

type BridgePointerManifest struct {
	SchemaVersion       int             `json:"schemaVersion"`
	BridgeVersion       string          `json:"bridgeVersion"`
	Platform            string          `json:"platform"`
	RuntimeRepo         string          `json:"runtimeRepo"`
	RuntimeID           string          `json:"runtimeId"`
	RuntimeTag          string          `json:"runtimeTag"`
	RuntimeVersion      string          `json:"runtimeVersion"`
	RuntimeAsset        string          `json:"runtimeAsset"`
	RuntimeManifestAsset string         `json:"runtimeManifestAsset"`
	PatchSchemaVersion  int             `json:"patchSchemaVersion"`
	UpstreamRelease     UpstreamRelease `json:"upstreamRelease"`
	DesktopWebview      RuntimeAsset    `json:"desktopWebview"`
	AppServer           RuntimeAsset    `json:"appServer"`
}

type RuntimeManifest struct {
	SchemaVersion      int             `json:"schemaVersion"`
	RuntimeID          string          `json:"runtimeId"`
	RuntimeVersion     string          `json:"runtimeVersion"`
	Platform           string          `json:"platform"`
	PatchSchemaVersion int             `json:"patchSchemaVersion"`
	UpstreamRelease    UpstreamRelease `json:"upstreamRelease"`
	DesktopWebview     RuntimeAsset    `json:"desktopWebview"`
	AppServer          RuntimeAsset    `json:"appServer"`
}

type UpstreamRelease struct {
	Repo string `json:"repo"`
	Tag  string `json:"tag"`
}

type RuntimeAsset struct {
	Source string `json:"source"`
	Version string `json:"version"`
	Asset  string `json:"asset,omitempty"`
	Path   string `json:"path,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

type ActivatedRuntime struct {
	RuntimeRoot        string
	ManifestFile       string
	DesktopWebviewRoot string
	AppServerBin       string
	Manifest           RuntimeManifest
}

type Options struct {
	DataDir                  string
	PointerManifestCandidates []string
	LocalManifestCandidates  []string
	LocalArchiveCandidates   []string
	RequireAppServer         bool
	Logger                   *log.Logger
}

type githubReleaseMetadata struct {
	Assets []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

func EnsureRuntime(options Options) (ActivatedRuntime, bool, error) {
	logger := options.Logger
	if logger == nil {
		logger = log.Default()
	}

	dataDir := strings.TrimSpace(options.DataDir)
	if dataDir == "" {
		return ActivatedRuntime{}, false, fmt.Errorf("data dir is required")
	}

	resolvedDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return ActivatedRuntime{}, false, err
	}
	if err := os.MkdirAll(resolvedDataDir, 0o755); err != nil {
		return ActivatedRuntime{}, false, err
	}

	if manifestPath, archivePath, ok, err := resolveLocalRuntimeCandidates(options.LocalManifestCandidates, options.LocalArchiveCandidates); err != nil {
		return ActivatedRuntime{}, false, err
	} else if ok {
		manifest, err := loadRuntimeManifest(manifestPath)
		if err != nil {
			return ActivatedRuntime{}, false, err
		}
		activated, err := ensureActivatedRuntime(resolvedDataDir, manifest, archivePath, options.RequireAppServer)
		if err != nil {
			return ActivatedRuntime{}, false, err
		}
		logger.Printf("%s [codex-bridge-runtime-store] activated local packaged runtime %s", nowISO(), activated.RuntimeRoot)
		return activated, true, nil
	}

	pointerPath, ok, err := resolveFirstExistingFile(options.PointerManifestCandidates)
	if err != nil {
		return ActivatedRuntime{}, false, err
	}
	if !ok {
		return ActivatedRuntime{}, false, nil
	}

	pointer, err := loadPointerManifest(pointerPath)
	if err != nil {
		return ActivatedRuntime{}, false, err
	}
	if pointer.Platform != "" && pointer.Platform != runtimePlatform() {
		return ActivatedRuntime{}, false, fmt.Errorf("runtime pointer platform %q does not match host platform %q", pointer.Platform, runtimePlatform())
	}

	downloadDir := filepath.Join(resolvedDataDir, "downloads", pointer.RuntimeTag)
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		return ActivatedRuntime{}, false, err
	}
	manifestPath := filepath.Join(downloadDir, pointer.RuntimeManifestAsset)
	archivePath := filepath.Join(downloadDir, pointer.RuntimeAsset)

	manifest, err := ensureDownloadedRuntime(pointer, manifestPath, archivePath)
	if err != nil {
		return ActivatedRuntime{}, false, err
	}

	activated, err := ensureActivatedRuntime(resolvedDataDir, manifest, archivePath, options.RequireAppServer)
	if err != nil {
		return ActivatedRuntime{}, false, err
	}
	logger.Printf("%s [codex-bridge-runtime-store] activated release runtime %s", nowISO(), activated.RuntimeRoot)
	return activated, true, nil
}

func ensureDownloadedRuntime(pointer BridgePointerManifest, manifestPath, archivePath string) (RuntimeManifest, error) {
	if manifest, err := loadRuntimeManifest(manifestPath); err == nil && validArchivePath(archivePath) {
		if manifest.RuntimeID == pointer.RuntimeID && manifest.RuntimeVersion == pointer.RuntimeVersion {
			return manifest, nil
		}
	}

	metadata, err := fetchGitHubReleaseMetadata(pointer.RuntimeRepo, pointer.RuntimeTag)
	if err != nil {
		return RuntimeManifest{}, err
	}

	manifestURL, err := findReleaseAssetURL(metadata, pointer.RuntimeManifestAsset)
	if err != nil {
		return RuntimeManifest{}, err
	}
	archiveURL, err := findReleaseAssetURL(metadata, pointer.RuntimeAsset)
	if err != nil {
		return RuntimeManifest{}, err
	}

	if err := downloadFile(manifestURL, manifestPath); err != nil {
		return RuntimeManifest{}, err
	}
	if err := downloadFile(archiveURL, archivePath); err != nil {
		return RuntimeManifest{}, err
	}

	manifest, err := loadRuntimeManifest(manifestPath)
	if err != nil {
		return RuntimeManifest{}, err
	}
	if manifest.RuntimeID != pointer.RuntimeID || manifest.RuntimeVersion != pointer.RuntimeVersion {
		return RuntimeManifest{}, fmt.Errorf("downloaded runtime manifest does not match bridge pointer")
	}
	return manifest, nil
}

func ensureActivatedRuntime(dataDir string, manifest RuntimeManifest, archivePath string, requireAppServer bool) (ActivatedRuntime, error) {
	if strings.TrimSpace(manifest.RuntimeID) == "" {
		return ActivatedRuntime{}, fmt.Errorf("runtime manifest missing runtimeId")
	}
	platform := strings.TrimSpace(manifest.Platform)
	if platform == "" {
		platform = runtimePlatform()
	}
	runtimeRoot := filepath.Join(dataDir, "runtimes", platform, manifest.RuntimeID)
	manifestFile := filepath.Join(runtimeRoot, "manifest.json")

	if activated, ok := loadActivatedRuntime(runtimeRoot, manifestFile, requireAppServer); ok {
		if activated.Manifest.RuntimeID == manifest.RuntimeID && activated.Manifest.RuntimeVersion == manifest.RuntimeVersion {
			return activated, nil
		}
	}

	tempRoot := runtimeRoot + ".tmp"
	_ = os.RemoveAll(tempRoot)
	if err := os.MkdirAll(tempRoot, 0o755); err != nil {
		return ActivatedRuntime{}, err
	}

	if err := extractTarGz(archivePath, tempRoot); err != nil {
		_ = os.RemoveAll(tempRoot)
		return ActivatedRuntime{}, err
	}

	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = os.RemoveAll(tempRoot)
		return ActivatedRuntime{}, err
	}
	body = append(body, '\n')
	if err := os.WriteFile(filepath.Join(tempRoot, "manifest.json"), body, 0o644); err != nil {
		_ = os.RemoveAll(tempRoot)
		return ActivatedRuntime{}, err
	}

	_ = os.RemoveAll(runtimeRoot)
	if err := os.MkdirAll(filepath.Dir(runtimeRoot), 0o755); err != nil {
		_ = os.RemoveAll(tempRoot)
		return ActivatedRuntime{}, err
	}
	if err := os.Rename(tempRoot, runtimeRoot); err != nil {
		_ = os.RemoveAll(tempRoot)
		return ActivatedRuntime{}, err
	}

	activated, ok := loadActivatedRuntime(runtimeRoot, manifestFile, requireAppServer)
	if !ok {
		return ActivatedRuntime{}, fmt.Errorf("activated runtime is invalid: %s", runtimeRoot)
	}
	return activated, nil
}

func loadActivatedRuntime(runtimeRoot, manifestFile string, requireAppServer bool) (ActivatedRuntime, bool) {
	manifest, err := loadRuntimeManifest(manifestFile)
	if err != nil {
		return ActivatedRuntime{}, false
	}
	webviewRoot := filepath.Join(runtimeRoot, filepath.FromSlash(strings.TrimSpace(manifest.DesktopWebview.Path)))
	if !validDesktopWebviewRoot(webviewRoot) {
		return ActivatedRuntime{}, false
	}
	appServerBin := ""
	if strings.TrimSpace(manifest.AppServer.Path) != "" {
		candidate := filepath.Join(runtimeRoot, filepath.FromSlash(strings.TrimSpace(manifest.AppServer.Path)))
		if isExecutable(candidate) {
			appServerBin = candidate
		}
	}
	if requireAppServer && appServerBin == "" {
		return ActivatedRuntime{}, false
	}
	return ActivatedRuntime{
		RuntimeRoot:        runtimeRoot,
		ManifestFile:       manifestFile,
		DesktopWebviewRoot: webviewRoot,
		AppServerBin:       appServerBin,
		Manifest:           manifest,
	}, true
}

func resolveLocalRuntimeCandidates(manifestCandidates, archiveCandidates []string) (string, string, bool, error) {
	manifestPath, manifestOK, err := resolveFirstExistingFile(manifestCandidates)
	if err != nil || !manifestOK {
		return "", "", false, err
	}
	archivePath, archiveOK, err := resolveFirstExistingFile(archiveCandidates)
	if err != nil || !archiveOK {
		return "", "", false, err
	}
	return manifestPath, archivePath, true, nil
}

func resolveFirstExistingFile(candidates []string) (string, bool, error) {
	for _, candidate := range candidates {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "" {
			continue
		}
		resolved, err := filepath.Abs(trimmed)
		if err != nil {
			return "", false, err
		}
		info, err := os.Stat(resolved)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", false, err
		}
		if info.IsDir() {
			continue
		}
		return resolved, true, nil
	}
	return "", false, nil
}

func loadPointerManifest(path string) (BridgePointerManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return BridgePointerManifest{}, err
	}
	var manifest BridgePointerManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return BridgePointerManifest{}, err
	}
	return manifest, nil
}

func loadRuntimeManifest(path string) (RuntimeManifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return RuntimeManifest{}, err
	}
	var manifest RuntimeManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return RuntimeManifest{}, err
	}
	return manifest, nil
}

func fetchGitHubReleaseMetadata(repo, tag string) (githubReleaseMetadata, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", strings.TrimSpace(repo), strings.TrimSpace(tag))
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return githubReleaseMetadata{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "codex-bridge")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	} else if token := strings.TrimSpace(os.Getenv("GH_TOKEN")); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return githubReleaseMetadata{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return githubReleaseMetadata{}, fmt.Errorf("github release metadata request failed: %s %s", response.Status, strings.TrimSpace(string(body)))
	}

	var metadata githubReleaseMetadata
	if err := json.NewDecoder(response.Body).Decode(&metadata); err != nil {
		return githubReleaseMetadata{}, err
	}
	return metadata, nil
}

func findReleaseAssetURL(metadata githubReleaseMetadata, assetName string) (string, error) {
	for _, asset := range metadata.Assets {
		if asset.Name == assetName && strings.TrimSpace(asset.BrowserDownloadURL) != "" {
			return asset.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("release asset %q not found", assetName)
}

func downloadFile(url, destination string) error {
	tempPath := destination + ".tmp"
	_ = os.Remove(tempPath)

	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "codex-bridge")
	if token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	} else if token := strings.TrimSpace(os.Getenv("GH_TOKEN")); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 0}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return fmt.Errorf("download failed: %s %s", response.Status, strings.TrimSpace(string(body)))
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(tempPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, response.Body); err != nil {
		file.Close()
		_ = os.Remove(tempPath)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, destination); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

func extractTarGz(archivePath, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		cleanName := filepath.Clean(header.Name)
		if cleanName == "." || cleanName == string(filepath.Separator) {
			continue
		}
		targetPath := filepath.Join(destination, cleanName)
		if !strings.HasPrefix(targetPath, destination+string(filepath.Separator)) && targetPath != destination {
			return fmt.Errorf("invalid tar entry path %q", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			fileMode := os.FileMode(header.Mode)
			if fileMode == 0 {
				fileMode = 0o644
			}
			targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, fileMode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(targetFile, tarReader); err != nil {
				targetFile.Close()
				return err
			}
			if err := targetFile.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, targetPath); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
		}
	}
}

func validDesktopWebviewRoot(root string) bool {
	info, err := os.Stat(filepath.Join(root, "index.html"))
	return err == nil && !info.IsDir()
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}

func validArchivePath(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.Size() > 0
}

func runtimePlatform() string {
	arch := runtimepkg.GOARCH
	if arch == "amd64" {
		arch = "x64"
	}
	return runtimepkg.GOOS + "-" + arch
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
