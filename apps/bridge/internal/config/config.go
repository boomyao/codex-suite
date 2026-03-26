package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	RuntimeModeManaged = "managed"
	RuntimeModeRemote  = "remote"
)

type Config struct {
	RuntimeMode        string   `json:"runtimeMode"`
	RemoteUpstreamURL  string   `json:"remoteUpstreamUrl"`
	BridgeHost         string   `json:"bridgeHost"`
	BridgePort         int      `json:"bridgePort"`
	DesktopWebviewRoot string   `json:"desktopWebviewRoot"`
	UIPathPrefix       string   `json:"uiPathPrefix"`
	AppServerPort      int      `json:"appServerPort"`
	AppServerBin       string   `json:"appServerBin"`
	AppServerArgs      []string `json:"appServerArgs"`
	AutoRestart        bool     `json:"autoRestart"`
	RestartDelayMS     int      `json:"restartDelayMs"`
}

func DefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".codex-bridge/config.json"
	}
	return filepath.Join(homeDir, ".codex-bridge", "config.json")
}

func DefaultConfig() Config {
	return Config{
		RuntimeMode:        RuntimeModeManaged,
		RemoteUpstreamURL:  "",
		BridgeHost:         "127.0.0.1",
		BridgePort:         8787,
		DesktopWebviewRoot: "",
		UIPathPrefix:       "/ui",
		AppServerPort:      9876,
		AppServerBin:       "codex",
		AppServerArgs:      []string{},
		AutoRestart:        true,
		RestartDelayMS:     1500,
	}
}

func EnvConfig() map[string]any {
	result := map[string]any{}

	upstreamURL := envString("CODEX_BRIDGE_UPSTREAM_URL")
	manageAppServer := envBool("CODEX_MANAGE_APP_SERVER")

	if manageAppServer != nil && !*manageAppServer && upstreamURL != "" {
		result["runtimeMode"] = RuntimeModeRemote
	}

	if upstreamURL != "" {
		result["remoteUpstreamUrl"] = upstreamURL
	}
	if value := envString("CODEX_BRIDGE_HOST"); value != "" {
		result["bridgeHost"] = value
	}
	if value := envInt("CODEX_BRIDGE_PORT"); value != nil {
		result["bridgePort"] = *value
	}
	if value := envString("CODEX_BRIDGE_DESKTOP_WEBVIEW_ROOT"); value != "" {
		result["desktopWebviewRoot"] = value
	}
	if value := envString("CODEX_BRIDGE_UI_PATH_PREFIX"); value != "" {
		result["uiPathPrefix"] = value
	}
	if value := envIntAny("CODEX_APP_SERVER_PORT", "CODEX_DESKTOP_APP_SERVER_PORT"); value != nil {
		result["appServerPort"] = *value
	}
	if value := envString("CODEX_APP_SERVER_BIN"); value != "" {
		result["appServerBin"] = value
	}
	if value := envString("CODEX_APP_SERVER_EXTRA_ARGS"); value != "" {
		result["appServerArgs"] = normalizeStringArray(value)
	}
	if value := envBoolAny("CODEX_AUTO_RESTART_RUNTIME", "CODEX_DESKTOP_AUTO_RESTART_RUNTIME"); value != nil {
		result["autoRestart"] = *value
	}
	if value := envIntAny("CODEX_RUNTIME_RESTART_DELAY_MS", "CODEX_DESKTOP_RESTART_DELAY_MS"); value != nil {
		result["restartDelayMs"] = *value
	}

	return result
}

func MergeConfigs(configs ...map[string]any) (Config, error) {
	merged := map[string]any{}
	for _, current := range configs {
		for key, value := range current {
			merged[key] = value
		}
	}
	return NormalizeConfig(merged)
}

func ConfigToMap(cfg Config) map[string]any {
	return map[string]any{
		"runtimeMode":        cfg.RuntimeMode,
		"remoteUpstreamUrl":  cfg.RemoteUpstreamURL,
		"bridgeHost":         cfg.BridgeHost,
		"bridgePort":         cfg.BridgePort,
		"desktopWebviewRoot": cfg.DesktopWebviewRoot,
		"uiPathPrefix":       cfg.UIPathPrefix,
		"appServerPort":      cfg.AppServerPort,
		"appServerBin":       cfg.AppServerBin,
		"appServerArgs":      append([]string{}, cfg.AppServerArgs...),
		"autoRestart":        cfg.AutoRestart,
		"restartDelayMs":     cfg.RestartDelayMS,
	}
}

