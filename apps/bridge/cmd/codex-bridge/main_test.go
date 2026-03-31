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

func TestBuildStartupMobileQRKeepsQuietZoneForScanning(t *testing.T) {
	payload := []byte(`{"type":"codex-mobile-enrollment","bridgeName":"example","bridgeServerEndpoint":"ws://example.ts.net:8787","pairingCode":"12345678","tailnet":{"clientSecret":"oauth-client-secret","oauthClientId":"oauth-client-id","oauthTailnet":"example.ts.net","oauthTags":["tag:codex-mobile"],"loginMode":"oauth-client-secret"}}`)

	actual, err := buildStartupMobileQR(payload)
	if err != nil {
		t.Fatal(err)
	}

	expectedQR, err := qrcode.New(string(payload), qrcode.Low)
	if err != nil {
		t.Fatal(err)
	}
	expected := expectedQR.ToSmallString(false)

	borderlessQR, err := qrcode.New(string(payload), qrcode.Low)
	if err != nil {
		t.Fatal(err)
	}
	borderlessQR.DisableBorder = true
	borderless := borderlessQR.ToSmallString(false)

	actualWidth, actualHeight := qrStringDimensions(actual)
	expectedWidth, expectedHeight := qrStringDimensions(expected)
	borderlessWidth, borderlessHeight := qrStringDimensions(borderless)
	if actualWidth != expectedWidth || actualHeight != expectedHeight {
		t.Fatalf(
			"expected QR dimensions %dx%d with quiet zone, got %dx%d",
			expectedWidth,
			expectedHeight,
			actualWidth,
			actualHeight,
		)
	}
	if actualWidth <= borderlessWidth || actualHeight <= borderlessHeight {
		t.Fatalf(
			"expected generated QR to keep a larger quiet zone than borderless output, got actual %dx%d vs borderless %dx%d",
			actualWidth,
			actualHeight,
			borderlessWidth,
			borderlessHeight,
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
