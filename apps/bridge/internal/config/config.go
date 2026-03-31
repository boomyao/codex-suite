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

	ExposureModeNone    = "none"
	ExposureModeLocal   = "local"
	ExposureModeTunnel  = "tunnel"
	ExposureModeTailnet = "tailnet"

	AuthModeNone        = "none"
	AuthModeDeviceToken = "device-token"
)

type Config struct {
	Runtime  RuntimeConfig  `json:"runtime"`
	Exposure ExposureConfig `json:"exposure"`
	Auth     AuthConfig     `json:"auth"`
	Gateway  GatewayConfig  `json:"gateway"`
}

type RuntimeConfig struct {
	Mode           string   `json:"mode"`
	AppServerBin   string   `json:"appServerBin"`
	AppServerArgs  []string `json:"appServerArgs"`
	CodexHome      string   `json:"codexHome"`
	ListenHost     string   `json:"listenHost"`
	ListenPort     int      `json:"listenPort"`
	RemoteURL      string   `json:"remoteUrl"`
	AutoRestart    bool     `json:"autoRestart"`
	RestartDelayMS int      `json:"restartDelayMs"`
}

type ExposureConfig struct {
	Mode      string        `json:"mode"`
	AutoStart bool          `json:"autoStart"`
	Tunnel    TunnelConfig  `json:"tunnel"`
	Tailnet   TailnetConfig `json:"tailnet"`
}

type TunnelConfig struct {
	SSHBinary      string   `json:"sshBinary"`
	SSHDestination string   `json:"sshDestination"`
	SSHPort        int      `json:"sshPort"`
	RemotePort     int      `json:"remotePort"`
	PublicHost     string   `json:"publicHost"`
	PublicPort     int      `json:"publicPort"`
	PublicScheme   string   `json:"publicScheme"`
	SSHArgs        []string `json:"sshArgs"`
	AutoRestart    bool     `json:"autoRestart"`
	RestartDelayMS int      `json:"restartDelayMs"`
}

type TailnetConfig struct {
	Socket                  string   `json:"socket"`
	Hostname                string   `json:"hostname"`
	AddressStrategy         string   `json:"addressStrategy"`
	MobileControlURL        string   `json:"mobileControlUrl"`
	MobileHostname          string   `json:"mobileHostname"`
	MobileOAuthClientID     string   `json:"mobileOAuthClientId"`
	MobileOAuthClientSecret string   `json:"mobileOAuthClientSecret"`
	MobileOAuthTailnet      string   `json:"mobileOAuthTailnet"`
	MobileOAuthTags         []string `json:"mobileOAuthTags"`
}

type AuthConfig struct {
	Mode             string `json:"mode"`
	RequireApproval  bool   `json:"requireApproval"`
	DeviceStorePath  string `json:"deviceStorePath"`
	PairingCodeTTLMS int    `json:"pairingCodeTtlMs"`
}

type GatewayConfig struct {
	Host               string `json:"host"`
	Port               int    `json:"port"`
	DesktopWebviewRoot string `json:"desktopWebviewRoot"`
	UIPathPrefix       string `json:"uiPathPrefix"`
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
		Runtime: RuntimeConfig{
			Mode:           RuntimeModeManaged,
			AppServerBin:   "codex",
			AppServerArgs:  []string{},
			CodexHome:      "",
			ListenHost:     "127.0.0.1",
			ListenPort:     9876,
			RemoteURL:      "",
			AutoRestart:    true,
			RestartDelayMS: 1500,
		},
		Exposure: ExposureConfig{
			Mode:      ExposureModeTailnet,
			AutoStart: true,
			Tunnel: TunnelConfig{
				SSHBinary:      "ssh",
				SSHDestination: "",
				SSHPort:        22,
				RemotePort:     0,
				PublicHost:     "",
				PublicPort:     0,
				PublicScheme:   "http",
				SSHArgs:        []string{},
				AutoRestart:    true,
				RestartDelayMS: 2000,
			},
			Tailnet: TailnetConfig{
				Socket:                  "",
				Hostname:                "",
				AddressStrategy:         "auto",
				MobileControlURL:        "",
				MobileHostname:          "",
				MobileOAuthClientID:     "",
				MobileOAuthClientSecret: "",
				MobileOAuthTailnet:      "",
				MobileOAuthTags:         []string{},
			},
		},
		Auth: AuthConfig{
			Mode:             AuthModeDeviceToken,
			RequireApproval:  false,
			DeviceStorePath:  "",
			PairingCodeTTLMS: 300000,
		},
		Gateway: GatewayConfig{
			Host:               "0.0.0.0",
			Port:               8787,
			DesktopWebviewRoot: "",
			UIPathPrefix:       "/ui",
		},
	}
}

