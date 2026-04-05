package bridge

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const mobileResolvedResourceTTL = 10 * time.Minute
const mobilePickedFilesDirName = "codex-mobile-picked-files"
const mobileFullDesktopFileAccessConfigKey = "mobile.fullDesktopFileAccess"

type resolvedMobileResource struct {
	Path      string
	Name      string
	MIMEType  string
	Size      int64
	Kind      string
	ExpiresAt time.Time
}

func (b *Bridge) mobileHostCapabilities() map[string]any {
	fullDesktopFileAccessEnabled := b.mobileFullDesktopFileAccessEnabled()
	return map[string]any{
		"attachmentPick": map[string]any{
			"device":  false,
			"desktop": fullDesktopFileAccessEnabled,
		},
		"resourceResolve": true,
		"workspaceBrowse": fullDesktopFileAccessEnabled,
		"terminalStream":  false,
	}
}

func (b *Bridge) resolveMobileResource(params map[string]any) map[string]any {
	resourcePath, ok, err := b.resolveMobileResourcePath(params)
	if err != nil {
		return map[string]any{
			"supported": false,
			"error":     err.Error(),
		}
	}
	if !ok {
		return map[string]any{
			"supported": false,
			"error":     "A local file path inside an allowed desktop root is required.",
		}
	}

	info, err := os.Stat(resourcePath)
	if err != nil {
		return map[string]any{
			"supported": false,
			"error":     fmt.Sprintf("Failed to stat resource: %v", err),
		}
	}
	if info.IsDir() {
		return map[string]any{
			"supported": false,
			"error":     "Directories cannot be resolved as mobile resources.",
		}
	}

	resourceID, err := randomMobileResourceID()
	if err != nil {
		return map[string]any{
			"supported": false,
			"error":     fmt.Sprintf("Failed to allocate resource handle: %v", err),
		}
	}

	name := filepath.Base(resourcePath)
	mimeType := guessContentType(resourcePath)
	kind := "file"
	if strings.HasPrefix(strings.ToLower(mimeType), "image/") {
		kind = "image"
	}
	expiresAt := time.Now().UTC().Add(mobileResolvedResourceTTL)

	b.mu.Lock()
	b.pruneResolvedMobileResourcesLocked(time.Now().UTC())
	if b.resolvedMobileResources == nil {
		b.resolvedMobileResources = map[string]resolvedMobileResource{}
	}
	b.resolvedMobileResources[resourceID] = resolvedMobileResource{
		Path:      resourcePath,
		Name:      name,
		MIMEType:  mimeType,
		Size:      info.Size(),
		Kind:      kind,
		ExpiresAt: expiresAt,
	}
	b.mu.Unlock()

	return map[string]any{
		"supported": true,
		"handle": map[string]any{
			"id":       resourceID,
			"origin":   "desktop",
			"kind":     kind,
			"name":     name,
			"mimeType": mimeType,
			"size":     info.Size(),
		},
		"url":       "/codex-mobile/resource/" + url.PathEscape(resourceID),
		"expiresAt": expiresAt.Format(time.RFC3339),
	}
}

