package bridge

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/boomyao/codex-bridge/internal/auth"
)

const LocalMobileEnrollmentPath = "/auth/mobile-enrollment"

func (b *Bridge) handleLocalMobileEnrollment(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != LocalMobileEnrollmentPath {
		return false
	}
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "Method Not Allowed"})
		return true
	}
	if !isLoopbackHTTPRequest(r) {
		sendJSON(w, http.StatusForbidden, map[string]any{"ok": false, "error": "Mobile enrollment requires local access"})
		return true
	}

	var pairing *auth.PairingInfo

	b.mu.Lock()
	authorizer := b.authorizer
	b.mu.Unlock()
	if deviceTokenAuthorizer, ok := authorizer.(*auth.DeviceTokenAuthorizer); ok {
		info, err := deviceTokenAuthorizer.GeneratePairingCode()
		if err != nil {
			sendJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
			return true
		}
		pairing = &info
	}

	payload, err := b.BuildMobileEnrollmentPayload(pairingCodeValue(pairing))
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return true
	}

	sendJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"pairing": pairing,
		"payload": payload,
	})
	return true
}

func pairingCodeValue(info *auth.PairingInfo) string {
	if info == nil {
		return ""
	}
	return strings.TrimSpace(info.Code)
}

func isLoopbackHTTPRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (b *Bridge) BuildMobileEnrollmentPayload(pairingCode string) (map[string]any, error) {
	_, exposureStatus, authState, info, _ := b.snapshotStatus()
	return b.buildMobileEnrollmentPayload(
		buildConnectionTarget(b.config.BridgeID, exposureStatus, info),
		exposureStatus,
		authState,
		pairingCode,
	)
}

func (b *Bridge) buildMobileEnrollmentPayload(
	connection map[string]any,
	exposureStatus ExposureStatus,
	authState auth.State,
	pairingCode string,
) (map[string]any, error) {
	endpoint, _ := connection["recommendedServerEndpoint"].(string)
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("no reachable bridge endpoint is available for mobile enrollment")
	}

	payload := map[string]any{
		"type":           "codex-mobile-bridge",
		"version":        1,
		"bridgeId":       strings.TrimSpace(connectionStringValue(connection, "bridgeId")),
		"name":           deriveEnrollmentBridgeName(endpoint),
		"serverEndpoint": strings.TrimSpace(endpoint),
		"authMode":       strings.TrimSpace(authState.Mode),
	}
	if code := strings.TrimSpace(pairingCode); code != "" {
		payload["pairingCode"] = code
	}
	return payload, nil
}

func connectionStringValue(connection map[string]any, key string) string {
	if connection == nil {
		return ""
	}
	value, _ := connection[key].(string)
	return strings.TrimSpace(value)
}

func deriveEnrollmentBridgeName(endpoint string) string {
	normalized := strings.TrimSpace(endpoint)
	if normalized == "" {
		return "Codex Bridge"
	}
	httpURL := normalized
	if strings.HasPrefix(httpURL, "ws://") {
		httpURL = "http://" + strings.TrimPrefix(httpURL, "ws://")
	} else if strings.HasPrefix(httpURL, "wss://") {
		httpURL = "https://" + strings.TrimPrefix(httpURL, "wss://")
	}
	parsed, err := url.Parse(httpURL)
	if err == nil && strings.TrimSpace(parsed.Host) != "" {
		return parsed.Host
	}
	return "Codex Bridge"
}