func EnvConfig() map[string]any {
	result := map[string]any{}
	runtimeCfg := map[string]any{}
	exposureCfg := map[string]any{}
	authCfg := map[string]any{}
	gatewayCfg := map[string]any{}

	if value := envString("CODEX_RUNTIME_MODE"); value != "" {
		runtimeCfg["mode"] = value
	}
	if value := envString("CODEX_RUNTIME_REMOTE_URL"); value != "" {
		runtimeCfg["remoteUrl"] = value
	}
	if value := envString("CODEX_RUNTIME_LISTEN_HOST"); value != "" {
		runtimeCfg["listenHost"] = value
	}
	if value := envIntAny("CODEX_RUNTIME_LISTEN_PORT", "CODEX_APP_SERVER_PORT", "CODEX_DESKTOP_APP_SERVER_PORT"); value != nil {
		runtimeCfg["listenPort"] = *value
	}
	if value := envString("CODEX_APP_SERVER_BIN"); value != "" {
		runtimeCfg["appServerBin"] = value
	}
	if value := envString("CODEX_APP_SERVER_EXTRA_ARGS"); value != "" {
		runtimeCfg["appServerArgs"] = normalizeStringArray(value)
	}
	if value := envString("CODEX_RUNTIME_CODEX_HOME"); value != "" {
		runtimeCfg["codexHome"] = value
	}
	if value := envBoolAny("CODEX_AUTO_RESTART_RUNTIME", "CODEX_DESKTOP_AUTO_RESTART_RUNTIME"); value != nil {
		runtimeCfg["autoRestart"] = *value
	}
	if value := envIntAny("CODEX_RUNTIME_RESTART_DELAY_MS", "CODEX_DESKTOP_RESTART_DELAY_MS"); value != nil {
		runtimeCfg["restartDelayMs"] = *value
	}

	if value := envString("CODEX_EXPOSURE_MODE"); value != "" {
		exposureCfg["mode"] = value
	}
	if value := envBool("CODEX_EXPOSURE_AUTO_START"); value != nil {
		exposureCfg["autoStart"] = *value
	}
	tunnelCfg := map[string]any{}
	if value := envString("CODEX_EXPOSURE_TUNNEL_SSH_BINARY"); value != "" {
		tunnelCfg["sshBinary"] = value
	}
	if value := envString("CODEX_EXPOSURE_TUNNEL_SSH_DESTINATION"); value != "" {
		tunnelCfg["sshDestination"] = value
	}
	if value := envInt("CODEX_EXPOSURE_TUNNEL_SSH_PORT"); value != nil {
		tunnelCfg["sshPort"] = *value
	}
	if value := envInt("CODEX_EXPOSURE_TUNNEL_REMOTE_PORT"); value != nil {
		tunnelCfg["remotePort"] = *value
	}
	if value := envString("CODEX_EXPOSURE_TUNNEL_PUBLIC_HOST"); value != "" {
		tunnelCfg["publicHost"] = value
	}
	if value := envInt("CODEX_EXPOSURE_TUNNEL_PUBLIC_PORT"); value != nil {
		tunnelCfg["publicPort"] = *value
	}
	if value := envString("CODEX_EXPOSURE_TUNNEL_PUBLIC_SCHEME"); value != "" {
		tunnelCfg["publicScheme"] = value
	}
	if value := envString("CODEX_EXPOSURE_TUNNEL_SSH_ARGS"); value != "" {
		tunnelCfg["sshArgs"] = normalizeStringArray(value)
	}
	if value := envBool("CODEX_EXPOSURE_TUNNEL_AUTO_RESTART"); value != nil {
		tunnelCfg["autoRestart"] = *value
	}
	if value := envInt("CODEX_EXPOSURE_TUNNEL_RESTART_DELAY_MS"); value != nil {
		tunnelCfg["restartDelayMs"] = *value
	}
	if len(tunnelCfg) > 0 {
		exposureCfg["tunnel"] = tunnelCfg
	}
	tailnetCfg := map[string]any{}
	if value := envString("CODEX_EXPOSURE_TAILNET_SOCKET"); value != "" {
		tailnetCfg["socket"] = value
	}
	if value := envString("CODEX_EXPOSURE_TAILNET_HOSTNAME"); value != "" {
		tailnetCfg["hostname"] = value
	}
	if value := envString("CODEX_EXPOSURE_TAILNET_ADDRESS_STRATEGY"); value != "" {
		tailnetCfg["addressStrategy"] = value
	}
	if value := envString("CODEX_EXPOSURE_TAILNET_MOBILE_CONTROL_URL"); value != "" {
		tailnetCfg["mobileControlUrl"] = value
	}
	if value := envString("CODEX_EXPOSURE_TAILNET_MOBILE_HOSTNAME"); value != "" {
		tailnetCfg["mobileHostname"] = value
	}
	if value := envString("CODEX_EXPOSURE_TAILNET_MOBILE_OAUTH_CLIENT_ID"); value != "" {
		tailnetCfg["mobileOAuthClientId"] = value
	}
	if value := envString("CODEX_EXPOSURE_TAILNET_MOBILE_OAUTH_CLIENT_SECRET"); value != "" {
		tailnetCfg["mobileOAuthClientSecret"] = value
	}
	if value := envString("CODEX_EXPOSURE_TAILNET_MOBILE_OAUTH_TAILNET"); value != "" {
		tailnetCfg["mobileOAuthTailnet"] = value
	}
	if value := envString("CODEX_EXPOSURE_TAILNET_MOBILE_OAUTH_TAGS"); value != "" {
		tailnetCfg["mobileOAuthTags"] = normalizeCSVStringArray(value)
	}
	if len(tailnetCfg) > 0 {
		exposureCfg["tailnet"] = tailnetCfg
	}

	if value := envString("CODEX_AUTH_MODE"); value != "" {
		authCfg["mode"] = value
	}
	if value := envBool("CODEX_AUTH_REQUIRE_APPROVAL"); value != nil {
		authCfg["requireApproval"] = *value
	}
	if value := envString("CODEX_AUTH_DEVICE_STORE_PATH"); value != "" {
		authCfg["deviceStorePath"] = value
	}
	if value := envInt("CODEX_AUTH_PAIRING_CODE_TTL_MS"); value != nil {
		authCfg["pairingCodeTtlMs"] = *value
	}

	if value := envString("CODEX_BRIDGE_HOST"); value != "" {
		gatewayCfg["host"] = value
	}
	if value := envInt("CODEX_BRIDGE_PORT"); value != nil {
		gatewayCfg["port"] = *value
	}
	if value := envString("CODEX_BRIDGE_DESKTOP_WEBVIEW_ROOT"); value != "" {
		gatewayCfg["desktopWebviewRoot"] = value
	}
	if value := envString("CODEX_BRIDGE_UI_PATH_PREFIX"); value != "" {
		gatewayCfg["uiPathPrefix"] = value
	}

	if len(runtimeCfg) > 0 {
		result["runtime"] = runtimeCfg
	}
	if len(exposureCfg) > 0 {
		result["exposure"] = exposureCfg
	}
	if len(authCfg) > 0 {
		result["auth"] = authCfg
	}
	if len(gatewayCfg) > 0 {
		result["gateway"] = gatewayCfg
	}
	return result
}

