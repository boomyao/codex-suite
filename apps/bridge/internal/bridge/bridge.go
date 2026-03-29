package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/boomyao/codex-bridge/internal/auth"
	"github.com/boomyao/codex-bridge/internal/exposure"
	"github.com/gorilla/websocket"
)

var localDirectRPCMethodNames = []string{
	"active-workspace-roots",
	"account-info",
	"app/list",
	"codex-home",
	"developer-instructions",
	"experimentalFeature/list",
	"extension-info",
	"fast-mode-rollout-metrics",
	"get-configuration",
	"get-copilot-api-proxy-info",
	"get-global-state",
	"gh-cli-status",
	"gh-pr-status",
	"git-origins",
	"has-custom-cli-executable",
	"hotkey-window-hotkey-state",
	"ide-context",
	"inbox-items",
	"is-copilot-api-available",
	"list-automations",
	"list-pending-automation-run-threads",
	"list-pinned-threads",
	"local-custom-agents",
	"local-environments",
	"locale-info",
	"mcp-codex-config",
	"open-in-targets",
	"os-info",
	"paths-exist",
	"recommended-skills",
	"remote-workspace-directory-entries",
	"set-configuration",
	"set-global-state",
	"set-pinned-threads-order",
	"set-thread-pinned",
	"thread-terminal-snapshot",
	"workspace-root-options",
	"worktree-shell-environment-config",
}

var passthroughMobileDirectRPCMethods = []string{
	"account/read",
	"account/rateLimits/read",
	"collaborationMode/list",
	"config/read",
	"mcpServerStatus/list",
	"model/list",
	"skills/list",
	"thread/list",
}

var MobileDirectRPCMethods = append(
	append([]string{}, localDirectRPCMethodNames...),
	passthroughMobileDirectRPCMethods...,
)

var LocalDirectRPCMethods = sliceToSet(append(
	append([]string{}, localDirectRPCMethodNames...),
	"get-global-state-snapshot",
))

func sliceToSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

type localDirectRPCState struct {
	mu                 sync.Mutex
	configurationState map[string]any
	pinnedThreadIDs    []string
}

type Config struct {
	Host               string
	Port               int
	UpstreamURL        string
	UpstreamHeaders    http.Header
	HealthEnabled      bool
	HealthPath         string
	ReadyPath          string
	ProbeTimeout       time.Duration
	ProbeCacheTTL      time.Duration
	DesktopWebviewRoot string
	UIPathPrefix       string
	MobileEnrollment   MobileEnrollmentConfig
}

type MobileEnrollmentConfig struct {
	ControlURL           string
	AuthKey              string
	APIAccessToken       string
	Hostname             string
	LoginMode            string
	OAuthClientID        string
	OAuthClientSecret    string
	OAuthTailnet         string
	OAuthTags            []string
	AuthKeyExpirySeconds int
}

type Info struct {
	BridgeHTTPURL      string
	BridgeWebSocketURL string
	BridgeReadyURL     string
	AppServerWebSocket string
}

type RuntimeStatus struct {
	Mode                  string `json:"mode"`
	Ready                 bool   `json:"ready"`
	AppServerWebSocketURL string `json:"appServerWebSocketUrl"`
}

type ExposureStatus struct {
	Mode    string            `json:"mode"`
	Ready   bool              `json:"ready"`
	Session *exposure.Session `json:"session,omitempty"`
}

type healthState struct {
	startedAt           time.Time
	lastProbeAt         time.Time
	lastProbeOK         *bool
	lastProbeError      string
	lastProbeDurationMS int64
	inflightProbe       chan readiness
}

type readiness struct {
	OK         bool   `json:"ok"`
	CheckedAt  int64  `json:"checkedAt"`
	DurationMS int64  `json:"durationMs"`
	Error      string `json:"error"`
}

type Bridge struct {
	config                         Config
	logger                         *log.Logger
	server                         *http.Server
	healthState                    *healthState
	upgrader                       websocket.Upgrader
	closeOnce                      sync.Once
	mu                             sync.Mutex
	localState                     *localDirectRPCState
	info                           *Info
	runtime                        RuntimeStatus
	exposure                       ExposureStatus
	auth                           auth.State
	authorizer                     auth.Authorizer
	authKeys                       *mobileAuthKeyProvider
	tailnetBootstrapStatusProvider func() map[string]any
}

