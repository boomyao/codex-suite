package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boomyao/codex-bridge/internal/config"
	"github.com/boomyao/codex-bridge/internal/runtimestore"
	qrcode "github.com/skip2/go-qrcode"
)

func TestResolveDesktopWebviewRootPrefersConfiguredRoot(t *testing.T) {
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
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bridge-config.json")
	cfg := config.DefaultConfig()
	previousAsarPath := bundledDesktopWebviewAppAsarPath
	bundledDesktopWebviewAppAsarPath = filepath.Join(tempDir, "missing-app.asar")
	t.Cleanup(func() {
		bundledDesktopWebviewAppAsarPath = previousAsarPath
	})

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

func TestExtractBundledDesktopWebviewRootUsesNpxAsarExtract(t *testing.T) {
	tempDir := t.TempDir()
	fixtureRoot := filepath.Join(tempDir, "fixture")
	writeDesktopWebviewRoot(t, filepath.Join(fixtureRoot, "webview"))

	fakeNpxPath := filepath.Join(tempDir, "fake-npx.sh")
	if err := os.Setenv("CODEX_BRIDGE_TEST_FIXTURE_ROOT", fixtureRoot); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("CODEX_BRIDGE_TEST_FIXTURE_ROOT")
	})
	script := "#!/bin/sh\nset -eu\nmkdir -p \"$5\"\ncp -R \"$CODEX_BRIDGE_TEST_FIXTURE_ROOT/.\" \"$5\"\n"
	if err := os.WriteFile(fakeNpxPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}

	previousLookPath := execLookPath
	previousCommandContext := execCommandContext
	execLookPath = func(file string) (string, error) {
		if file != "npx" {
			return "", exec.ErrNotFound
		}
		return fakeNpxPath, nil
	}
	execCommandContext = exec.CommandContext
	t.Cleanup(func() {
		execLookPath = previousLookPath
		execCommandContext = previousCommandContext
	})

	asarPath := filepath.Join(tempDir, "app.asar")
	if err := os.WriteFile(asarPath, []byte("asar"), 0o644); err != nil {
		t.Fatal(err)
	}

	root, err := extractBundledDesktopWebviewRoot(asarPath, filepath.Join(tempDir, "cache"))
	if err != nil {
		t.Fatal(err)
	}
	if root != filepath.Join(tempDir, "cache", "webview") {
		t.Fatalf("expected extracted webview root, got %q", root)
	}
	if _, err := os.Stat(filepath.Join(root, "index.html")); err != nil {
		t.Fatalf("expected extracted index.html: %v", err)
	}
}

func TestBuildStartupMobileQRMakesCompactTerminalQR(t *testing.T) {
	payload := []byte(`{"type":"codex-mobile-enrollment","version":1,"bridgeName":"example","bridgeServerEndpoint":"ws://example.ts.net:8787","pairingCode":"12345678","tailnet":{"authKey":"tskey-auth-example-example-example"}}`)

	compact, err := buildStartupMobileQR(payload)
	if err != nil {
		t.Fatal(err)
	}

	defaultQR, err := qrcode.New(string(payload), qrcode.Medium)
	if err != nil {
		t.Fatal(err)
	}

	compactWidth, compactHeight := qrStringDimensions(compact)
	defaultWidth, defaultHeight := qrStringDimensions(defaultQR.ToSmallString(false))
	if compactWidth >= defaultWidth && compactHeight >= defaultHeight {
		t.Fatalf(
			"expected compact QR to shrink terminal output, got compact %dx%d vs default %dx%d",
			compactWidth,
			compactHeight,
			defaultWidth,
			defaultHeight,
		)
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

func qrStringDimensions(value string) (int, int) {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	height := len(lines)
	width := 0
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}
	return width, height
}
