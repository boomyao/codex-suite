package runtimestore

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureRuntimeActivatesLocalPackagedRuntime(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	packageRoot := filepath.Join(tempDir, "package-root")
	if err := os.MkdirAll(filepath.Join(packageRoot, "desktop-webview"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(packageRoot, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "desktop-webview", "index.html"), []byte("<!doctype html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "bin", "codex"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := RuntimeManifest{
		SchemaVersion:      1,
		RuntimeID:          "test-runtime",
		RuntimeVersion:     "0.0.1",
		Platform:           runtimePlatform(),
		PatchSchemaVersion: 1,
		DesktopWebview: RuntimeAsset{
			Source:  "test",
			Version: "0.0.1",
			Asset:   "runtime.tar.gz",
			Path:    "desktop-webview",
		},
		AppServer: RuntimeAsset{
			Source:  "test",
			Version: "0.0.1",
			Asset:   "runtime.tar.gz",
			Path:    "bin/codex",
		},
	}

	manifestPath := filepath.Join(tempDir, "runtime-manifest-"+runtimePlatform()+".json")
	manifestBody, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestBody = append(manifestBody, '\n')
	if err := os.WriteFile(manifestPath, manifestBody, 0o644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(tempDir, "runtime-"+runtimePlatform()+".tar.gz")
	if err := writeTarGz(archivePath, packageRoot); err != nil {
		t.Fatal(err)
	}

	activated, ok, err := EnsureRuntime(Options{
		DataDir:                filepath.Join(tempDir, "data"),
		LocalManifestCandidates: []string{manifestPath},
		LocalArchiveCandidates:  []string{archivePath},
		RequireAppServer:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected runtime activation")
	}
	if activated.Manifest.RuntimeID != manifest.RuntimeID {
		t.Fatalf("unexpected runtime id %q", activated.Manifest.RuntimeID)
	}
	if _, err := os.Stat(filepath.Join(activated.DesktopWebviewRoot, "index.html")); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(activated.AppServerBin); err != nil {
		t.Fatal(err)
	} else if info.Mode()&0o111 == 0 {
		t.Fatalf("expected executable app-server, got mode %v", info.Mode())
	}
}

func writeTarGz(targetPath, root string) error {
	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relative)
		if info.IsDir() {
			header.Name += "/"
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		sourceFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer sourceFile.Close()
		if _, err := io.Copy(tarWriter, sourceFile); err != nil {
			return err
		}
		return nil
	})
}