func New(config Config, logger *log.Logger) *Bridge {
	if logger == nil {
		logger = log.Default()
	}
	if config.HealthPath == "" {
		config.HealthPath = "/healthz"
	}
	if config.ReadyPath == "" {
		config.ReadyPath = "/readyz"
	}
	if config.ProbeTimeout <= 0 {
		config.ProbeTimeout = 2 * time.Second
	}
	if config.ProbeCacheTTL <= 0 {
		config.ProbeCacheTTL = 5 * time.Second
	}
	if config.UIPathPrefix == "" {
		config.UIPathPrefix = "/ui"
	}
	return &Bridge{
		config: config,
		logger: logger,
		healthState: &healthState{
			startedAt: time.Now(),
		},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(*http.Request) bool { return true },
		},
		localState: &localDirectRPCState{
			configurationState: map[string]any{},
			pinnedThreadIDs:    []string{},
		},
		exposure: ExposureStatus{
			Mode:  "none",
			Ready: false,
		},
		auth: auth.State{
			Mode: "none",
		},
		authorizer: auth.NewNoopAuthorizer(),
		authKeys:   newMobileAuthKeyProvider(config.MobileEnrollment, logger),
	}
}

func (b *Bridge) Start(ctx context.Context) (*Info, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", b.handleRoot)

	address := net.JoinHostPort(b.config.Host, fmt.Sprintf("%d", b.config.Port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	b.server = &http.Server{
		Handler: mux,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = b.server.Shutdown(shutdownCtx)
	}()

	go func() {
		if err := b.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			b.logger.Printf("%s [codex-bridge] http server failed: %v", nowISO(), err)
		}
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	httpURL := fmt.Sprintf("http://%s:%d", b.config.Host, port)
	info := &Info{
		BridgeHTTPURL:      httpURL,
		BridgeWebSocketURL: fmt.Sprintf("ws://%s:%d", b.config.Host, port),
		BridgeReadyURL:     httpURL + b.config.ReadyPath,
		AppServerWebSocket: b.config.UpstreamURL,
	}
	b.mu.Lock()
	b.info = info
	b.mu.Unlock()

	b.logger.Printf("%s [codex-bridge] listening on %s", nowISO(), httpURL)
	b.logger.Printf("%s [codex-bridge] websocket proxying to %s", nowISO(), b.config.UpstreamURL)
	if b.config.HealthEnabled {
		b.logger.Printf("%s [codex-bridge] health: %s%s", nowISO(), httpURL, b.config.HealthPath)
		b.logger.Printf("%s [codex-bridge] ready: %s%s", nowISO(), httpURL, b.config.ReadyPath)
	}
	if uiRoot, ok := b.uiRoot(); ok {
		b.logger.Printf("%s [codex-bridge] ui: %s%s from %s", nowISO(), httpURL, normalizedUIPathPrefix(b.config.UIPathPrefix), uiRoot)
	}

	return info, nil
}

func (b *Bridge) SetRuntimeStatus(status RuntimeStatus) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.runtime = status
}

func (b *Bridge) SetExposureStatus(mode string, session *exposure.Session) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.exposure = ExposureStatus{
		Mode:    mode,
		Ready:   session != nil && session.Status == "ready",
		Session: session,
	}
}

func (b *Bridge) SetAuthState(state auth.State) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.auth = state
}

func (b *Bridge) SetAuthorizer(authorizer auth.Authorizer) {
	if authorizer == nil {
		authorizer = auth.NewNoopAuthorizer()
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.authorizer = authorizer
	b.auth = authorizer.State()
}

func (b *Bridge) SetMobileEnrollmentConfig(config MobileEnrollmentConfig) {
	b.mu.Lock()
	b.config.MobileEnrollment = config
	authKeys := b.authKeys
	b.mu.Unlock()

	if authKeys != nil {
		authKeys.UpdateConfig(config)
	}
}

func (b *Bridge) SetTailnetBootstrapStatusProvider(provider func() map[string]any) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.tailnetBootstrapStatusProvider = provider
}

func (b *Bridge) snapshotStatus() (RuntimeStatus, ExposureStatus, auth.State, *Info, func() map[string]any) {
	b.mu.Lock()
	defer b.mu.Unlock()

	runtimeStatus := b.runtime
	exposureStatus := b.exposure
	authState := b.auth
	statusProvider := b.tailnetBootstrapStatusProvider
	var info *Info
	if b.info != nil {
		copy := *b.info
		info = &copy
	}
	return runtimeStatus, exposureStatus, authState, info, statusProvider
}