func (b *Bridge) uploadPickedMobileFile(params map[string]any) map[string]any {
	name, _ := stringParam(params, "name")
	name = sanitizedPickedFileName(name)

	contentsBase64, _ := stringParam(params, "contentsBase64")
	contentsBase64 = strings.TrimSpace(contentsBase64)
	if contentsBase64 == "" {
		return map[string]any{
			"error": "Picked file contents are required.",
		}
	}

	contents, err := base64.StdEncoding.DecodeString(contentsBase64)
	if err != nil {
		return map[string]any{
			"error": "Picked file contents were not valid base64.",
		}
	}

	dirPath := filepath.Join(os.TempDir(), mobilePickedFilesDirName)
	if err := os.MkdirAll(dirPath, 0o700); err != nil {
		return map[string]any{
			"error": fmt.Sprintf("Failed to prepare picked files directory: %v", err),
		}
	}

	resourceID, err := randomMobileResourceID()
	if err != nil {
		return map[string]any{
			"error": fmt.Sprintf("Failed to allocate picked file handle: %v", err),
		}
	}

	extension := filepath.Ext(name)
	baseName := strings.TrimSuffix(name, extension)
	if strings.TrimSpace(baseName) == "" {
		baseName = "attachment"
	}
	filePath := filepath.Join(dirPath, baseName+"-"+resourceID+extension)
	if err := os.WriteFile(filePath, contents, 0o600); err != nil {
		return map[string]any{
			"error": fmt.Sprintf("Failed to write picked file: %v", err),
		}
	}

	return map[string]any{
		"label":  name,
		"path":   filePath,
		"fsPath": filePath,
	}
}

func (b *Bridge) handleMobileResource(w http.ResponseWriter, r *http.Request) bool {
	const resourcePathPrefix = "/codex-mobile/resource/"
	if !strings.HasPrefix(r.URL.Path, resourcePathPrefix) {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
		return true
	}

	resourceID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, resourcePathPrefix))
	if resourceID == "" || strings.Contains(resourceID, "/") {
		sendText(w, http.StatusNotFound, "Not Found\n")
		return true
	}

	now := time.Now().UTC()
	b.mu.Lock()
	b.pruneResolvedMobileResourcesLocked(now)
	resource, ok := b.resolvedMobileResources[resourceID]
	b.mu.Unlock()
	if !ok || now.After(resource.ExpiresAt) {
		sendText(w, http.StatusNotFound, "Not Found\n")
		return true
	}

	file, err := os.Open(resource.Path)
	if err != nil {
		sendText(w, http.StatusNotFound, "Not Found\n")
		return true
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		sendText(w, http.StatusNotFound, "Not Found\n")
		return true
	}

	w.Header().Set("Content-Type", resource.MIMEType)
	w.Header().Set("Cache-Control", "private, max-age=300")
	http.ServeContent(w, r, resource.Name, info.ModTime(), file)
	return true
}

func (b *Bridge) readFileBinary(params map[string]any) map[string]any {
	filePath, ok, err := b.resolveMobileResourcePath(params)
	if err != nil || !ok {
		return map[string]any{
			"contentsBase64": nil,
		}
	}
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() || !info.Mode().IsRegular() {
		return map[string]any{
			"contentsBase64": nil,
		}
	}

	contents, err := os.ReadFile(filePath)
	if err != nil {
		return map[string]any{
			"contentsBase64": nil,
		}
	}

	return map[string]any{
		"contentsBase64": base64.StdEncoding.EncodeToString(contents),
	}
}

