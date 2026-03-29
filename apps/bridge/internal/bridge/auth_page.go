package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/boomyao/codex-bridge/internal/auth"
	qrcode "github.com/skip2/go-qrcode"
)

const localAuthPagePath = "/auth/local"
const localQRCodePath = "/auth/mobile-qr.png"

func localAuthPageURL(state auth.State) string {
	if state.Mode != "device-token" {
		return ""
	}
	return localAuthPagePath
}

func (b *Bridge) handleLocalAuthPage(w http.ResponseWriter, r *http.Request) bool {
	if b.handleLocalQRCode(w, r) {
		return true
	}
	if r.URL.Path != localAuthPagePath {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
		return true
	}
	if !isLoopbackHTTPRequest(r) {
		sendText(w, http.StatusForbidden, "Local auth page requires loopback access.\n")
		return true
	}

	_, exposureStatus, authState, info, tailnetBootstrapStatusProvider := b.snapshotStatus()
	var tailnetBootstrap map[string]any
	if tailnetBootstrapStatusProvider != nil {
		tailnetBootstrap = tailnetBootstrapStatusProvider()
	}
	bootstrap := map[string]any{
		"auth":       authState,
		"connection": buildConnectionTarget(exposureStatus, info),
		"mobileEnrollment": map[string]any{
			"tailnetEnabled": b.tailnetEnrollmentAvailable(exposureStatus),
		},
		"paths": map[string]string{
			"status":        "/status",
			"connect":       "/codex-mobile/connect",
			"pairStart":     "/auth/pair/start",
			"devices":       "/auth/devices",
			"localAuthPage": localAuthPagePath,
			"mobileQR":      localQRCodePath,
		},
	}
	if tailnetBootstrap != nil {
		bootstrap["tailnetBootstrap"] = tailnetBootstrap
	}
	body := []byte(buildLocalAuthPageHTML(bootstrap))
	sendBody(w, r, http.StatusOK, body, "text/html; charset=utf-8")
	return true
}

func isLoopbackHTTPRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (b *Bridge) handleLocalQRCode(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != localQRCodePath {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
		return true
	}
	if !isLoopbackHTTPRequest(r) {
		sendText(w, http.StatusForbidden, "Local QR endpoint requires loopback access.\n")
		return true
	}

	payload, err := b.BuildMobileEnrollmentPayload(r.URL.Query().Get("code"))
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
		return true
	}
	body, err := json.Marshal(payload)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "Failed to encode mobile enrollment payload.",
		})
		return true
	}

	image, err := qrcode.Encode(string(body), qrcode.Medium, 320)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, map[string]any{
			"ok":    false,
			"error": "Failed to generate QR code.",
		})
		return true
	}

	sendBody(w, r, http.StatusOK, image, "image/png")
	return true
}