func buildConnectionTarget(exposureStatus ExposureStatus, info *Info) map[string]any {
	gatewayEndpoint := ""
	if info != nil {
		gatewayEndpoint = sanitizeAdvertisedWebSocketURL(info.BridgeWebSocketURL)
	}

	recommendedEndpoint := gatewayEndpoint
	source := "gateway"
	exposureEndpoint := ""
	exposureHTTPURL := ""
	if exposureStatus.Session != nil {
		exposureEndpoint = sanitizeAdvertisedWebSocketURL(exposureStatus.Session.ReachableWS)
		exposureHTTPURL = sanitizeAdvertisedHTTPURL(exposureStatus.Session.ReachableHTTP)
	}
	if exposureStatus.Ready && exposureEndpoint != "" {
		recommendedEndpoint = exposureEndpoint
		source = exposureStatus.Mode
	} else if recommendedEndpoint == "" {
		source = "unavailable"
	}

	payload := map[string]any{
		"recommendedServerEndpoint": recommendedEndpoint,
		"source":                    source,
		"gatewayServerEndpoint":     gatewayEndpoint,
	}
	if exposureEndpoint != "" {
		payload["exposureServerEndpoint"] = exposureEndpoint
	}
	if exposureHTTPURL != "" {
		payload["exposureHttpUrl"] = exposureHTTPURL
	}
	return payload
}

func sanitizeAdvertisedWebSocketURL(rawURL string) string {
	parsed, ok := parseAdvertisedURL(rawURL)
	if !ok {
		return ""
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return ""
	}
	return parsed.String()
}

func sanitizeAdvertisedHTTPURL(rawURL string) string {
	parsed, ok := parseAdvertisedURL(rawURL)
	if !ok {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return parsed.String()
}

func parseAdvertisedURL(rawURL string) (*url.URL, bool) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, false
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" || isWildcardAdvertiseHost(host) {
		return nil, false
	}
	return parsed, true
}

func isWildcardAdvertiseHost(host string) bool {
	switch strings.TrimSpace(strings.ToLower(host)) {
	case "0.0.0.0", "::", "[::]":
		return true
	default:
		return false
	}
}

func (b *Bridge) handleAuthHTTP(w http.ResponseWriter, r *http.Request) bool {
	b.mu.Lock()
	authorizer := b.authorizer
	b.mu.Unlock()
	if authorizer == nil {
		return false
	}
	return authorizer.HandleHTTP(w, r)
}

func (b *Bridge) requireAuthorization(w http.ResponseWriter, r *http.Request) bool {
	b.mu.Lock()
	authorizer := b.authorizer
	b.mu.Unlock()
	if authorizer == nil {
		return true
	}
	if err := authorizer.AuthorizeRequest(r); err != nil {
		if errors.Is(err, auth.ErrUnauthorized) {
			sendJSON(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "Unauthorized"})
			return false
		}
		sendJSON(w, http.StatusInternalServerError, map[string]any{"ok": false, "error": err.Error()})
		return false
	}
	return true
}

func (b *Bridge) Close(ctx context.Context) error {
	var err error
	b.closeOnce.Do(func() {
		if b.server != nil {
			err = b.server.Shutdown(ctx)
		}
	})
	return err
}

