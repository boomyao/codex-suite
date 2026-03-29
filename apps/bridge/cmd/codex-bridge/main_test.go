package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/boomyao/codex-bridge/internal/config"
	"github.com/boomyao/codex-bridge/internal/runtimestore"
)

func TestResolveDesktopWebviewRootPrefersConfiguredRoot(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configuredRoot := filepath.Join(tempDir, "configured")
	releaseRoot := filepath.Join(tempDir, "release-runtime")

	writeDesktopWebviewRoot(t, configuredRoot)
	writeDesktopWebviewRoot(t, releaseRoot)

	root, err := resolveDesktopWebviewRoot(configuredRoot, releaseRoot)
	if err != nil {
		t.Fatal(err)
	}
	if root != configuredRoot {
		t.Fatalf("expected configured root %q, got %q", configuredRoot, root)
	}
}

func TestResolveDesktopWebviewRootUsesReleaseRuntimeWhenConfiguredRootMissing(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	releaseRoot := filepath.Join(tempDir, "release-runtime")
	writeDesktopWebviewRoot(t, releaseRoot)

	root, err := resolveDesktopWebviewRoot("", releaseRoot)
	if err != nil {
		t.Fatal(err)
	}
	if root != releaseRoot {
		t.Fatalf("expected release runtime root %q, got %q", releaseRoot, root)
	}
}

func TestBuildBridgeConfigUsesReleaseRuntimeDesktopWebview(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bridge-config.json")
	releaseRoot := filepath.Join(tempDir, "release-runtime")
	writeDesktopWebviewRoot(t, releaseRoot)

	cfg := config.DefaultConfig()
	bridgeConfig, err := buildBridgeConfig(
		cfg,
		configPath,
		"ws://127.0.0.1:9876",
		"",
		runtimestore.ActivatedRuntime{DesktopWebviewRoot: releaseRoot},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if bridgeConfig.DesktopWebviewRoot != releaseRoot {
		t.Fatalf("expected bridge desktop webview root %q, got %q", releaseRoot, bridgeConfig.DesktopWebviewRoot)
	}
}

func TestBuildBridgeConfigFailsWithoutAnyDesktopWebviewBundle(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bridge-config.json")
	cfg := config.DefaultConfig()

	_, err := buildBridgeConfig(
		cfg,
		configPath,
		"ws://127.0.0.1:9876",
		"",
		runtimestore.ActivatedRuntime{},
		false,
	)
	if err == nil {
		t.Fatal("expected missing desktop webview bundle error")
	}
}

func writeDesktopWebviewRoot(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<!doctype html>"), 0o644); err != nil {
		t.Fatal(err)
	}
}
