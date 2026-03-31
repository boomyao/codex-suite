package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
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

const (
	tailscaleOAuthSettingsURL            = "https://login.tailscale.com/admin/settings/oauth"
	tailnetOAuthValidationRequestTimeout = 10 * time.Second
)

var (
	tailscaleOAuthTokenURL           = "https://api.tailscale.com/api/v2/oauth/token"
	tailnetOAuthValidationHTTPClient = http.DefaultClient
)

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
	clientSecret, err := promptRequiredSecret(reader, promptOut, "OAuth client secret")
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
	fmt.Fprintln(promptOut, "codex-bridge will continue startup and use this OAuth client for mobile enrollment recovery.")
	fmt.Fprintln(promptOut)
	return cfg, nil
}

func maybeRepairTailnetMobileEnrollmentOAuth(
	ctx context.Context,
	cfg config.Config,
	configPath string,
	promptOut *os.File,
	logger *log.Logger,
	validationErr error,
) (config.Config, bool, error) {
	if promptOut == nil {
		return cfg, false, nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(promptOut.Fd())) {
		return cfg, false, nil
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintln(promptOut, "codex-bridge detected invalid tailnet mobile enrollment OAuth credentials.")
	if validationErr != nil {
		fmt.Fprintf(promptOut, "Validation error: %v\n", validationErr)
	}
	fmt.Fprintln(promptOut, "Reconfigure the saved Tailscale OAuth client now to continue startup.")
	fmt.Fprintf(promptOut, "Open Tailscale OAuth settings: %s\n", tailscaleOAuthSettingsURL)
	fmt.Fprintln(promptOut)

	openedBrowser := tryOpenBrowser(tailscaleOAuthSettingsURL, logger)
	if openedBrowser {
		fmt.Fprintln(promptOut, "Opened Tailscale OAuth settings in your browser.")
	}

	proceed, err := promptYesNo(reader, promptOut, "Reconfigure tailnet mobile enrollment now? [Y/n]", true)
	if err != nil {
		return cfg, false, err
	}
	if !proceed {
		return cfg, false, nil
	}

	detectedTailnet := detectCurrentTailnetName(ctx)
	if detectedTailnet == "" {
		detectedTailnet = strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthTailnet)
	}
	clientIDDefault := strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthClientID)
	tagsDefault := strings.Join(cfg.Exposure.Tailnet.MobileOAuthTags, ",")
	if tagsDefault == "" {
		tagsDefault = "tag:codex-mobile"
	}

	tailnetName, err := promptRequiredStringWithDefault(reader, promptOut, "Tailnet name", detectedTailnet)
	if err != nil {
		return cfg, false, err
	}
	clientID, err := promptRequiredStringWithDefault(reader, promptOut, "OAuth client ID", clientIDDefault)
	if err != nil {
		return cfg, false, err
	}
	clientSecret, err := promptRequiredSecret(reader, promptOut, "OAuth client secret")
	if err != nil {
		return cfg, false, err
	}
	tags, err := promptRequiredStringWithDefault(reader, promptOut, "Mobile tags (comma-separated)", tagsDefault)
	if err != nil {
		return cfg, false, err
	}

	cfg.Exposure.Tailnet.MobileOAuthTailnet = tailnetName
	cfg.Exposure.Tailnet.MobileOAuthClientID = clientID
	cfg.Exposure.Tailnet.MobileOAuthClientSecret = clientSecret
	cfg.Exposure.Tailnet.MobileOAuthTags = splitCSV(tags)
	if len(cfg.Exposure.Tailnet.MobileOAuthTags) == 0 {
		cfg.Exposure.Tailnet.MobileOAuthTags = []string{"tag:codex-mobile"}
	}

	if err := config.SaveConfig(configPath, cfg); err != nil {
		return cfg, false, err
	}

	fmt.Fprintln(promptOut)
	fmt.Fprintf(promptOut, "Updated tailnet mobile enrollment config in %s\n", configPath)
	fmt.Fprintln(promptOut)
	return cfg, true, nil
}