func (b *Bridge) handleRoot(w http.ResponseWriter, r *http.Request) {
	if b.handleAuthHTTP(w, r) {
		return
	}
	if r.URL.Path == "/auth/session" {
		b.handleAuthSession(w, r)
		return
	}

	if websocket.IsWebSocketUpgrade(r) {
		if b.config.HealthEnabled && (r.URL.Path == b.config.HealthPath || r.URL.Path == b.config.ReadyPath) {
			http.NotFound(w, r)
			return
		}
		if !b.requireAuthorization(w, r) {
			return
		}
		b.handleWebSocketProxy(w, r)
		return
	}

	if r.URL.Path == "/codex-mobile/rpc" {
		if !b.requireAuthorization(w, r) {
			return
		}
		b.handleDirectRPC(w, r)
		return
	}
	if b.handleWhamStub(w, r) {
		return
	}
	if b.handleMobilePreload(w, r) {
		return
	}
	if b.handleLocalAuthPage(w, r) {
		return
	}
	if b.handleUIAsset(w, r) {
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
		return
	}

	switch r.URL.Path {
	case "/", "/status":
		readiness := b.getUpstreamReadiness(r.Context(), false)
		runtimeStatus, exposureStatus, authState, info, tailnetBootstrapStatusProvider := b.snapshotStatus()
		gatewayStatus := map[string]any{
			"host":          b.config.Host,
			"port":          b.config.Port,
			"uptimeMs":      time.Since(b.healthState.startedAt).Milliseconds(),
			"websocketPath": "/",
		}
		if info != nil {
			gatewayStatus["httpUrl"] = info.BridgeHTTPURL
			gatewayStatus["webSocketUrl"] = info.BridgeWebSocketURL
			gatewayStatus["readyUrl"] = info.BridgeReadyURL
		}
		payload := map[string]any{
			"ok":      true,
			"runtime": runtimeStatus,
			"gateway": gatewayStatus,
			"upstream": map[string]any{
				"url":        b.config.UpstreamURL,
				"ready":      readiness.OK,
				"checkedAt":  readiness.CheckedAt,
				"durationMs": readiness.DurationMS,
				"error":      emptyToNil(readiness.Error),
			},
			"exposure": exposureStatus,
			"auth":     authState,
			"healthPaths": map[string]any{
				"health": b.config.HealthPath,
				"ready":  b.config.ReadyPath,
			},
			"localAuthPage": emptyToNil(localAuthPageURL(authState)),
			"ui":            b.uiStatus(),
		}
		if tailnetBootstrapStatusProvider != nil {
			payload["tailnetBootstrap"] = tailnetBootstrapStatusProvider()
		}
		sendJSON(w, http.StatusOK, payload)
		return
	case "/codex-mobile/connect":
		_, exposureStatus, authState, info, _ := b.snapshotStatus()
		sendJSON(w, http.StatusOK, map[string]any{
			"ok":            true,
			"connection":    buildConnectionTarget(exposureStatus, info),
			"auth":          authState,
			"exposure":      exposureStatus,
			"localAuthPage": emptyToNil(localAuthPageURL(authState)),
		})
		return
	}

	if b.config.HealthEnabled && r.URL.Path == b.config.HealthPath {
		sendJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"bridge": map[string]any{
				"host":     b.config.Host,
				"port":     b.config.Port,
				"uptimeMs": time.Since(b.healthState.startedAt).Milliseconds(),
			},
		})
		return
	}

	if b.config.HealthEnabled && r.URL.Path == b.config.ReadyPath {
		readiness := b.getUpstreamReadiness(r.Context(), true)
		status := http.StatusServiceUnavailable
		if readiness.OK {
			status = http.StatusOK
		}
		runtimeStatus, exposureStatus, authState, info, tailnetBootstrapStatusProvider := b.snapshotStatus()
		gatewayStatus := map[string]any{
			"host":     b.config.Host,
			"port":     b.config.Port,
			"uptimeMs": time.Since(b.healthState.startedAt).Milliseconds(),
		}
		if info != nil {
			gatewayStatus["httpUrl"] = info.BridgeHTTPURL
			gatewayStatus["webSocketUrl"] = info.BridgeWebSocketURL
			gatewayStatus["readyUrl"] = info.BridgeReadyURL
		}
		payload := map[string]any{
			"ok":      readiness.OK,
			"runtime": runtimeStatus,
			"gateway": gatewayStatus,
			"upstream": map[string]any{
				"url":        b.config.UpstreamURL,
				"ready":      readiness.OK,
				"checkedAt":  readiness.CheckedAt,
				"durationMs": readiness.DurationMS,
				"error":      emptyToNil(readiness.Error),
			},
			"exposure": exposureStatus,
			"auth":     authState,
		}
		if tailnetBootstrapStatusProvider != nil {
			payload["tailnetBootstrap"] = tailnetBootstrapStatusProvider()
		}
		sendJSON(w, status, payload)
		return
	}

	sendText(w, http.StatusNotFound, "Not Found\n")
}

func (b *Bridge) handleAuthSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
		return
	}

	_, _, authState, _, _ := b.snapshotStatus()

	b.mu.Lock()
	authorizer := b.authorizer
	b.mu.Unlock()
	if authorizer == nil {
		authorizer = auth.NewNoopAuthorizer()
	}
	session := authorizer.DescribeRequest(r)

	if !session.Authorized {
		statusCode := http.StatusInternalServerError
		errorMessage := "Unauthorized"
		if session.Reason != "" {
			errorMessage = session.Reason
		}
		if session.Reason == "missing_token" || session.Reason == "unknown_token" || session.Reason == "pending_approval" || session.Reason == "revoked" {
			statusCode = http.StatusUnauthorized
		}
		sendJSON(w, statusCode, map[string]any{
			"ok":            false,
			"authorized":    session.Authorized,
			"reason":        emptyToNil(session.Reason),
			"deviceId":      emptyToNil(session.DeviceID),
			"deviceName":    emptyToNil(session.DeviceName),
			"auth":          authState,
			"localAuthPage": emptyToNil(localAuthPageURL(authState)),
			"error":         errorMessage,
		})
		return
	}

	sendJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"authorized":    session.Authorized,
		"reason":        emptyToNil(session.Reason),
		"deviceId":      emptyToNil(session.DeviceID),
		"deviceName":    emptyToNil(session.DeviceName),
		"auth":          authState,
		"localAuthPage": emptyToNil(localAuthPageURL(authState)),
	})
}