func (b *Bridge) listMobileDirectoryEntries(params map[string]any) map[string]any {
	directoryPath, ok, err := b.resolveMobileDirectoryPath(params)
	if err != nil {
		return map[string]any{
			"directoryPath": directoryPath,
			"entries":       []any{},
			"error":         err.Error(),
		}
	}
	if !ok {
		return map[string]any{
			"directoryPath": directoryPath,
			"entries":       []any{},
			"error":         "A desktop directory inside an allowed root is required.",
		}
	}

	info, err := os.Stat(directoryPath)
	if err != nil {
		return map[string]any{
			"directoryPath": directoryPath,
			"entries":       []any{},
			"error":         fmt.Sprintf("Failed to stat directory: %v", err),
		}
	}
	if !info.IsDir() {
		return map[string]any{
			"directoryPath": directoryPath,
			"entries":       []any{},
			"error":         "Only directories can be browsed.",
		}
	}

	rawEntries, err := os.ReadDir(directoryPath)
	if err != nil {
		return map[string]any{
			"directoryPath": directoryPath,
			"entries":       []any{},
			"error":         fmt.Sprintf("Failed to read directory: %v", err),
		}
	}

	directoriesOnly, _ := boolParam(params, "directoriesOnly")
	type mobileDirectoryEntry struct {
		Name        string
		Path        string
		IsDirectory bool
	}
	entries := make([]mobileDirectoryEntry, 0, len(rawEntries))
	for _, entry := range rawEntries {
		entryPath := filepath.Join(directoryPath, entry.Name())
		resolvedEntryPath := entryPath
		if evaluatedPath, evalErr := filepath.EvalSymlinks(entryPath); evalErr == nil && strings.TrimSpace(evaluatedPath) != "" {
			resolvedEntryPath = evaluatedPath
		}

		entryInfo, err := os.Stat(resolvedEntryPath)
		if err != nil {
			continue
		}
		if entryInfo.IsDir() {
			entries = append(entries, mobileDirectoryEntry{
				Name:        entry.Name(),
				Path:        resolvedEntryPath,
				IsDirectory: true,
			})
			continue
		}
		if directoriesOnly || !entryInfo.Mode().IsRegular() {
			continue
		}
		entries = append(entries, mobileDirectoryEntry{
			Name:        entry.Name(),
			Path:        resolvedEntryPath,
			IsDirectory: false,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDirectory != entries[j].IsDirectory {
			return entries[i].IsDirectory
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})

	serializedEntries := make([]any, 0, len(entries))
	for _, entry := range entries {
		serializedEntries = append(serializedEntries, map[string]any{
			"name":        entry.Name,
			"path":        entry.Path,
			"isDirectory": entry.IsDirectory,
		})
	}

	return map[string]any{
		"directoryPath": directoryPath,
		"entries":       serializedEntries,
	}
}

func (b *Bridge) resolveMobileResourcePath(params map[string]any) (string, bool, error) {
	resolvedPath, ok, err := b.normalizeMobileFilesystemPath(params, "path", "uri")
	if err != nil || !ok {
		return "", ok, err
	}
	if b.mobileFullDesktopFileAccessEnabled() {
		return resolvedPath, true, nil
	}
	for _, root := range b.allowedMobileResourceRoots() {
		if resolvedPath == root || isPathInside(root, resolvedPath) {
			return resolvedPath, true, nil
		}
	}
	return "", false, nil
}

func (b *Bridge) resolveMobileDirectoryPath(params map[string]any) (string, bool, error) {
	rawPath, _ := stringParam(params, "directoryPath")
	if strings.TrimSpace(rawPath) == "" {
		rawPath = b.defaultMobileBrowseDirectory()
	}
	if strings.TrimSpace(rawPath) == "" {
		return "", false, fmt.Errorf("No desktop directory is available to browse.")
	}
	resolvedPath, ok, err := b.normalizeMobileFilesystemPath(map[string]any{"path": rawPath}, "path")
	if err != nil || !ok {
		return resolvedPath, ok, err
	}
	if b.mobileFullDesktopFileAccessEnabled() {
		return resolvedPath, true, nil
	}
	for _, root := range b.allowedMobileResourceRoots() {
		if resolvedPath == root || isPathInside(root, resolvedPath) {
			return resolvedPath, true, nil
		}
	}
	return resolvedPath, false, nil
}

func (b *Bridge) normalizeMobileFilesystemPath(params map[string]any, preferredKeys ...string) (string, bool, error) {
	var rawPath string
	for _, key := range preferredKeys {
		switch key {
		case "path", "directoryPath":
			if value, ok := stringParam(params, key); ok && strings.TrimSpace(value) != "" {
				rawPath = value
				break
			}
		case "uri":
			if value, ok := stringParam(params, key); ok && strings.TrimSpace(value) != "" {
				parsed, err := url.Parse(strings.TrimSpace(value))
				if err != nil {
					return "", false, fmt.Errorf("Invalid resource URI.")
				}
				if !strings.EqualFold(parsed.Scheme, "file") {
					return "", false, fmt.Errorf("Only file:// resource URIs are supported.")
				}
				rawPath = parsed.Path
				break
			}
		}
		if strings.TrimSpace(rawPath) != "" {
			break
		}
	}
	if strings.TrimSpace(rawPath) == "" {
		return "", false, nil
	}
	if !filepath.IsAbs(rawPath) {
		return "", false, fmt.Errorf("Only absolute desktop file paths are supported.")
	}
	resolvedPath, err := filepath.Abs(rawPath)
	if err != nil {
		return "", false, fmt.Errorf("Failed to resolve desktop file path.")
	}
	if evaluatedPath, evalErr := filepath.EvalSymlinks(resolvedPath); evalErr == nil && strings.TrimSpace(evaluatedPath) != "" {
		resolvedPath = evaluatedPath
	}
	return resolvedPath, true, nil
}

func (b *Bridge) allowedMobileResourceRoots() []string {
	roots, activeRoots, _ := b.workspaceRootOptions()
	allowed := make([]string, 0, len(roots)+len(activeRoots)+2)
	allowed = append(allowed, deriveCodexHome(), os.TempDir())
	allowed = append(allowed, roots...)
	allowed = append(allowed, activeRoots...)

	seen := map[string]struct{}{}
	next := make([]string, 0, len(allowed))
	for _, root := range allowed {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}
		resolvedRoot, err := filepath.Abs(trimmed)
		if err != nil {
			continue
		}
		if evaluatedRoot, evalErr := filepath.EvalSymlinks(resolvedRoot); evalErr == nil && strings.TrimSpace(evaluatedRoot) != "" {
			resolvedRoot = evaluatedRoot
		}
		if _, exists := seen[resolvedRoot]; exists {
			continue
		}
		seen[resolvedRoot] = struct{}{}
		next = append(next, resolvedRoot)
	}
	return next
}

func (b *Bridge) defaultMobileBrowseDirectory() string {
	candidates := []string{}
	if b.mobileFullDesktopFileAccessEnabled() {
		if homeDir, err := os.UserHomeDir(); err == nil && strings.TrimSpace(homeDir) != "" {
			candidates = append(candidates, homeDir)
		}
	}

	roots, activeRoots, _ := b.workspaceRootOptions()
	candidates = append(candidates, activeRoots...)
	candidates = append(candidates, roots...)
	candidates = append(candidates, deriveCodexHome(), os.TempDir())

	for _, candidate := range uniqueNonEmptyResolvedPaths(candidates) {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func (b *Bridge) mobileFullDesktopFileAccessEnabled() bool {
	b.localState.mu.Lock()
	defer b.localState.mu.Unlock()
	enabled, _ := b.localState.configurationState[mobileFullDesktopFileAccessConfigKey].(bool)
	return enabled
}

func uniqueNonEmptyResolvedPaths(paths []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		resolvedPath, err := filepath.Abs(trimmed)
		if err != nil {
			continue
		}
		if evaluatedPath, evalErr := filepath.EvalSymlinks(resolvedPath); evalErr == nil && strings.TrimSpace(evaluatedPath) != "" {
			resolvedPath = evaluatedPath
		}
		if _, exists := seen[resolvedPath]; exists {
			continue
		}
		seen[resolvedPath] = struct{}{}
		result = append(result, resolvedPath)
	}
	return result
}

func (b *Bridge) pruneResolvedMobileResourcesLocked(now time.Time) {
	if len(b.resolvedMobileResources) == 0 {
		return
	}
	for resourceID, resource := range b.resolvedMobileResources {
		if now.After(resource.ExpiresAt) {
			delete(b.resolvedMobileResources, resourceID)
		}
	}
}

func randomMobileResourceID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func sanitizedPickedFileName(value string) string {
	name := strings.TrimSpace(filepath.Base(value))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "attachment"
	}
	return name
}