func (b *Bridge) BuildMobileEnrollmentPayload(pairingCode string) (map[string]any, error) {
	_, exposureStatus, authState, info, _ := b.snapshotStatus()
	return b.buildMobileEnrollmentPayload(
		buildConnectionTarget(exposureStatus, info),
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

	useTailnetEnrollment := b.tailnetEnrollmentAvailable(exposureStatus)
	authKey, err := b.authKeys.Resolve(context.Background())
	if err != nil {
		return nil, err
	}
	if useTailnetEnrollment && strings.TrimSpace(authKey) != "" {
		payload := map[string]any{
			"type":                 "codex-mobile-enrollment",
			"version":              1,
			"bridgeName":           deriveEnrollmentBridgeName(endpoint),
			"bridgeServerEndpoint": strings.TrimSpace(endpoint),
			"tailnet": map[string]any{
				"authKey": strings.TrimSpace(authKey),
			},
		}
		if controlURL := strings.TrimSpace(b.config.MobileEnrollment.ControlURL); controlURL != "" {
			payload["tailnet"].(map[string]any)["controlUrl"] = controlURL
		}
		if hostname := strings.TrimSpace(b.config.MobileEnrollment.Hostname); hostname != "" {
			payload["tailnet"].(map[string]any)["hostname"] = hostname
		}
		if loginMode := strings.TrimSpace(b.config.MobileEnrollment.LoginMode); loginMode != "" {
			payload["tailnet"].(map[string]any)["loginMode"] = loginMode
		}
		if code := strings.TrimSpace(pairingCode); code != "" {
			payload["pairingCode"] = code
		}
		return payload, nil
	}

	payload := map[string]any{
		"type":           "codex-mobile-bridge",
		"version":        1,
		"name":           deriveEnrollmentBridgeName(endpoint),
		"serverEndpoint": strings.TrimSpace(endpoint),
		"authMode":       strings.TrimSpace(authState.Mode),
	}
	if code := strings.TrimSpace(pairingCode); code != "" {
		payload["pairingCode"] = code
	}
	return payload, nil
}

func (b *Bridge) tailnetEnrollmentAvailable(exposureStatus ExposureStatus) bool {
	if b == nil || b.authKeys == nil || !b.authKeys.Enabled() {
		return false
	}
	if exposureStatus.Mode == "tailnet" && !exposureStatus.Ready {
		return false
	}
	return true
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

func buildLocalAuthPageHTML(bootstrap map[string]any) string {
	serialized, _ := json.Marshal(bootstrap)
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>Codex Bridge Access</title>
    <style>
      :root {
        color-scheme: dark;
        --bg: #0b1312;
        --panel: rgba(18, 29, 28, 0.92);
        --panel-border: rgba(172, 206, 179, 0.18);
        --text: #edf5ef;
        --muted: #9db0a2;
        --accent: #80d7a5;
        --accent-strong: #5ac48b;
        --danger: #ff8e88;
        --warn: #ffd37b;
        --shadow: 0 20px 60px rgba(0, 0, 0, 0.34);
        --radius: 22px;
      }

      * {
        box-sizing: border-box;
      }

      body {
        margin: 0;
        min-height: 100vh;
        font-family: "Iosevka Aile", "IBM Plex Sans", "SF Pro Display", sans-serif;
        color: var(--text);
        background:
          radial-gradient(circle at top left, rgba(70, 123, 88, 0.28), transparent 28%%),
          radial-gradient(circle at top right, rgba(53, 91, 120, 0.24), transparent 24%%),
          linear-gradient(180deg, #0a1110 0%%, #0f1918 45%%, #0c1312 100%%);
      }

      main {
        width: min(1100px, calc(100vw - 32px));
        margin: 0 auto;
        padding: 40px 0 64px;
      }

      .hero {
        display: grid;
        gap: 14px;
        margin-bottom: 22px;
      }

      .eyebrow {
        font-size: 12px;
        font-weight: 700;
        letter-spacing: 0.18em;
        text-transform: uppercase;
        color: var(--accent);
      }

      h1 {
        margin: 0;
        font-size: clamp(34px, 5vw, 56px);
        line-height: 0.94;
        letter-spacing: -0.04em;
      }

      .hero p {
        margin: 0;
        max-width: 720px;
        font-size: 15px;
        line-height: 1.65;
        color: var(--muted);
      }

      .grid {
        display: grid;
        gap: 18px;
        grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      }

      .panel {
        background: var(--panel);
        border: 1px solid var(--panel-border);
        border-radius: var(--radius);
        box-shadow: var(--shadow);
        padding: 20px;
        backdrop-filter: blur(14px);
      }

      .panel h2 {
        margin: 0 0 10px;
        font-size: 18px;
        letter-spacing: -0.02em;
      }

      .meta {
        display: grid;
        gap: 10px;
      }

      .meta-row {
        display: flex;
        flex-wrap: wrap;
        justify-content: space-between;
        gap: 12px;
        padding-top: 10px;
        border-top: 1px solid rgba(255, 255, 255, 0.06);
      }

      .meta-row:first-child {
        border-top: 0;
        padding-top: 0;
      }

      .meta-label {
        font-size: 12px;
        font-weight: 700;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--muted);
      }

      .meta-value {
        font-family: "Berkeley Mono", "SFMono-Regular", "Menlo", monospace;
        font-size: 13px;
        text-align: right;
        word-break: break-all;
      }

      .actions {
        display: flex;
        flex-wrap: wrap;
        gap: 10px;
        margin-top: 16px;
      }

      button {
        appearance: none;
        border: 0;
        border-radius: 999px;
        padding: 11px 16px;
        font: inherit;
        font-weight: 700;
        cursor: pointer;
        transition: transform 120ms ease, opacity 120ms ease, background 120ms ease;
      }

      button:hover {
        transform: translateY(-1px);
      }

      button:disabled {
        opacity: 0.58;
        cursor: default;
        transform: none;
      }

      .primary {
        background: linear-gradient(135deg, var(--accent-strong), #9ce6bc);
        color: #082213;
      }

      .secondary {
        background: rgba(255, 255, 255, 0.06);
        color: var(--text);
      }

      .danger {
        background: rgba(255, 142, 136, 0.18);
        color: var(--danger);
      }

      .pair-code {
        margin-top: 18px;
        padding: 18px;
        border-radius: 18px;
        background: linear-gradient(180deg, rgba(128, 215, 165, 0.11), rgba(128, 215, 165, 0.04));
        border: 1px solid rgba(128, 215, 165, 0.24);
      }

      .pair-code.hidden {
        display: none;
      }

      .pair-label {
        font-size: 12px;
        font-weight: 700;
        letter-spacing: 0.12em;
        text-transform: uppercase;
        color: var(--muted);
      }

      .pair-value {
        margin-top: 8px;
        font-family: "Berkeley Mono", "SFMono-Regular", "Menlo", monospace;
        font-size: clamp(28px, 6vw, 44px);
        letter-spacing: 0.18em;
      }

      .hint, .flash {
        font-size: 14px;
        line-height: 1.6;
        color: var(--muted);
      }

      .flash {
        min-height: 22px;
        margin-top: 14px;
      }

      .flash[data-tone="error"] {
        color: var(--danger);
      }

      .flash[data-tone="warning"] {
        color: var(--warn);
      }

      table {
        width: 100%%;
        border-collapse: collapse;
        margin-top: 8px;
      }

      th, td {
        text-align: left;
        vertical-align: top;
        padding: 12px 0;
        border-top: 1px solid rgba(255, 255, 255, 0.08);
        font-size: 14px;
      }

      th {
        font-size: 12px;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        color: var(--muted);
      }

      td code {
        font-family: "Berkeley Mono", "SFMono-Regular", "Menlo", monospace;
        font-size: 12px;
      }

      .device-name {
        font-weight: 700;
      }

      .badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 6px 9px;
        border-radius: 999px;
        font-size: 12px;
        font-weight: 700;
        letter-spacing: 0.04em;
      }

      .badge.ready {
        background: rgba(128, 215, 165, 0.12);
        color: var(--accent);
      }

      .badge.pending {
        background: rgba(255, 211, 123, 0.14);
        color: var(--warn);
      }

      .badge.revoked {
        background: rgba(255, 142, 136, 0.16);
        color: var(--danger);
      }

      .empty {
        margin-top: 14px;
        padding: 18px;
        border-radius: 18px;
        border: 1px dashed rgba(255, 255, 255, 0.12);
        color: var(--muted);
      }

      .qr-shell {
        display: grid;
        gap: 14px;
        margin-top: 14px;
      }

      .qr-frame {
        width: min(100%%, 320px);
        aspect-ratio: 1;
        border-radius: 20px;
        padding: 14px;
        background: rgba(255, 255, 255, 0.98);
        box-shadow: inset 0 0 0 1px rgba(8, 18, 14, 0.08);
      }

      .qr-frame img {
        display: block;
        width: 100%%;
        height: 100%%;
        border-radius: 12px;
      }

      .device-actions {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
      }

      @media (max-width: 720px) {
        main {
          width: min(100vw - 20px, 1100px);
          padding-top: 24px;
        }

        .panel {
          padding: 16px;
          border-radius: 18px;
        }

        th:nth-child(3),
        td:nth-child(3) {
          display: none;
        }
      }
    </style>
  </head>
  <body>
    <main>
      <section class="hero">
        <div class="eyebrow">Codex Bridge</div>
        <h1>Local Access Control</h1>
        <p>
          Generate a pairing code for Codex Mobile, inspect approved devices, and revoke stale
          tokens without exposing these controls over the public bridge endpoint.
        </p>
      </section>

      <section class="grid">
        <article class="panel">
          <h2>Bridge Session</h2>
          <div class="meta" id="connection-meta"></div>
          <div class="actions">
            <button class="secondary" id="refresh-status" type="button">Refresh Status</button>
            <button class="secondary" id="copy-endpoint" type="button">Copy Endpoint</button>
          </div>
        </article>

        <article class="panel">
          <h2>Pair Device</h2>
          <div class="hint" id="pairing-hint"></div>
          <div class="actions">
            <button class="primary" id="start-pairing" type="button">Generate Pairing Code</button>
            <button class="secondary" id="copy-pairing" type="button" disabled>Copy Code</button>
          </div>
          <div class="pair-code hidden" id="pairing-box">
            <div class="pair-label">Current Pairing Code</div>
            <div class="pair-value" id="pairing-code">--------</div>
            <div class="hint" id="pairing-expiry"></div>
          </div>
          <div class="flash" id="pairing-flash"></div>
        </article>

        <article class="panel">
          <h2>Scan in Codex Mobile</h2>
          <div class="hint" id="qr-hint"></div>
          <div class="qr-shell">
            <div class="qr-frame hidden" id="qr-frame">
              <img alt="Codex Mobile enrollment QR code" id="mobile-qr" />
            </div>
            <div class="empty" id="qr-empty"></div>
          </div>
        </article>
      </section>

      <section class="panel" style="margin-top: 18px;">
        <h2>Registered Devices</h2>
        <div class="actions">
          <button class="secondary" id="refresh-devices" type="button">Refresh Devices</button>
        </div>
        <div class="flash" id="devices-flash"></div>
        <div id="device-list"></div>
      </section>
    </main>

    <script>
      const BOOTSTRAP = %s;
      const pairStartPath = BOOTSTRAP.paths.pairStart;
      const devicesPath = BOOTSTRAP.paths.devices;
      const connectInfo = BOOTSTRAP.connection || {};
      const authInfo = BOOTSTRAP.auth || {};
      const state = { pairing: null, devices: [] };

      const connectionMeta = document.getElementById("connection-meta");
      const pairingHint = document.getElementById("pairing-hint");
      const pairingBox = document.getElementById("pairing-box");
      const pairingCode = document.getElementById("pairing-code");
      const pairingExpiry = document.getElementById("pairing-expiry");
      const pairingFlash = document.getElementById("pairing-flash");
      const qrHint = document.getElementById("qr-hint");
      const qrFrame = document.getElementById("qr-frame");
      const qrImage = document.getElementById("mobile-qr");
      const qrEmpty = document.getElementById("qr-empty");
      const devicesFlash = document.getElementById("devices-flash");
      const deviceList = document.getElementById("device-list");
      const startPairingButton = document.getElementById("start-pairing");
      const copyPairingButton = document.getElementById("copy-pairing");
      const copyEndpointButton = document.getElementById("copy-endpoint");

      function formatDate(value) {
        if (!value) return "n/a";
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) return String(value);
        return date.toLocaleString();
      }

      function setFlash(element, message, tone) {
        element.textContent = message || "";
        element.dataset.tone = tone || "";
      }

      function escapeHtml(value) {
        return String(value == null ? "" : value)
          .replaceAll("&", "&amp;")
          .replaceAll("<", "&lt;")
          .replaceAll(">", "&gt;")
          .replaceAll('"', "&quot;")
          .replaceAll("'", "&#39;");
      }

      async function copyText(value, successMessage) {
        if (!value) return;
        try {
          await navigator.clipboard.writeText(value);
          setFlash(pairingFlash, successMessage, "");
        } catch (error) {
          setFlash(pairingFlash, error instanceof Error ? error.message : "Copy failed.", "error");
        }
      }

      async function fetchJSON(path, options) {
        const response = await fetch(path, options);
        const payload = await response.json().catch(() => null);
        if (!response.ok) {
          const message = payload && typeof payload.error === "string" ? payload.error : "Request failed.";
          throw new Error(message);
        }
        return payload || {};
      }

      function renderConnectionMeta() {
        const recommended = connectInfo.recommendedServerEndpoint || "n/a";
        const exposure = connectInfo.exposureServerEndpoint || "n/a";
        const source = connectInfo.source || "gateway";
        const approval = authInfo.requireApproval ? "manual approval required" : "auto approve";
        connectionMeta.innerHTML = [
          ["Auth Mode", authInfo.mode || "none"],
          ["Approval", approval],
          ["Recommended Endpoint", recommended],
          ["Exposure Endpoint", exposure],
          ["Endpoint Source", source],
        ].map(([label, value]) => [
          '<div class="meta-row">',
          '<div class="meta-label">' + escapeHtml(label) + '</div>',
          '<div class="meta-value">' + escapeHtml(value) + '</div>',
          '</div>'
        ].join("")).join("");

        pairingHint.textContent = authInfo.mode === "device-token"
          ? "Use this page on the bridge host. Generate one code, enter it in the mobile settings page, then approve the device if approval is enabled."
          : "Device pairing is disabled because auth mode is not set to device-token.";
        startPairingButton.disabled = authInfo.mode !== "device-token";
        copyEndpointButton.disabled = !recommended || recommended === "n/a";
      }

      function statusBadge(device) {
        if (device.revokedAt) {
          return '<span class="badge revoked">Revoked</span>';
        }
        if (!device.approved) {
          return '<span class="badge pending">Pending</span>';
        }
        return '<span class="badge ready">Approved</span>';
      }

      function renderDevices() {
        if (!Array.isArray(state.devices) || state.devices.length === 0) {
          deviceList.innerHTML = '<div class="empty">No devices have paired yet.</div>';
          return;
        }

        deviceList.innerHTML = [
          "<table>",
          "<thead><tr><th>Device</th><th>Status</th><th>Last Seen</th><th>Actions</th></tr></thead>",
          "<tbody>",
          state.devices.map((device) => {
            const deviceName = typeof device.name === "string" && device.name ? device.name : "Unnamed device";
            const deviceID = typeof device.id === "string" ? device.id : "";
            const canApprove = !device.approved && !device.revokedAt;
            const canRevoke = !device.revokedAt;
            return [
              "<tr>",
              '<td><div class="device-name">' + escapeHtml(deviceName) + '</div><code>' + escapeHtml(deviceID) + "</code></td>",
              "<td>" + statusBadge(device) + "</td>",
              "<td>" + escapeHtml(formatDate(device.lastSeenAt)) + "</td>",
              '<td><div class="device-actions">',
              '<button class="secondary" type="button" data-action="approve" data-id="' + escapeHtml(deviceID) + '"' + (canApprove ? "" : " disabled") + ">Approve</button>",
              '<button class="danger" type="button" data-action="revoke" data-id="' + escapeHtml(deviceID) + '"' + (canRevoke ? "" : " disabled") + ">Revoke</button>",
              "</div></td>",
              "</tr>",
            ].join("");
          }).join(""),
          "</tbody>",
          "</table>",
        ].join("");
      }

      function renderPairing() {
        if (!state.pairing || !state.pairing.code) {
          pairingBox.classList.add("hidden");
          copyPairingButton.disabled = true;
          renderEnrollmentQR();
          return;
        }
        pairingBox.classList.remove("hidden");
        pairingCode.textContent = state.pairing.code;
        pairingExpiry.textContent = "Expires at " + formatDate(state.pairing.expiresAt);
        copyPairingButton.disabled = false;
        renderEnrollmentQR();
      }

      function renderEnrollmentQR() {
	        const requiresPairing = authInfo.mode === "device-token";
	        const tailnetEnrollmentEnabled = !!(BOOTSTRAP.mobileEnrollment && BOOTSTRAP.mobileEnrollment.tailnetEnabled);
	        const currentCode = state.pairing && typeof state.pairing.code === "string"
	          ? state.pairing.code
	          : "";

	        if (requiresPairing && !currentCode) {
	          qrHint.textContent = tailnetEnrollmentEnabled
	            ? "Generate a pairing code first. The QR will then include an embedded tailnet enrollment payload for Codex Mobile."
	            : "Generate a pairing code first. The QR will then include a one-step bridge enrollment payload for Codex Mobile.";
	          qrFrame.classList.add("hidden");
	          qrEmpty.textContent = "Waiting for pairing code.";
	          return;
        }

        const params = new URLSearchParams();
        if (currentCode) {
          params.set("code", currentCode);
        }
        params.set("ts", String(Date.now()));
        qrImage.src = BOOTSTRAP.paths.mobileQR + "?" + params.toString();
	        qrFrame.classList.remove("hidden");
	        qrEmpty.textContent = requiresPairing
	          ? (tailnetEnrollmentEnabled
	            ? "This QR includes the current pairing code, bridge endpoint, and embedded tailnet enrollment."
	            : "This QR includes the current pairing code and bridge endpoint.")
	          : (tailnetEnrollmentEnabled
	            ? "This QR includes the bridge endpoint and embedded tailnet enrollment."
	            : "This QR includes the bridge endpoint.");
	        qrHint.textContent = "Open Codex Mobile, tap Scan QR, and point it at this code.";
	      }

      async function refreshDevices() {
        try {
          const payload = await fetchJSON(devicesPath);
          state.devices = Array.isArray(payload.devices) ? payload.devices : [];
          renderDevices();
          setFlash(devicesFlash, "", "");
        } catch (error) {
          setFlash(devicesFlash, error instanceof Error ? error.message : "Failed to load devices.", "error");
        }
      }

      async function startPairing() {
        startPairingButton.disabled = true;
        setFlash(pairingFlash, "", "");
        try {
          const payload = await fetchJSON(pairStartPath, { method: "POST" });
          state.pairing = {
            code: payload.code || "",
            expiresAt: payload.expiresAt || null,
          };
          renderPairing();
          setFlash(pairingFlash, "Pairing code generated.", "");
        } catch (error) {
          setFlash(pairingFlash, error instanceof Error ? error.message : "Failed to generate pairing code.", "error");
        } finally {
          startPairingButton.disabled = authInfo.mode !== "device-token";
        }
      }

      async function runDeviceAction(deviceId, action) {
        setFlash(devicesFlash, "", "");
        try {
          await fetchJSON(devicesPath + "/" + encodeURIComponent(deviceId) + "/" + action, { method: "POST" });
          await refreshDevices();
        } catch (error) {
          setFlash(devicesFlash, error instanceof Error ? error.message : "Device update failed.", "error");
        }
      }

      document.getElementById("refresh-status").addEventListener("click", function () {
        window.location.reload();
      });
      document.getElementById("refresh-devices").addEventListener("click", function () {
        void refreshDevices();
      });
      startPairingButton.addEventListener("click", function () {
        void startPairing();
      });
      copyPairingButton.addEventListener("click", function () {
        void copyText(state.pairing && state.pairing.code, "Pairing code copied.");
      });
      copyEndpointButton.addEventListener("click", function () {
        void copyText(connectInfo.recommendedServerEndpoint || "", "Recommended endpoint copied.");
      });
      deviceList.addEventListener("click", function (event) {
        const target = event.target;
        if (!(target instanceof HTMLButtonElement)) return;
        const action = target.dataset.action;
        const deviceId = target.dataset.id;
        if (!action || !deviceId) return;
        void runDeviceAction(deviceId, action);
      });

      renderConnectionMeta();
      renderPairing();
      renderEnrollmentQR();
      renderDevices();
      void refreshDevices();
    </script>
  </body>
</html>
`, serialized)
}