func (b *Bridge) handleDirectRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
		return
	}

	var payload struct {
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Invalid JSON body."})
		return
	}
	payload.Method = strings.TrimSpace(payload.Method)
	if payload.Method == "" {
		sendJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Direct RPC method is required."})
		return
	}

	startedAt := time.Now()
	if _, ok := LocalDirectRPCMethods[payload.Method]; ok {
		result := b.performLocalDirectRPC(payload.Method, payload.Params)
		b.logger.Printf("%s [codex-bridge] direct rpc local %s %dms", nowISO(), payload.Method, time.Since(startedAt).Milliseconds())
		sendJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
		return
	}

	result, err := performUpstreamRPC(r.Context(), b.config, payload.Method, payload.Params)
	if err != nil {
		b.logger.Printf("%s [codex-bridge] direct rpc failed %s %v", nowISO(), payload.Method, err)
		sendJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	b.logger.Printf("%s [codex-bridge] direct rpc %s %dms", nowISO(), payload.Method, time.Since(startedAt).Milliseconds())
	sendJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

func (b *Bridge) performLocalDirectRPC(method string, params map[string]any) any {
	switch method {
	case "get-global-state":
		key, _ := stringParam(params, "key")
		state, err := b.loadGlobalState()
		if err != nil {
			b.logger.Printf("%s [codex-bridge] failed to load global state: %v", nowISO(), err)
			return map[string]any{"value": nil}
		}
		return map[string]any{"value": state[key]}
	case "get-global-state-snapshot":
		state, err := b.loadGlobalState()
		if err != nil {
			b.logger.Printf("%s [codex-bridge] failed to load global state snapshot: %v", nowISO(), err)
			return map[string]any{
				"state":                map[string]any{},
				"workspaceRootOptions": map[string]any{"roots": []string{}, "activeRoots": []string{}, "labels": map[string]any{}},
			}
		}
		roots, activeRoots, labels := b.workspaceRootOptions()
		return map[string]any{
			"state": state,
			"workspaceRootOptions": map[string]any{
				"roots":       roots,
				"activeRoots": activeRoots,
				"labels":      labels,
			},
		}
	case "set-global-state":
		key, _ := stringParam(params, "key")
		if key == "" {
			return map[string]any{"success": false}
		}
		b.localState.mu.Lock()
		defer b.localState.mu.Unlock()
		state, err := b.loadGlobalStateLocked()
		if err != nil {
			b.logger.Printf("%s [codex-bridge] failed to load global state for update: %v", nowISO(), err)
			return map[string]any{"success": false}
		}
		if value, ok := params["value"]; ok {
			state[key] = value
		} else {
			delete(state, key)
		}
		if err := saveGlobalStateFile(globalStateFilePath(), state); err != nil {
			b.logger.Printf("%s [codex-bridge] failed to save global state: %v", nowISO(), err)
			return map[string]any{"success": false}
		}
		return map[string]any{"success": true}
	case "get-configuration":
		key, _ := stringParam(params, "key")
		b.localState.mu.Lock()
		value, ok := b.localState.configurationState[key]
		b.localState.mu.Unlock()
		if !ok {
			value = nil
		}
		return map[string]any{"value": value}
	case "set-configuration":
		key, _ := stringParam(params, "key")
		if key == "" {
			return map[string]any{"success": false}
		}
		b.localState.mu.Lock()
		if value, ok := params["value"]; ok {
			b.localState.configurationState[key] = value
		} else {
			delete(b.localState.configurationState, key)
		}
		b.localState.mu.Unlock()
		return map[string]any{"success": true}
	case "list-pinned-threads":
		b.localState.mu.Lock()
		threadIDs := append([]string(nil), b.localState.pinnedThreadIDs...)
		b.localState.mu.Unlock()
		return map[string]any{"threadIds": threadIDs}
	case "set-thread-pinned":
		threadID, _ := stringParam(params, "threadId")
		if threadID == "" {
			return map[string]any{"success": false, "threadIds": []string{}}
		}
		pinned, _ := boolParam(params, "pinned")
		b.localState.mu.Lock()
		nextThreadIDs := append([]string(nil), b.localState.pinnedThreadIDs...)
		if pinned {
			nextThreadIDs = appendUniqueStrings(nextThreadIDs, threadID)
		} else {
			nextThreadIDs = filterStringValue(nextThreadIDs, threadID)
		}
		b.localState.pinnedThreadIDs = nextThreadIDs
		b.localState.mu.Unlock()
		return map[string]any{"success": true, "threadIds": nextThreadIDs}
	case "set-pinned-threads-order":
		threadIDs := uniqueStringSlice(anySliceParam(params, "threadIds"))
		b.localState.mu.Lock()
		b.localState.pinnedThreadIDs = append([]string(nil), threadIDs...)
		b.localState.mu.Unlock()
		return map[string]any{"success": true, "threadIds": threadIDs}
	case "extension-info":
		return map[string]any{
			"version":     "26.323.20928",
			"buildFlavor": "prod",
			"buildNumber": "1173",
		}
	case "is-copilot-api-available":
		return map[string]any{"available": false}
	case "account-info":
		return map[string]any{"plan": nil, "accountId": nil}
	case "app/list":
		return map[string]any{"data": []any{}, "nextCursor": nil}
	case "list-pending-automation-run-threads":
		return map[string]any{"threadIds": []string{}}
	case "inbox-items":
		return map[string]any{"items": []any{}}
	case "list-automations":
		return map[string]any{"items": []any{}}
	case "local-environments":
		return map[string]any{"environments": []any{}}
	case "open-in-targets":
		return map[string]any{"targets": []any{}}
	case "gh-cli-status":
		return map[string]any{"isInstalled": false, "isAuthenticated": false}
	case "gh-pr-status":
		return map[string]any{
			"status":    "success",
			"hasOpenPr": false,
			"isDraft":   false,
			"url":       nil,
			"canMerge":  false,
			"ciStatus":  nil,
		}
	case "recommended-skills":
		return map[string]any{"skills": []any{}, "error": nil}
	case "ide-context":
		return map[string]any{"status": "disconnected", "connected": false, "context": nil}
	case "local-custom-agents":
		return map[string]any{"agents": []any{}}
	case "hotkey-window-hotkey-state":
		return map[string]any{"supported": false, "configuredHotkey": nil, "state": nil}
	case "fast-mode-rollout-metrics":
		return map[string]any{"enabled": true, "estimatedSavedMs": 0, "rolloutCountWithCompletedTurns": 0}
	case "experimentalFeature/list":
		return map[string]any{"data": []any{}, "nextCursor": nil}
	case "os-info":
		return map[string]any{
			"platform":  runtime.GOOS,
			"isMacOS":   runtime.GOOS == "darwin",
			"isWindows": runtime.GOOS == "windows",
			"isLinux":   runtime.GOOS == "linux",
		}
	case "locale-info":
		locale := localeFromEnvironment()
		return map[string]any{"ideLocale": locale, "systemLocale": locale}
	case "active-workspace-roots":
		return map[string]any{"roots": b.activeWorkspaceRoots()}
	case "workspace-root-options":
		roots, activeRoots, labels := b.workspaceRootOptions()
		return map[string]any{"roots": roots, "activeRoots": activeRoots, "labels": labels}
	case "has-custom-cli-executable":
		return map[string]any{"hasCustomCliExecutable": false}
	case "get-copilot-api-proxy-info":
		return nil
	case "codex-home":
		return map[string]any{"codexHome": deriveCodexHome(), "config": nil, "layers": []any{}, "origins": nil}
	case "git-origins":
		return map[string]any{"origins": []any{}}
	case "paths-exist":
		return map[string]any{"existingPaths": existingPaths(anySliceParam(params, "paths"))}
	case "mcp-codex-config":
		return map[string]any{"config": nil}
	case "worktree-shell-environment-config":
		return map[string]any{"shellEnvironment": nil}
	case "developer-instructions":
		value, _ := stringParam(params, "baseInstructions")
		return map[string]any{"instructions": nullableString(value)}
	case "thread-terminal-snapshot":
		return map[string]any{
			"session": map[string]any{
				"cwd":       "",
				"shell":     "unknown",
				"buffer":    "",
				"truncated": false,
			},
		}
	case "remote-workspace-directory-entries":
		directoryPath, _ := stringParam(params, "directoryPath")
		return map[string]any{"directoryPath": directoryPath, "entries": []any{}}
	default:
		return nil
	}
}

func stringParam(params map[string]any, key string) (string, bool) {
	value, ok := params[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return text, ok
}

func boolParam(params map[string]any, key string) (bool, bool) {
	value, ok := params[key]
	if !ok {
		return false, false
	}
	flag, ok := value.(bool)
	return flag, ok
}

func anySliceParam(params map[string]any, key string) []any {
	value, ok := params[key]
	if !ok {
		return nil
	}
	items, _ := value.([]any)
	return items
}

func appendUniqueStrings(values []string, next string) []string {
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}

func filterStringValue(values []string, target string) []string {
	next := make([]string, 0, len(values))
	for _, value := range values {
		if value != target {
			next = append(next, value)
		}
	}
	return next
}

func uniqueStringSlice(values []any) []string {
	seen := make(map[string]struct{}, len(values))
	next := make([]string, 0, len(values))
	for _, value := range values {
		text, ok := value.(string)
		if !ok {
			continue
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if _, exists := seen[text]; exists {
			continue
		}
		seen[text] = struct{}{}
		next = append(next, text)
	}
	return next
}

func deriveCodexHome() string {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		return "~/.codex"
	}
	return filepath.Join(homeDir, ".codex")
}

func globalStateFilePath() string {
	return filepath.Join(deriveCodexHome(), ".codex-global-state.json")
}

func (b *Bridge) loadGlobalState() (map[string]any, error) {
	b.localState.mu.Lock()
	defer b.localState.mu.Unlock()
	return b.loadGlobalStateLocked()
}

func (b *Bridge) loadGlobalStateLocked() (map[string]any, error) {
	return loadGlobalStateFile(globalStateFilePath())
}

func loadGlobalStateFile(path string) (map[string]any, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace(contents)) == 0 {
		return map[string]any{}, nil
	}
	state := make(map[string]any)
	if err := json.Unmarshal(contents, &state); err != nil {
		return nil, err
	}
	return state, nil
}

func saveGlobalStateFile(path string, state map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	contents = append(contents, '\n')
	tempPath := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, contents, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

func (b *Bridge) activeWorkspaceRoots() []string {
	state, err := b.loadGlobalState()
	if err != nil {
		b.logger.Printf("%s [codex-bridge] failed to load active workspace roots: %v", nowISO(), err)
		return []string{}
	}
	return uniqueNonEmptyStrings(anyToStringSlice(state["active-workspace-roots"]))
}

func (b *Bridge) workspaceRootOptions() ([]string, []string, map[string]any) {
	state, err := b.loadGlobalState()
	if err != nil {
		b.logger.Printf("%s [codex-bridge] failed to load workspace root options: %v", nowISO(), err)
		return []string{}, []string{}, map[string]any{}
	}
	activeRoots := uniqueNonEmptyStrings(anyToStringSlice(state["active-workspace-roots"]))
	roots := uniqueNonEmptyStrings(append(
		append(anyToStringSlice(state["project-order"]), anyToStringSlice(state["electron-saved-workspace-roots"])...),
		activeRoots...,
	))
	if len(roots) == 0 {
		roots = activeRoots
	}
	labels := map[string]any{}
	if value, ok := state["workspace-root-labels"].(map[string]any); ok {
		for key, label := range value {
			text, ok := label.(string)
			if !ok || strings.TrimSpace(text) == "" {
				continue
			}
			labels[key] = text
		}
	}
	return roots, activeRoots, labels
}

func localeFromEnvironment() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		value = strings.SplitN(value, ".", 2)[0]
		value = strings.ReplaceAll(value, "_", "-")
		if value != "" && value != "C" && value != "POSIX" {
			return value
		}
	}
	return "en-US"
}