func MergeConfigs(configs ...map[string]any) (Config, error) {
	merged := map[string]any{}
	for _, current := range configs {
		merged = mergeMaps(merged, current)
	}
	return NormalizeConfig(merged)
}

func ConfigToMap(cfg Config) map[string]any {
	return map[string]any{
		"runtime": map[string]any{
			"mode":           cfg.Runtime.Mode,
			"appServerBin":   cfg.Runtime.AppServerBin,
			"appServerArgs":  append([]string{}, cfg.Runtime.AppServerArgs...),
			"codexHome":      cfg.Runtime.CodexHome,
			"listenHost":     cfg.Runtime.ListenHost,
			"listenPort":     cfg.Runtime.ListenPort,
			"remoteUrl":      cfg.Runtime.RemoteURL,
			"autoRestart":    cfg.Runtime.AutoRestart,
			"restartDelayMs": cfg.Runtime.RestartDelayMS,
		},
		"exposure": map[string]any{
			"mode":      cfg.Exposure.Mode,
			"autoStart": cfg.Exposure.AutoStart,
			"tunnel": map[string]any{
				"sshBinary":      cfg.Exposure.Tunnel.SSHBinary,
				"sshDestination": cfg.Exposure.Tunnel.SSHDestination,
				"sshPort":        cfg.Exposure.Tunnel.SSHPort,
				"remotePort":     cfg.Exposure.Tunnel.RemotePort,
				"publicHost":     cfg.Exposure.Tunnel.PublicHost,
				"publicPort":     cfg.Exposure.Tunnel.PublicPort,
				"publicScheme":   cfg.Exposure.Tunnel.PublicScheme,
				"sshArgs":        append([]string{}, cfg.Exposure.Tunnel.SSHArgs...),
				"autoRestart":    cfg.Exposure.Tunnel.AutoRestart,
				"restartDelayMs": cfg.Exposure.Tunnel.RestartDelayMS,
			},
			"tailnet": map[string]any{
				"socket":                  cfg.Exposure.Tailnet.Socket,
				"hostname":                cfg.Exposure.Tailnet.Hostname,
				"addressStrategy":         cfg.Exposure.Tailnet.AddressStrategy,
				"mobileControlUrl":        cfg.Exposure.Tailnet.MobileControlURL,
				"mobileHostname":          cfg.Exposure.Tailnet.MobileHostname,
				"mobileOAuthClientId":     cfg.Exposure.Tailnet.MobileOAuthClientID,
				"mobileOAuthClientSecret": cfg.Exposure.Tailnet.MobileOAuthClientSecret,
				"mobileOAuthTailnet":      cfg.Exposure.Tailnet.MobileOAuthTailnet,
				"mobileOAuthTags":         append([]string{}, cfg.Exposure.Tailnet.MobileOAuthTags...),
			},
		},
		"auth": map[string]any{
			"mode":             cfg.Auth.Mode,
			"requireApproval":  cfg.Auth.RequireApproval,
			"deviceStorePath":  cfg.Auth.DeviceStorePath,
			"pairingCodeTtlMs": cfg.Auth.PairingCodeTTLMS,
		},
		"gateway": map[string]any{
			"host":               cfg.Gateway.Host,
			"port":               cfg.Gateway.Port,
			"desktopWebviewRoot": cfg.Gateway.DesktopWebviewRoot,
			"uiPathPrefix":       cfg.Gateway.UIPathPrefix,
		},
	}
}

