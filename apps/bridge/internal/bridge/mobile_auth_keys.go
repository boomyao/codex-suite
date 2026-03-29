package bridge

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2/clientcredentials"
	tsclient "tailscale.com/client/tailscale"
)

const (
	defaultTailscaleAPIBaseURL           = "https://api.tailscale.com"
	defaultMobileAuthKeyExpirySeconds    = 600
	mobileAuthKeyRefreshGracePeriod      = 30 * time.Second
	mobileAuthKeyProvisionRequestTimeout = 10 * time.Second
)

type mobileAuthKeyProvider struct {
	logger *log.Logger

	mu      sync.Mutex
	config  MobileEnrollmentConfig
	cached  string
	expires time.Time
}

func newMobileAuthKeyProvider(config MobileEnrollmentConfig, logger *log.Logger) *mobileAuthKeyProvider {
	if logger == nil {
		logger = log.Default()
	}
	return &mobileAuthKeyProvider{
		config: config,
		logger: logger,
	}
}

func (c MobileEnrollmentConfig) canProvisionTailnetAuthKey() bool {
	if strings.TrimSpace(c.AuthKey) != "" {
		return true
	}
	return c.hasOAuthProvisioner() || c.hasAPIAccessTokenProvisioner()
}

func (c MobileEnrollmentConfig) hasOAuthProvisioner() bool {
	return strings.TrimSpace(c.OAuthClientID) != "" &&
		strings.TrimSpace(c.OAuthClientSecret) != "" &&
		strings.TrimSpace(c.OAuthTailnet) != "" &&
		len(c.OAuthTags) > 0
}

func (c MobileEnrollmentConfig) hasAPIAccessTokenProvisioner() bool {
	return strings.TrimSpace(c.APIAccessToken) != ""
}

func (c MobileEnrollmentConfig) authKeyExpiry() time.Duration {
	expirySeconds := c.AuthKeyExpirySeconds
	if expirySeconds <= 0 {
		expirySeconds = defaultMobileAuthKeyExpirySeconds
	}
	return time.Duration(expirySeconds) * time.Second
}

func (p *mobileAuthKeyProvider) Enabled() bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.config.canProvisionTailnetAuthKey()
}

func (p *mobileAuthKeyProvider) Resolve(ctx context.Context) (string, error) {
	if p == nil {
		return "", nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if authKey := strings.TrimSpace(p.config.AuthKey); authKey != "" {
		return authKey, nil
	}
	if !p.config.hasOAuthProvisioner() && !p.config.hasAPIAccessTokenProvisioner() {
		return "", nil
	}

	authKey, expiresAt, err := p.provision(ctx)
	if err != nil {
		return "", err
	}
	// Mobile enrollment currently provisions non-reusable auth keys. Reusing
	// a cached key across multiple QR generations makes later scans fail after
	// the first enrollment attempt consumes the key.
	p.cached = ""
	p.expires = expiresAt
	return authKey, nil
}

func (p *mobileAuthKeyProvider) UpdateConfig(config MobileEnrollmentConfig) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
	p.cached = ""
	p.expires = time.Time{}
}

func (p *mobileAuthKeyProvider) provision(ctx context.Context) (string, time.Time, error) {
	if p.config.hasOAuthProvisioner() {
		return p.provisionWithOAuth(ctx)
	}
	if p.config.hasAPIAccessTokenProvisioner() {
		return p.provisionWithAPIAccessToken(ctx)
	}
	return "", time.Time{}, nil
}

func (p *mobileAuthKeyProvider) provisionWithOAuth(ctx context.Context) (string, time.Time, error) {
	requestCtx, cancel := context.WithTimeout(ctx, mobileAuthKeyProvisionRequestTimeout)
	defer cancel()

	credentials := clientcredentials.Config{
		ClientID:     strings.TrimSpace(p.config.OAuthClientID),
		ClientSecret: strings.TrimSpace(p.config.OAuthClientSecret),
		TokenURL:     defaultTailscaleAPIBaseURL + "/api/v2/oauth/token",
	}

	tsclient.I_Acknowledge_This_API_Is_Unstable = true
	client := tsclient.NewClient(strings.TrimSpace(p.config.OAuthTailnet), nil)
	client.BaseURL = defaultTailscaleAPIBaseURL
	client.UserAgent = "codex-bridge"
	client.HTTPClient = credentials.Client(requestCtx)

	caps := tsclient.KeyCapabilities{
		Devices: tsclient.KeyDeviceCapabilities{
			Create: tsclient.KeyDeviceCreateCapabilities{
				Reusable:      false,
				Ephemeral:     true,
				Preauthorized: true,
				Tags:          append([]string{}, p.config.OAuthTags...),
			},
		},
	}

	authKey, meta, err := client.CreateKeyWithExpiry(requestCtx, caps, p.config.authKeyExpiry())
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to provision tailscale mobile auth key: %w", err)
	}

	expiresAt := time.Now().Add(p.config.authKeyExpiry())
	if meta != nil && !meta.Expires.IsZero() {
		expiresAt = meta.Expires
	}

	p.logger.Printf("%s [codex-bridge] provisioned tailscale mobile auth key expiring at %s", nowISO(), expiresAt.UTC().Format(time.RFC3339))
	return authKey, expiresAt, nil
}

func (p *mobileAuthKeyProvider) provisionWithAPIAccessToken(ctx context.Context) (string, time.Time, error) {
	requestCtx, cancel := context.WithTimeout(ctx, mobileAuthKeyProvisionRequestTimeout)
	defer cancel()

	tsclient.I_Acknowledge_This_API_Is_Unstable = true
	client := tsclient.NewClient("-", tsclient.APIKey(strings.TrimSpace(p.config.APIAccessToken)))
	client.BaseURL = defaultTailscaleAPIBaseURL
	client.UserAgent = "codex-bridge"

	caps := tsclient.KeyCapabilities{
		Devices: tsclient.KeyDeviceCapabilities{
			Create: tsclient.KeyDeviceCreateCapabilities{
				Reusable:      false,
				Ephemeral:     true,
				Preauthorized: true,
				Tags:          append([]string{}, p.config.OAuthTags...),
			},
		},
	}

	authKey, meta, err := client.CreateKeyWithExpiry(requestCtx, caps, p.config.authKeyExpiry())
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to provision tailscale mobile auth key with API access token: %w", err)
	}

	expiresAt := time.Now().Add(p.config.authKeyExpiry())
	if meta != nil && !meta.Expires.IsZero() {
		expiresAt = meta.Expires
	}

	p.logger.Printf("%s [codex-bridge] provisioned tailscale mobile auth key expiring at %s", nowISO(), expiresAt.UTC().Format(time.RFC3339))
	return authKey, expiresAt, nil
}