func existingPaths(values []any) []string {
	seen := map[string]struct{}{}
	next := make([]string, 0, len(values))
	for _, value := range values {
		path, ok := value.(string)
		if !ok {
			continue
		}
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		next = append(next, path)
	}
	return next
}

func anyToStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		next := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				continue
			}
			next = append(next, text)
		}
		return next
	default:
		return nil
	}
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	next := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		next = append(next, value)
	}
	return next
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (b *Bridge) handleWebSocketProxy(w http.ResponseWriter, r *http.Request) {
	downstream, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	upstreamConn, _, err := websocket.DefaultDialer.DialContext(r.Context(), b.config.UpstreamURL, b.config.UpstreamHeaders)
	if err != nil {
		_ = downstream.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1011, err.Error()), time.Now().Add(time.Second))
		_ = downstream.Close()
		return
	}

	clientID := fmt.Sprintf("%d", time.Now().UnixNano())
	remoteAddress := r.RemoteAddr
	b.logger.Printf("%s [codex-bridge] client connected %s %s %s", nowISO(), clientID, remoteAddress, r.URL.Path)
	b.logger.Printf("%s [codex-bridge] upstream connected %s %s", nowISO(), clientID, b.config.UpstreamURL)

	var once sync.Once
	closeBoth := func(message string) {
		once.Do(func() {
			_ = downstream.Close()
			_ = upstreamConn.Close()
			b.logger.Printf("%s [codex-bridge] client disconnected %s %s %s", nowISO(), clientID, remoteAddress, message)
		})
	}

	go proxyWebSocket(upstreamConn, downstream, func(err error) {
		if err != nil {
			b.logger.Printf("%s [codex-bridge] failed to forward upstream message %s %v", nowISO(), clientID, err)
		}
		closeBoth("Upstream closed")
	})

	go proxyWebSocket(downstream, upstreamConn, func(err error) {
		if err != nil {
			b.logger.Printf("%s [codex-bridge] failed to forward downstream message %s %v", nowISO(), clientID, err)
		}
		closeBoth("Downstream closed")
	})
}

