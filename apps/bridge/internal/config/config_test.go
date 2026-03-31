package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadConfigRoundTripsMobileOAuthEnrollmentConfig(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	cfg := DefaultConfig()
	cfg.Exposure.Tailnet.MobileOAuthClientID = "oauth-client-id"
	cfg.Exposure.Tailnet.MobileOAuthClientSecret = "oauth-client-secret"
	cfg.Exposure.Tailnet.MobileOAuthTailnet = "example.ts.net"
	cfg.Exposure.Tailnet.MobileOAuthTags = []string{"tag:codex-mobile"}

	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Exposure.Tailnet.MobileOAuthClientID != cfg.Exposure.Tailnet.MobileOAuthClientID {
		t.Fatalf("unexpected mobile OAuth client ID %q", loaded.Exposure.Tailnet.MobileOAuthClientID)
	}
	if loaded.Exposure.Tailnet.MobileOAuthClientSecret != cfg.Exposure.Tailnet.MobileOAuthClientSecret {
		t.Fatalf("unexpected mobile OAuth client secret %q", loaded.Exposure.Tailnet.MobileOAuthClientSecret)
	}
	if loaded.Exposure.Tailnet.MobileOAuthTailnet != cfg.Exposure.Tailnet.MobileOAuthTailnet {
		t.Fatalf("unexpected mobile OAuth tailnet %q", loaded.Exposure.Tailnet.MobileOAuthTailnet)
	}
	if len(loaded.Exposure.Tailnet.MobileOAuthTags) != len(cfg.Exposure.Tailnet.MobileOAuthTags) ||
		loaded.Exposure.Tailnet.MobileOAuthTags[0] != cfg.Exposure.Tailnet.MobileOAuthTags[0] {
		t.Fatalf("unexpected mobile OAuth tags %v", loaded.Exposure.Tailnet.MobileOAuthTags)
	}
}

func TestSaveConfigRestrictsPermissions(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not portable on Windows")
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	if err := os.WriteFile(configPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := SaveConfig(configPath, DefaultConfig()); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("unexpected config permissions %o", got)
	}
}
