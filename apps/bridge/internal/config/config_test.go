package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadConfigRoundTripsMobileAPIAccessToken(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	cfg := DefaultConfig()
	cfg.Exposure.Tailnet.MobileAPIAccessToken = "tskey-api-test"

	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Exposure.Tailnet.MobileAPIAccessToken != cfg.Exposure.Tailnet.MobileAPIAccessToken {
		t.Fatalf("unexpected mobile API access token %q", loaded.Exposure.Tailnet.MobileAPIAccessToken)
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
