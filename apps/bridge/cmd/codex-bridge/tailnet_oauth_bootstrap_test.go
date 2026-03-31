package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/boomyao/codex-bridge/internal/config"
)

func TestValidateTailnetMobileEnrollmentOAuthSkipsWhenUnconfigured(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Exposure.Mode = config.ExposureModeTailnet

	if err := validateTailnetMobileEnrollmentOAuth(context.Background(), cfg); err != nil {
		t.Fatalf("expected unconfigured mobile OAuth validation to be skipped, got %v", err)
	}
}

func TestValidateTailnetMobileEnrollmentOAuthRejectsMalformedSecret(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Exposure.Mode = config.ExposureModeTailnet
	cfg.Exposure.Tailnet.MobileOAuthClientID = "kKbVpduCZ521CNTRL"
	cfg.Exposure.Tailnet.MobileOAuthClientSecret = "\u001bqf"
	cfg.Exposure.Tailnet.MobileOAuthTailnet = "example.ts.net"
	cfg.Exposure.Tailnet.MobileOAuthTags = []string{"tag:codex-mobile"}

	err := validateTailnetMobileEnrollmentOAuth(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected malformed OAuth client secret validation error")
	}
	if !strings.Contains(err.Error(), "tskey-client") {
		t.Fatalf("expected malformed secret error to mention tskey-client, got %v", err)
	}
}

func TestValidateTailnetMobileEnrollmentOAuthExchangesToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("client_id"); got != "kKbVpduCZ521CNTRL" {
			t.Fatalf("unexpected client_id %q", got)
		}
		if got := r.Form.Get("client_secret"); got != "tskey-client-kKbVpduCZ521CNTRL-abcdef" {
			t.Fatalf("unexpected client_secret %q", got)
		}
		if got := r.Form.Get("grant_type"); got != "client_credentials" {
			t.Fatalf("unexpected grant_type %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"tskey-0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ","token_type":"Bearer","expires_in":3600}`))
	}))
	defer server.Close()

	previousTokenURL := tailscaleOAuthTokenURL
	previousHTTPClient := tailnetOAuthValidationHTTPClient
	tailscaleOAuthTokenURL = server.URL
	tailnetOAuthValidationHTTPClient = server.Client()
	t.Cleanup(func() {
		tailscaleOAuthTokenURL = previousTokenURL
		tailnetOAuthValidationHTTPClient = previousHTTPClient
	})

	cfg := config.DefaultConfig()
	cfg.Exposure.Mode = config.ExposureModeTailnet
	cfg.Exposure.Tailnet.MobileOAuthClientID = "kKbVpduCZ521CNTRL"
	cfg.Exposure.Tailnet.MobileOAuthClientSecret = "tskey-client-kKbVpduCZ521CNTRL-abcdef"
	cfg.Exposure.Tailnet.MobileOAuthTailnet = "example.ts.net"
	cfg.Exposure.Tailnet.MobileOAuthTags = []string{"tag:codex-mobile"}

	if err := validateTailnetMobileEnrollmentOAuth(context.Background(), cfg); err != nil {
		t.Fatalf("expected OAuth validation success, got %v", err)
	}
}

func TestValidateTailnetMobileEnrollmentOAuthSurfacesTokenExchangeErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_client","error_description":"invalid client credentials"}`))
	}))
	defer server.Close()

	previousTokenURL := tailscaleOAuthTokenURL
	previousHTTPClient := tailnetOAuthValidationHTTPClient
	tailscaleOAuthTokenURL = server.URL
	tailnetOAuthValidationHTTPClient = server.Client()
	t.Cleanup(func() {
		tailscaleOAuthTokenURL = previousTokenURL
		tailnetOAuthValidationHTTPClient = previousHTTPClient
	})

	cfg := config.DefaultConfig()
	cfg.Exposure.Mode = config.ExposureModeTailnet
	cfg.Exposure.Tailnet.MobileOAuthClientID = "kKbVpduCZ521CNTRL"
	cfg.Exposure.Tailnet.MobileOAuthClientSecret = "tskey-client-kKbVpduCZ521CNTRL-abcdef"
	cfg.Exposure.Tailnet.MobileOAuthTailnet = "example.ts.net"
	cfg.Exposure.Tailnet.MobileOAuthTags = []string{"tag:codex-mobile"}

	err := validateTailnetMobileEnrollmentOAuth(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected OAuth validation failure")
	}
	if !strings.Contains(err.Error(), "invalid client credentials") {
		t.Fatalf("expected token exchange error detail, got %v", err)
	}
}