func (b *Bridge) getUpstreamReadiness(ctx context.Context, force bool) readiness {
	b.mu.Lock()
	age := time.Since(b.healthState.lastProbeAt)
	if !force && b.healthState.lastProbeOK != nil && age < b.config.ProbeCacheTTL {
		result := readiness{
			OK:         *b.healthState.lastProbeOK,
			CheckedAt:  b.healthState.lastProbeAt.UnixMilli(),
			DurationMS: b.healthState.lastProbeDurationMS,
			Error:      b.healthState.lastProbeError,
		}
		b.mu.Unlock()
		return result
	}

	if b.healthState.inflightProbe != nil {
		probeCh := b.healthState.inflightProbe
		b.mu.Unlock()
		select {
		case result := <-probeCh:
			return result
		case <-ctx.Done():
			return readiness{OK: false, CheckedAt: time.Now().UnixMilli(), Error: ctx.Err().Error()}
		}
	}

	probeCh := make(chan readiness, 1)
	b.healthState.inflightProbe = probeCh
	b.mu.Unlock()

	result := probeUpstream(ctx, b.config)

	b.mu.Lock()
	b.healthState.lastProbeAt = time.UnixMilli(result.CheckedAt)
	b.healthState.lastProbeOK = &result.OK
	b.healthState.lastProbeError = result.Error
	b.healthState.lastProbeDurationMS = result.DurationMS
	b.healthState.inflightProbe = nil
	b.mu.Unlock()

	probeCh <- result
	close(probeCh)
	return result
}