func NormalizeConfig(input map[string]any) (Config, error) {
	cfg := DefaultConfig()

	if runtimeMode, ok := input["runtimeMode"].(string); ok && strings.TrimSpace(runtimeMode) != "" {
		if strings.TrimSpace(runtimeMode) == RuntimeModeRemote {
			cfg.RuntimeMode = RuntimeModeRemote
		}
	}
	if remoteURL, ok := input["remoteUpstreamUrl"].(string); ok {
		cfg.RemoteUpstreamURL = strings.TrimSpace(remoteURL)
	}
	if bridgeHost, ok := input["bridgeHost"].(string); ok && strings.TrimSpace(bridgeHost) != "" {
		cfg.BridgeHost = strings.TrimSpace(bridgeHost)
	}
	if bridgePort, ok := normalizeInt(input["bridgePort"]); ok {
		cfg.BridgePort = bridgePort
	}
	if desktopWebviewRoot, ok := input["desktopWebviewRoot"].(string); ok {
		cfg.DesktopWebviewRoot = strings.TrimSpace(desktopWebviewRoot)
	}
	if uiPathPrefix, ok := input["uiPathPrefix"].(string); ok && strings.TrimSpace(uiPathPrefix) != "" {
		cfg.UIPathPrefix = strings.TrimSpace(uiPathPrefix)
	}
	if appServerPort, ok := normalizeInt(input["appServerPort"]); ok {
		cfg.AppServerPort = appServerPort
	}
	if appServerBin, ok := input["appServerBin"].(string); ok && strings.TrimSpace(appServerBin) != "" {
		cfg.AppServerBin = strings.TrimSpace(appServerBin)
	}
	if appServerArgs, exists := input["appServerArgs"]; exists {
		cfg.AppServerArgs = normalizeStringArray(appServerArgs)
	}
	if autoRestart, ok := input["autoRestart"].(bool); ok {
		cfg.AutoRestart = autoRestart
	}
	if restartDelayMS, ok := normalizeInt(input["restartDelayMs"]); ok {
		cfg.RestartDelayMS = restartDelayMS
	}

	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func LoadConfig(configPath string) (Config, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return Config{}, err
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Config{}, err
	}
	return NormalizeConfig(payload)
}

func SaveConfig(configPath string, cfg Config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(configPath, body, 0o644)
}

func validateConfig(cfg Config) error {
	if cfg.BridgePort <= 0 {
		return errors.New("bridgePort must be a positive integer")
	}
	if cfg.AppServerPort <= 0 {
		return errors.New("appServerPort must be a positive integer")
	}
	if cfg.UIPathPrefix == "" {
		return errors.New("uiPathPrefix cannot be empty")
	}
	if !strings.HasPrefix(cfg.UIPathPrefix, "/") {
		return errors.New("uiPathPrefix must start with /")
	}
	if cfg.RestartDelayMS < 0 {
		return errors.New("restartDelayMs must be zero or greater")
	}

	switch cfg.RuntimeMode {
	case RuntimeModeManaged:
	case RuntimeModeRemote:
		if cfg.RemoteUpstreamURL == "" {
			return errors.New("remoteUpstreamUrl is required in remote mode")
		}
		parsed, err := url.Parse(cfg.RemoteUpstreamURL)
		if err != nil {
			return fmt.Errorf("remoteUpstreamUrl must be a valid ws:// or wss:// URL: %w", err)
		}
		if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
			return errors.New("remoteUpstreamUrl must use ws:// or wss://")
		}
	default:
		return fmt.Errorf("unsupported runtimeMode %q", cfg.RuntimeMode)
	}

	return nil
}

func envString(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	return value
}

func envInt(name string) *int {
	raw := envString(name)
	if raw == "" {
		return nil
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return &parsed
}

func envIntAny(names ...string) *int {
	for _, name := range names {
		if value := envInt(name); value != nil {
			return value
		}
	}
	return nil
}

func envBool(name string) *bool {
	raw := strings.ToLower(envString(name))
	if raw == "" {
		return nil
	}
	value := !contains([]string{"0", "false", "no", "off"}, raw)
	return &value
}

func envBoolAny(names ...string) *bool {
	for _, name := range names {
		if value := envBool(name); value != nil {
			return value
		}
	}
	return nil
}

func normalizeStringArray(value any) []string {
	switch typed := value.(type) {
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if stringValue, ok := item.(string); ok {
				stringValue = strings.TrimSpace(stringValue)
				if stringValue != "" {
					out = append(out, stringValue)
				}
			}
		}
		return out
	case string:
		fields := strings.Fields(typed)
		return append([]string{}, fields...)
	default:
		return []string{}
	}
}

func normalizeInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return int(parsed), true
		}
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func contains(values []string, target string) bool {
	for _, current := range values {
		if current == target {
			return true
		}
	}
	return false
}
