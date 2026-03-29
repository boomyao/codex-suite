package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/boomyao/codex-bridge/internal/config"
	"golang.org/x/term"
	"tailscale.com/client/local"
)

const tailscaleOAuthSettingsURL = "https://login.tailscale.com/admin/settings/oauth"

func maybeBootstrapTailnetMobileEnrollment(
	cfg config.Config,
	configPath string,
	promptOut *os.File,
	logger *log.Logger,
) (config.Config, error) {
	if !needsTailnetMobileEnrollmentBootstrap(cfg) {
		return cfg, nil
	}
	if promptOut == nil {
		return cfg, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(promptOut.Fd())) {
		return cfg, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintln(promptOut, "codex-bridge tailnet mobile enrollment is not configured.")
	fmt.Fprintln(promptOut, "A one-time Tailscale OAuth setup can be completed now and stored in codex-bridge config.")
	fmt.Fprintln(promptOut, "This enables startup QR codes that let Codex Mobile auto-join the tailnet.")
	fmt.Fprintf(promptOut, "Open Tailscale OAuth settings: %s\n", tailscaleOAuthSettingsURL)
	fmt.Fprintln(promptOut, "Create an OAuth client with scope 'Auth Keys: Write' and choose a tag that Codex Mobile nodes will use.")
	fmt.Fprintln(promptOut)

	openedBrowser := tryOpenBrowser(tailscaleOAuthSettingsURL, logger)
	if openedBrowser {
		fmt.Fprintln(promptOut, "Opened Tailscale OAuth settings in your browser.")
	}

	proceed, err := promptYesNo(reader, promptOut, "Configure tailnet mobile enrollment now? [Y/n]", true)
	if err != nil {
		return cfg, err
	}
	if !proceed {
		return cfg, nil
	}

	detectedTailnet := detectCurrentTailnetName(context.Background())
	tailnetName, err := promptString(reader, promptOut, "Tailnet name", detectedTailnet)
	if err != nil {
		return cfg, err
	}
	clientID, err := promptRequiredString(reader, promptOut, "OAuth client ID")
	if err != nil {
		return cfg, err
	}
	clientSecret, err := promptRequiredSecret(promptOut, "OAuth client secret")
	if err != nil {
		return cfg, err
	}
	tags, err := promptString(reader, promptOut, "Mobile tags (comma-separated)", "tag:codex-mobile")
	if err != nil {
		return cfg, err
	}

	cfg.Exposure.Tailnet.MobileOAuthTailnet = tailnetName
	cfg.Exposure.Tailnet.MobileOAuthClientID = clientID
	cfg.Exposure.Tailnet.MobileOAuthClientSecret = clientSecret
	cfg.Exposure.Tailnet.MobileOAuthTags = splitCSV(tags)
	if len(cfg.Exposure.Tailnet.MobileOAuthTags) == 0 {
		cfg.Exposure.Tailnet.MobileOAuthTags = []string{"tag:codex-mobile"}
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		return cfg, err
	}

	fmt.Fprintln(promptOut)
	fmt.Fprintf(promptOut, "Saved tailnet mobile enrollment config to %s\n", configPath)
	fmt.Fprintln(promptOut, "codex-bridge will continue startup and use this OAuth client to generate short-lived mobile enrollment keys.")
	fmt.Fprintln(promptOut)
	return cfg, nil
}

func needsTailnetMobileEnrollmentBootstrap(cfg config.Config) bool {
	if cfg.Exposure.Mode != config.ExposureModeTailnet {
		return false
	}
	if strings.TrimSpace(cfg.Exposure.Tailnet.MobileAuthKey) != "" {
		return false
	}
	return strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthClientID) == "" ||
		strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthClientSecret) == "" ||
		strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthTailnet) == "" ||
		len(cfg.Exposure.Tailnet.MobileOAuthTags) == 0
}

func detectCurrentTailnetName(ctx context.Context) string {
	client := &local.Client{}
	status, err := client.StatusWithoutPeers(ctx)
	if err != nil || status == nil || status.CurrentTailnet == nil {
		return ""
	}
	return strings.TrimSpace(status.CurrentTailnet.Name)
}

func tryOpenBrowser(target string, logger *log.Logger) bool {
	if strings.TrimSpace(os.Getenv("SSH_CONNECTION")) != "" || strings.TrimSpace(os.Getenv("SSH_TTY")) != "" {
		return false
	}

	var command string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
	case "linux":
		command = "xdg-open"
	default:
		return false
	}
	if err := exec.Command(command, target).Start(); err != nil {
		if logger != nil {
			logger.Printf("%s [codex-bridge] failed to open browser for tailnet OAuth bootstrap: %v", time.Now().UTC().Format(time.RFC3339), err)
		}
		return false
	}
	return true
}

func promptYesNo(reader *bufio.Reader, out io.Writer, label string, defaultYes bool) (bool, error) {
	raw, err := promptString(reader, out, label, "")
	if err != nil {
		return false, err
	}
	if raw == "" {
		return defaultYes, nil
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return defaultYes, nil
	}
}

func promptString(reader *bufio.Reader, out io.Writer, label string, defaultValue string) (string, error) {
	if defaultValue != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	raw, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		return strings.TrimSpace(defaultValue), nil
	}
	return value, nil
}

func promptRequiredString(reader *bufio.Reader, out io.Writer, label string) (string, error) {
	for {
		value, err := promptString(reader, out, label, "")
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
		fmt.Fprintln(out, "This value is required.")
	}
}

func promptRequiredSecret(out io.Writer, label string) (string, error) {
	for {
		fmt.Fprintf(out, "%s: ", label)
		raw, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(out)
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(string(raw))
		if value != "" {
			return value, nil
		}
		fmt.Fprintln(out, "This value is required.")
	}
}

func splitCSV(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}