func NormalizeConfig(input map[string]any) (Config, error) {
	cfg := DefaultConfig()

	runtimeCfg := mapValue(input, "runtime")
	if mode, ok := stringValue(runtimeCfg, "mode"); ok && mode != "" {
		cfg.Runtime.Mode = mode
	}
	if value, ok := stringValue(runtimeCfg, "appServerBin"); ok && value != "" {
		cfg.Runtime.AppServerBin = value
	}
	if value, ok := runtimeCfg["appServerArgs"]; ok {
		cfg.Runtime.AppServerArgs = normalizeStringArray(value)
	}
	if value, ok := stringValue(runtimeCfg, "codexHome"); ok {
		cfg.Runtime.CodexHome = value
	}
	if value, ok := stringValue(runtimeCfg, "listenHost"); ok && value != "" {
		cfg.Runtime.ListenHost = value
	}
	if value, ok := normalizeInt(runtimeCfg["listenPort"]); ok {
		cfg.Runtime.ListenPort = value
	}
	if value, ok := stringValue(runtimeCfg, "remoteUrl"); ok {
		cfg.Runtime.RemoteURL = value
	}
	if value, ok := runtimeCfg["autoRestart"].(bool); ok {
		cfg.Runtime.AutoRestart = value
	}
	if value, ok := normalizeInt(runtimeCfg["restartDelayMs"]); ok {
		cfg.Runtime.RestartDelayMS = value
	}

	exposureCfg := mapValue(input, "exposure")
	if value, ok := stringValue(exposureCfg, "mode"); ok && value != "" {
		cfg.Exposure.Mode = value
	}
	if value, ok := exposureCfg["autoStart"].(bool); ok {
		cfg.Exposure.AutoStart = value
	}
	tunnelCfg := mapValue(exposureCfg, "tunnel")
	if value, ok := stringValue(tunnelCfg, "sshBinary"); ok && value != "" {
		cfg.Exposure.Tunnel.SSHBinary = value
	}
	if value, ok := stringValue(tunnelCfg, "sshDestination"); ok {
		cfg.Exposure.Tunnel.SSHDestination = value
	}
	if value, ok := normalizeInt(tunnelCfg["sshPort"]); ok {
		cfg.Exposure.Tunnel.SSHPort = value
	}
	if value, ok := normalizeInt(tunnelCfg["remotePort"]); ok {
		cfg.Exposure.Tunnel.RemotePort = value
	}
	if value, ok := stringValue(tunnelCfg, "publicHost"); ok {
		cfg.Exposure.Tunnel.PublicHost = value
	}
	if value, ok := normalizeInt(tunnelCfg["publicPort"]); ok {
		cfg.Exposure.Tunnel.PublicPort = value
	}
	if value, ok := stringValue(tunnelCfg, "publicScheme"); ok && value != "" {
		cfg.Exposure.Tunnel.PublicScheme = value
	}
	if value, exists := tunnelCfg["sshArgs"]; exists {
		cfg.Exposure.Tunnel.SSHArgs = normalizeStringArray(value)
	}
	if value, ok := tunnelCfg["autoRestart"].(bool); ok {
		cfg.Exposure.Tunnel.AutoRestart = value
	}
	if value, ok := normalizeInt(tunnelCfg["restartDelayMs"]); ok {
		cfg.Exposure.Tunnel.RestartDelayMS = value
	}
	tailnetCfg := mapValue(exposureCfg, "tailnet")
	if value, ok := stringValue(tailnetCfg, "socket"); ok {
		cfg.Exposure.Tailnet.Socket = value
	}
	if value, ok := stringValue(tailnetCfg, "hostname"); ok {
		cfg.Exposure.Tailnet.Hostname = value
	}
	if value, ok := stringValue(tailnetCfg, "addressStrategy"); ok && value != "" {
		cfg.Exposure.Tailnet.AddressStrategy = value
	}
	if value, ok := stringValue(tailnetCfg, "mobileControlUrl"); ok {
		cfg.Exposure.Tailnet.MobileControlURL = value
	}
	if value, ok := stringValue(tailnetCfg, "mobileHostname"); ok {
		cfg.Exposure.Tailnet.MobileHostname = value
	}
	if value, ok := stringValue(tailnetCfg, "mobileOAuthClientId"); ok {
		cfg.Exposure.Tailnet.MobileOAuthClientID = value
	}
	if value, ok := stringValue(tailnetCfg, "mobileOAuthClientSecret"); ok {
		cfg.Exposure.Tailnet.MobileOAuthClientSecret = value
	}
	if value, ok := stringValue(tailnetCfg, "mobileOAuthTailnet"); ok {
		cfg.Exposure.Tailnet.MobileOAuthTailnet = value
	}
	if value, exists := tailnetCfg["mobileOAuthTags"]; exists {
		cfg.Exposure.Tailnet.MobileOAuthTags = normalizeCSVStringArray(value)
	}

	authCfg := mapValue(input, "auth")
	if value, ok := stringValue(authCfg, "mode"); ok && value != "" {
		cfg.Auth.Mode = value
	}
	if value, ok := authCfg["requireApproval"].(bool); ok {
		cfg.Auth.RequireApproval = value
	}
	if value, ok := stringValue(authCfg, "deviceStorePath"); ok {
		cfg.Auth.DeviceStorePath = value
	}
	if value, ok := normalizeInt(authCfg["pairingCodeTtlMs"]); ok {
		cfg.Auth.PairingCodeTTLMS = value
	}

	gatewayCfg := mapValue(input, "gateway")
	if value, ok := stringValue(gatewayCfg, "host"); ok && value != "" {
		cfg.Gateway.Host = value
	}
	if value, ok := normalizeInt(gatewayCfg["port"]); ok {
		cfg.Gateway.Port = value
	}
	if value, ok := stringValue(gatewayCfg, "desktopWebviewRoot"); ok {
		cfg.Gateway.DesktopWebviewRoot = value
	}
	if value, ok := stringValue(gatewayCfg, "uiPathPrefix"); ok && value != "" {
		cfg.Gateway.UIPathPrefix = value
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
	if err := os.WriteFile(configPath, body, 0o600); err != nil {
		return err
	}
	return os.Chmod(configPath, 0o600)
}

func validateConfig(cfg Config) error {
	if cfg.Runtime.ListenPort <= 0 {
		return errors.New("runtime.listenPort must be a positive integer")
	}
	if cfg.Gateway.Port <= 0 {
		return errors.New("gateway.port must be a positive integer")
	}
	if cfg.Gateway.UIPathPrefix == "" {
		return errors.New("gateway.uiPathPrefix cannot be empty")
	}
	if !strings.HasPrefix(cfg.Gateway.UIPathPrefix, "/") {
		return errors.New("gateway.uiPathPrefix must start with /")
	}
	if cfg.Runtime.RestartDelayMS < 0 {
		return errors.New("runtime.restartDelayMs must be zero or greater")
	}
	if cfg.Auth.PairingCodeTTLMS <= 0 {
		return errors.New("auth.pairingCodeTtlMs must be a positive integer")
	}

	switch cfg.Runtime.Mode {
	case RuntimeModeManaged:
	case RuntimeModeRemote:
		if cfg.Runtime.RemoteURL == "" {
			return errors.New("runtime.remoteUrl is required in remote mode")
		}
		parsed, err := url.Parse(cfg.Runtime.RemoteURL)
		if err != nil {
			return fmt.Errorf("runtime.remoteUrl must be a valid ws:// or wss:// URL: %w", err)
		}
		if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
			return errors.New("runtime.remoteUrl must use ws:// or wss://")
		}
	default:
		return fmt.Errorf("unsupported runtime.mode %q", cfg.Runtime.Mode)
	}

	switch cfg.Exposure.Mode {
	case ExposureModeNone, ExposureModeLocal, ExposureModeTunnel, ExposureModeTailnet:
	default:
		return fmt.Errorf("unsupported exposure.mode %q", cfg.Exposure.Mode)
	}
	if cfg.Exposure.Tunnel.SSHPort <= 0 {
		return errors.New("exposure.tunnel.sshPort must be a positive integer")
	}
	if cfg.Exposure.Tunnel.RestartDelayMS < 0 {
		return errors.New("exposure.tunnel.restartDelayMs must be zero or greater")
	}
	switch cfg.Exposure.Tunnel.PublicScheme {
	case "http", "https":
	default:
		return fmt.Errorf("unsupported exposure.tunnel.publicScheme %q", cfg.Exposure.Tunnel.PublicScheme)
	}
	if cfg.Exposure.Mode == ExposureModeTunnel {
		if strings.TrimSpace(cfg.Exposure.Tunnel.SSHDestination) == "" {
			return errors.New("exposure.tunnel.sshDestination is required in tunnel mode")
		}
		if cfg.Exposure.Tunnel.RemotePort <= 0 {
			return errors.New("exposure.tunnel.remotePort must be a positive integer in tunnel mode")
		}
	}
	switch cfg.Exposure.Tailnet.AddressStrategy {
	case "auto", "dns", "ipv4", "ipv6":
	default:
		return fmt.Errorf("unsupported exposure.tailnet.addressStrategy %q", cfg.Exposure.Tailnet.AddressStrategy)
	}
	if value := strings.TrimSpace(cfg.Exposure.Tailnet.MobileControlURL); value != "" {
		parsed, err := url.Parse(value)
		if err != nil {
			return fmt.Errorf("exposure.tailnet.mobileControlUrl must be a valid http:// or https:// URL: %w", err)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return errors.New("exposure.tailnet.mobileControlUrl must use http:// or https://")
		}
		if strings.TrimSpace(parsed.Host) == "" {
			return errors.New("exposure.tailnet.mobileControlUrl must include a host")
		}
	}
	hasOAuthField := strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthClientID) != "" ||
		strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthClientSecret) != "" ||
		strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthTailnet) != "" ||
		len(cfg.Exposure.Tailnet.MobileOAuthTags) > 0
	if hasOAuthField {
		if strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthClientID) == "" {
			return errors.New("exposure.tailnet.mobileOAuthClientId is required when tailnet mobile OAuth provisioning is configured")
		}
		if strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthClientSecret) == "" {
			return errors.New("exposure.tailnet.mobileOAuthClientSecret is required when tailnet mobile OAuth provisioning is configured")
		}
		if strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthTailnet) == "" {
			return errors.New("exposure.tailnet.mobileOAuthTailnet is required when tailnet mobile OAuth provisioning is configured")
		}
		if len(cfg.Exposure.Tailnet.MobileOAuthTags) == 0 {
			return errors.New("exposure.tailnet.mobileOAuthTags must include at least one tag when tailnet mobile OAuth provisioning is configured")
		}
	}

	switch cfg.Auth.Mode {
	case AuthModeNone:
	case AuthModeDeviceToken:
	default:
		return fmt.Errorf("unsupported auth.mode %q", cfg.Auth.Mode)
	}

	return nil
}

func mergeMaps(base map[string]any, next map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(next))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range next {
		if nestedNext, ok := value.(map[string]any); ok {
			if nestedBase, ok := result[key].(map[string]any); ok {
				result[key] = mergeMaps(nestedBase, nestedNext)
				continue
			}
		}
		result[key] = value
	}
	return result
}

func mapValue(input map[string]any, key string) map[string]any {
	value, ok := input[key]
	if !ok {
		return map[string]any{}
	}
	result, ok := value.(map[string]any)
	if ok {
		return result
	}
	return map[string]any{}
}

func stringValue(input map[string]any, key string) (string, bool) {
	value, ok := input[key]
	if !ok {
		return "", false
	}
	typed, ok := value.(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(typed), true
}

func envString(name string) string {
	return strings.TrimSpace(os.Getenv(name))
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

func normalizeCSVStringArray(value any) []string {
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
		parts := strings.FieldsFunc(typed, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
		})
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		return out
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