func performUpstreamRPC(ctx context.Context, cfg Config, method string, params map[string]any) (any, error) {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, cfg.UpstreamURL, cfg.UpstreamHeaders)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"clientInfo": map[string]any{
				"name":    "codex_bridge_rpc",
				"title":   "Codex Bridge RPC",
				"version": "0.1.0",
			},
			"capabilities": map[string]any{
				"experimentalApi": true,
			},
		},
	}); err != nil {
		return nil, err
	}

	nextState := "initialize"
	for {
		var payload map[string]any
		if err := conn.ReadJSON(&payload); err != nil {
			return nil, err
		}

		if nextState == "initialize" && numericID(payload["id"]) == 1 {
			if err := conn.WriteJSON(map[string]any{
				"jsonrpc": "2.0",
				"method":  "initialized",
				"params":  map[string]any{},
			}); err != nil {
				return nil, err
			}
			if err := conn.WriteJSON(map[string]any{
				"jsonrpc": "2.0",
				"id":      2,
				"method":  method,
				"params":  params,
			}); err != nil {
				return nil, err
			}
			nextState = "request"
			continue
		}

		if nextState == "request" && numericID(payload["id"]) == 2 {
			if errPayload, ok := payload["error"].(map[string]any); ok {
				if message, ok := errPayload["message"].(string); ok && message != "" {
					return nil, errors.New(message)
				}
				return nil, fmt.Errorf("upstream RPC failed")
			}
			return payload["result"], nil
		}
	}
}

func probeUpstream(ctx context.Context, cfg Config) readiness {
	startedAt := time.Now()
	dialer := websocket.Dialer{HandshakeTimeout: cfg.ProbeTimeout}
	conn, _, err := dialer.DialContext(ctx, cfg.UpstreamURL, cfg.UpstreamHeaders)
	if err != nil {
		return readiness{
			OK:         false,
			CheckedAt:  time.Now().UnixMilli(),
			DurationMS: time.Since(startedAt).Milliseconds(),
			Error:      err.Error(),
		}
	}
	_ = conn.Close()
	return readiness{
		OK:         true,
		CheckedAt:  time.Now().UnixMilli(),
		DurationMS: time.Since(startedAt).Milliseconds(),
	}
}

func proxyWebSocket(src *websocket.Conn, dst *websocket.Conn, onDone func(error)) {
	for {
		messageType, payload, err := src.ReadMessage()
		if err != nil {
			onDone(err)
			return
		}
		if err := dst.WriteMessage(messageType, payload); err != nil {
			onDone(err)
			return
		}
	}
}

func sendJSON(w http.ResponseWriter, statusCode int, payload any) {
	body, _ := json.MarshalIndent(payload, "", "  ")
	body = append(body, '\n')
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}

func sendText(w http.ResponseWriter, statusCode int, text string) {
	body := []byte(text)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}

func numericID(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	default:
		return 0
	}
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (b *Bridge) PerformInitializeProbe(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, b.config.UpstreamURL, b.config.UpstreamHeaders)
	if err != nil {
		return err
	}
	defer conn.Close()

	request := map[string]any{
		"id":     1,
		"method": "initialize",
		"params": map[string]any{
			"clientInfo": map[string]any{
				"name":    "codex_bridge_probe",
				"title":   "Codex Bridge Probe",
				"version": "0.1.0",
			},
		},
	}

	if err := conn.WriteJSON(request); err != nil {
		return err
	}

	_ = conn.SetReadDeadline(time.Now().Add(b.config.ProbeTimeout))
	messageType, body, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
		return fmt.Errorf("unexpected initialize response type %d", messageType)
	}

	var payload map[string]any
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&payload); err != nil {
		return err
	}
	if numericID(payload["id"]) != 1 {
		return fmt.Errorf("unexpected initialize response payload")
	}
	return nil
}