func needsTailnetMobileEnrollmentBootstrap(cfg config.Config) bool {
	if cfg.Exposure.Mode != config.ExposureModeTailnet {
		return false
	}
	return strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthClientID) == "" ||
		strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthClientSecret) == "" ||
		strings.TrimSpace(cfg.Exposure.Tailnet.MobileOAuthTailnet) == "" ||
		len(cfg.Exposure.Tailnet.MobileOAuthTags) == 0
}

func validateTailnetMobileEnrollmentOAuth(ctx context.Context, cfg config.Config) error {
	if cfg.Exposure.Mode != config.ExposureModeTailnet {
		return nil
	}

	tailnetCfg := cfg.Exposure.Tailnet
	hasOAuthField := strings.TrimSpace(tailnetCfg.MobileOAuthClientID) != "" ||
		strings.TrimSpace(tailnetCfg.MobileOAuthClientSecret) != "" ||
		strings.TrimSpace(tailnetCfg.MobileOAuthTailnet) != "" ||
		len(tailnetCfg.MobileOAuthTags) > 0
	if !hasOAuthField {
		return nil
	}

	clientID := strings.TrimSpace(tailnetCfg.MobileOAuthClientID)
	clientSecret := strings.TrimSpace(tailnetCfg.MobileOAuthClientSecret)
	if clientID == "" || clientSecret == "" {
		return nil
	}
	if !looksLikeTailscaleOAuthClientSecret(clientSecret) {
		return fmt.Errorf(
			"configured exposure.tailnet.mobileOAuthClientSecret does not look like a valid Tailscale OAuth client secret; expected a tskey-client secret",
		)
	}

	requestCtx, cancel := context.WithTimeout(ctx, tailnetOAuthValidationRequestTimeout)
	defer cancel()

	form := neturl.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(
		requestCtx,
		http.MethodPost,
		tailscaleOAuthTokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("build Tailscale OAuth validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tailnetOAuthValidationHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request Tailscale OAuth token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	if err != nil {
		return fmt.Errorf("read Tailscale OAuth validation response: %w", err)
	}

	var payload struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &payload)

	if resp.StatusCode != http.StatusOK {
		message := strings.TrimSpace(payload.ErrorDescription)
		if message == "" {
			message = strings.TrimSpace(payload.Error)
		}
		if message == "" {
			message = strings.TrimSpace(string(body))
		}
		if message == "" {
			message = resp.Status
		}
		return fmt.Errorf("Tailscale OAuth token exchange failed: %s", message)
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return fmt.Errorf("Tailscale OAuth token exchange succeeded but did not return an access_token")
	}
	return nil
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

func promptRequiredStringWithDefault(reader *bufio.Reader, out io.Writer, label string, defaultValue string) (string, error) {
	for {
		value, err := promptString(reader, out, label, defaultValue)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
		fmt.Fprintln(out, "This value is required.")
	}
}

func promptRequiredSecret(reader *bufio.Reader, out io.Writer, label string) (string, error) {
	for {
		fmt.Fprintf(out, "%s: ", label)
		raw, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(out)
		if err != nil {
			return "", err
		}
		value := strings.TrimSpace(string(raw))
		switch {
		case value == "":
			fmt.Fprintln(out, "This value is required.")
		case !looksLikeTailscaleOAuthClientSecret(value):
			fmt.Fprintf(out, "%s does not look like a valid Tailscale OAuth client secret.\n", label)
			fmt.Fprintln(out, "Tailscale OAuth client secrets normally start with 'tskey-client'.")
			fmt.Fprintln(out, "If your terminal captured paste control codes, paste the secret again using the visible fallback prompt below.")
			visibleValue, visibleErr := promptRequiredString(reader, out, label+" (visible fallback)")
			if visibleErr != nil {
				return "", visibleErr
			}
			visibleValue = strings.TrimSpace(visibleValue)
			if looksLikeTailscaleOAuthClientSecret(visibleValue) {
				return visibleValue, nil
			}
			fmt.Fprintf(out, "%s still does not look valid.\n", label)
		default:
			return value, nil
		}
	}
}

func looksLikeTailscaleOAuthClientSecret(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "tskey-client") {
		return false
	}
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
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
