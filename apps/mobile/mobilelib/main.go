package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unsafe"

	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tsnet"
)

type enrollmentPayload struct {
	Type                 string `json:"type"`
	Version              int    `json:"version"`
	BridgeID             string `json:"bridgeId,omitempty"`
	BridgeName           string `json:"bridgeName,omitempty"`
	BridgeServerEndpoint string `json:"bridgeServerEndpoint,omitempty"`
	PairingCode          string `json:"pairingCode,omitempty"`
	Tailnet              struct {
		ControlURL string `json:"controlUrl,omitempty"`
		AuthKey    string `json:"authKey,omitempty"`
		Hostname   string `json:"hostname,omitempty"`
		LoginMode  string `json:"loginMode,omitempty"`
	} `json:"tailnet"`
}

type authStatus struct {
	Backend          string   `json:"backend"`
	BackendState     string   `json:"backend_state"`
	LoggedIn         bool     `json:"logged_in"`
	NeedsLogin       bool     `json:"needs_login,omitempty"`
	NeedsMachineAuth bool     `json:"needs_machine_auth,omitempty"`
	AuthURL          string   `json:"auth_url,omitempty"`
	Tailnet          string   `json:"tailnet,omitempty"`
	SelfDNSName      string   `json:"self_dns_name,omitempty"`
	MagicDNSSuffix   string   `json:"magic_dns_suffix,omitempty"`
	MagicDNSEnabled  bool     `json:"magic_dns_enabled,omitempty"`
	TailscaleIPs     []string `json:"tailscale_ips,omitempty"`
	Health           []string `json:"health,omitempty"`
}

type bridgeResponse struct {
	OK      bool   `json:"ok"`
	Running bool   `json:"running,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type runtimeState struct {
	Running    bool               `json:"running"`
	Starting   bool               `json:"starting"`
	Message    string             `json:"message,omitempty"`
	LastError  string             `json:"last_error,omitempty"`
	StateDir   string             `json:"state_dir,omitempty"`
	LocalProxy string             `json:"local_proxy_url,omitempty"`
	Payload    *enrollmentPayload `json:"payload,omitempty"`
	Auth       *authStatus        `json:"auth,omitempty"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

type runtimeSession struct {
	server   *tsnet.Server
	cancel   context.CancelFunc
	done     chan struct{}
	stateDir string
	payload  *enrollmentPayload
	localMu  sync.Mutex
	local    *local.Client
	proxy    *localProxy
}

type localProxy struct {
	baseURL  string
	listener net.Listener
	server   *http.Server
	dial     func(context.Context, string, string) (net.Conn, error)

	mu           sync.RWMutex
	targetBase   *url.URL
	authToken    string
	lastEndpoint string
}

var (
	stateMu       sync.Mutex
	currentState  = runtimeState{Message: "embedded tailnet runtime is idle"}
	activeSession *runtimeSession
)

func main() {}

//export CodexMobileVersionJSON
func CodexMobileVersionJSON() *C.char {
	return jsonCString(bridgeResponse{
		OK:      true,
		Message: "codexmobile android bridge",
		Data: map[string]any{
			"runtime": "tsnet",
			"version": 2,
		},
	})
}

//export CodexMobileStatusJSON
func CodexMobileStatusJSON() *C.char {
	maybeRefreshRuntimeStatus()
	state := snapshotRuntimeState()
	return jsonCString(responseFromRuntimeState(state))
}

//export CodexMobileStartWithTunFDJSON
func CodexMobileStartWithTunFDJSON(payloadJSON *C.char, stateDirC *C.char, tunFD C.int) *C.char {
	payload, err := parsePayload(payloadJSON)
	if err != nil {
		closeTunFD(int(tunFD))
		return jsonCString(bridgeResponse{Error: err.Error()})
	}
	closeTunFD(int(tunFD))

	stateDir := strings.TrimSpace(goString(stateDirC))
	if stateDir == "" {
		stateDir = filepath.Join(os.TempDir(), "codexmobile-tailnet")
	}
	stateDir = filepath.Clean(stateDir)
	if err := os.MkdirAll(filepath.Join(stateDir, "tailscale"), 0o755); err != nil {
		return jsonCString(bridgeResponse{Error: err.Error()})
	}

	stateMu.Lock()
	if activeSession != nil {
		state := currentState
		stateMu.Unlock()
		return jsonCString(responseFromRuntimeState(state))
	}

	ctx, cancel := context.WithCancel(context.Background())
	session := &runtimeSession{
		cancel:   cancel,
		done:     make(chan struct{}),
		stateDir: stateDir,
		payload:  payload,
	}
	activeSession = session
	currentState = runtimeState{
		Running:   true,
		Starting:  true,
		Message:   "starting embedded tsnet runtime",
		StateDir:  stateDir,
		Payload:   payload,
		UpdatedAt: time.Now(),
	}
	state := currentState
	stateMu.Unlock()

	go runRuntimeSession(ctx, session)
	return jsonCString(responseFromRuntimeState(state))
}

//export CodexMobileStopJSON
func CodexMobileStopJSON() *C.char {
	stateMu.Lock()
	session := activeSession
	if session == nil {
		currentState = runtimeState{
			Message:   "embedded tailnet runtime is already stopped",
			UpdatedAt: time.Now(),
		}
		state := currentState
		stateMu.Unlock()
		return jsonCString(responseFromRuntimeState(state))
	}
	state := currentState
	stateMu.Unlock()

	session.cancel()
	select {
	case <-session.done:
	case <-time.After(3 * time.Second):
	}

	stateMu.Lock()
	if activeSession == session {
		activeSession = nil
	}
	currentState.Running = false
	currentState.Starting = false
	currentState.Message = "embedded tailnet runtime stopped"
	currentState.UpdatedAt = time.Now()
	state = currentState
	stateMu.Unlock()
	return jsonCString(responseFromRuntimeState(state))
}

//export CodexMobileConfigureBridgeProxyJSON
func CodexMobileConfigureBridgeProxyJSON(endpointC *C.char, authTokenC *C.char) *C.char {
	stateMu.Lock()
	session := activeSession
	stateMu.Unlock()
	if session == nil || session.proxy == nil {
		return jsonCString(bridgeResponse{
			OK:      false,
			Running: false,
			Error:   "embedded tailnet runtime is not running",
		})
	}

	if err := session.configureBridgeProxy(strings.TrimSpace(goString(endpointC)), strings.TrimSpace(goString(authTokenC))); err != nil {
		state := snapshotRuntimeState()
		return jsonCString(bridgeResponse{
			OK:      false,
			Running: state.Running,
			Message: state.Message,
			Error:   err.Error(),
			Data: map[string]any{
				"bridgeId":             bridgeIDFromPayload(state.Payload),
				"bridgeName":           bridgeNameFromPayload(state.Payload),
				"bridgeServerEndpoint": bridgeEndpointFromPayload(state.Payload),
				"localProxyUrl":        state.LocalProxy,
				"stateDir":             state.StateDir,
				"auth":                 state.Auth,
			},
		})
	}

	state := snapshotRuntimeState()
	return jsonCString(responseFromRuntimeState(state))
}

//export CodexMobileFreeString
func CodexMobileFreeString(ptr *C.char) {
	C.free(unsafe.Pointer(ptr))
}

func runRuntimeSession(ctx context.Context, session *runtimeSession) {
	defer close(session.done)

	logsDir := filepath.Join(session.stateDir, "logs")
	tempDir := filepath.Join(session.stateDir, "tmp")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		log.Printf("codexmobile: failed to create logs dir %s: %v", logsDir, err)
	}
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		log.Printf("codexmobile: failed to create temp dir %s: %v", tempDir, err)
	}
	_ = os.Setenv("TS_LOGS_DIR", logsDir)
	_ = os.Setenv("TMPDIR", tempDir)
	_ = os.Setenv("TMP", tempDir)
	_ = os.Setenv("TEMP", tempDir)

	server := &tsnet.Server{
		Dir:        filepath.Join(session.stateDir, "tailscale"),
		Hostname:   session.payload.Tailnet.Hostname,
		ControlURL: session.payload.Tailnet.ControlURL,
		AuthKey:    session.payload.Tailnet.AuthKey,
	}
	if strings.TrimSpace(session.payload.Tailnet.Hostname) == "" {
		server.Hostname = defaultHostname(session.payload)
	}

	session.server = server
	status, err := server.Up(ctx)
	if err != nil {
		updateRuntimeState(func(state *runtimeState) {
			state.Running = false
			state.Starting = false
			state.Message = "embedded tsnet runtime failed to start"
			state.LastError = err.Error()
			state.StateDir = session.stateDir
			state.Payload = session.payload
			state.Auth = nil
		})
		_ = server.Close()
		stateMu.Lock()
		if activeSession == session {
			activeSession = nil
		}
		stateMu.Unlock()
		return
	}

	localClient, err := server.LocalClient()
	if err != nil {
		updateRuntimeState(func(state *runtimeState) {
			state.Running = false
			state.Starting = false
			state.Message = "embedded tsnet runtime failed to create local client"
			state.LastError = err.Error()
			state.StateDir = session.stateDir
			state.Payload = session.payload
		})
		_ = server.Close()
		stateMu.Lock()
		if activeSession == session {
			activeSession = nil
		}
		stateMu.Unlock()
		return
	}

	session.localMu.Lock()
	session.local = localClient
	session.localMu.Unlock()

	proxy, err := startLocalProxy(session)
	if err != nil {
		updateRuntimeState(func(state *runtimeState) {
			state.Running = false
			state.Starting = false
			state.Message = "embedded tsnet runtime failed to start local proxy"
			state.LastError = err.Error()
			state.StateDir = session.stateDir
			state.Payload = session.payload
			state.Auth = authStatusFromIPN(status)
			state.LocalProxy = ""
		})
		_ = server.Close()
		stateMu.Lock()
		if activeSession == session {
			activeSession = nil
		}
		stateMu.Unlock()
		return
	}
	session.proxy = proxy
	if err := session.configureBridgeProxy(session.payload.BridgeServerEndpoint, ""); err != nil {
		log.Printf("codexmobile: configure local proxy: %v", err)
	}

	updateRuntimeState(func(state *runtimeState) {
		state.Running = true
		state.Starting = false
		state.Message = "embedded tsnet runtime connected"
		state.LastError = ""
		state.StateDir = session.stateDir
		state.LocalProxy = proxy.baseURL
		state.Payload = session.payload
		state.Auth = authStatusFromIPN(status)
	})

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if session.proxy != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = session.proxy.close(shutdownCtx)
				cancel()
			}
			_ = server.Close()
			stateMu.Lock()
			if activeSession == session {
				activeSession = nil
			}
			currentState.Running = false
			currentState.Starting = false
			currentState.Message = "embedded tailnet runtime stopped"
			currentState.LocalProxy = ""
			currentState.UpdatedAt = time.Now()
			stateMu.Unlock()
			return
		case <-ticker.C:
			refreshSessionStatus(ctx, session)
		}
	}
}

func maybeRefreshRuntimeStatus() {
	stateMu.Lock()
	session := activeSession
	stateMu.Unlock()
	if session == nil {
		return
	}
	refreshSessionStatus(context.Background(), session)
}

func refreshSessionStatus(ctx context.Context, session *runtimeSession) {
	session.localMu.Lock()
	localClient := session.local
	session.localMu.Unlock()
	if localClient == nil {
		return
	}

	statusCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	status, err := localClient.StatusWithoutPeers(statusCtx)
	if err != nil {
		updateRuntimeState(func(state *runtimeState) {
			state.Running = true
			state.Starting = false
			state.Message = "embedded tsnet runtime is running"
			state.LastError = err.Error()
			state.StateDir = session.stateDir
			state.LocalProxy = proxyURLFromSession(session)
			state.Payload = session.payload
		})
		return
	}

	updateRuntimeState(func(state *runtimeState) {
		state.Running = true
		state.Starting = false
		state.Message = "embedded tsnet runtime is running"
		state.LastError = ""
		state.StateDir = session.stateDir
		state.LocalProxy = proxyURLFromSession(session)
		state.Payload = session.payload
		state.Auth = authStatusFromIPN(status)
	})
}

func authStatusFromIPN(status *ipnstate.Status) *authStatus {
	if status == nil {
		return nil
	}
	selfDNSName := strings.TrimSuffix(strings.TrimSpace(status.Self.DNSName), ".")
	out := &authStatus{
		Backend:          "embedded",
		BackendState:     status.BackendState,
		LoggedIn:         status.BackendState == "Running",
		NeedsLogin:       status.BackendState == "NeedsLogin",
		NeedsMachineAuth: status.BackendState == "NeedsMachineAuth",
		AuthURL:          status.AuthURL,
		SelfDNSName:      selfDNSName,
		MagicDNSSuffix:   magicDNSSuffixFromSelfDNS(selfDNSName),
		MagicDNSEnabled:  selfDNSName != "",
		TailscaleIPs:     netipStrings(status.TailscaleIPs),
		Health:           slices.Clone(status.Health),
	}
	if status.CurrentTailnet != nil {
		out.Tailnet = strings.TrimSuffix(strings.TrimSpace(status.CurrentTailnet.Name), ".")
	}
	return out
}

func magicDNSSuffixFromSelfDNS(selfDNSName string) string {
	if selfDNSName == "" {
		return ""
	}
	index := strings.IndexByte(selfDNSName, '.')
	if index < 0 || index+1 >= len(selfDNSName) {
		return ""
	}
	return selfDNSName[index+1:]
}

func netipStrings(values []netip.Addr) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !value.IsValid() {
			continue
		}
		text := strings.TrimSpace(value.String())
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func defaultHostname(payload *enrollmentPayload) string {
	if payload == nil {
		return "codex-mobile-android"
	}
	if id := strings.TrimSpace(payload.BridgeID); id != "" {
		return "codex-mobile-" + sanitizeLabel(id)
	}
	return "codex-mobile-android"
}

func sanitizeLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	builder := strings.Builder{}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			builder.WriteRune(r)
		}
	}
	label := strings.Trim(builder.String(), "-")
	if label == "" {
		return "android"
	}
	if len(label) > 32 {
		return label[:32]
	}
	return label
}

func responseFromRuntimeState(state runtimeState) bridgeResponse {
	data := map[string]any{
		"bridgeId":             "",
		"bridgeName":           "",
		"bridgeServerEndpoint": "",
		"localProxyUrl":        strings.TrimSpace(state.LocalProxy),
		"stateDir":             state.StateDir,
		"auth":                 state.Auth,
	}
	if state.Payload != nil {
		data["bridgeId"] = state.Payload.BridgeID
		data["bridgeName"] = state.Payload.BridgeName
		data["bridgeServerEndpoint"] = state.Payload.BridgeServerEndpoint
		data["controlUrl"] = state.Payload.Tailnet.ControlURL
		data["hostname"] = defaultHostname(state.Payload)
	}
	response := bridgeResponse{
		OK:      state.LastError == "",
		Running: state.Running,
		Message: strings.TrimSpace(state.Message),
		Data:    data,
	}
	if state.LastError != "" {
		response.Error = state.LastError
	}
	return response
}

func (s *runtimeSession) configureBridgeProxy(endpoint, authToken string) error {
	if s.proxy == nil {
		return errors.New("local proxy is not available")
	}
	if strings.TrimSpace(endpoint) == "" && s.payload != nil {
		endpoint = s.payload.BridgeServerEndpoint
	}
	targetBase, err := normalizeBridgeHTTPBaseURL(endpoint)
	if err != nil {
		return err
	}
	s.proxy.configure(targetBase, authToken)
	updateRuntimeState(func(state *runtimeState) {
		state.LocalProxy = s.proxy.baseURL
		state.Message = "embedded tsnet runtime is running"
	})
	return nil
}

func startLocalProxy(session *runtimeSession) (*localProxy, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	proxy := &localProxy{
		baseURL:  "http://" + listener.Addr().String(),
		listener: listener,
		dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return session.server.Dial(ctx, network, address)
		},
	}

	reverseProxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			targetBase, authToken := proxy.currentConfig()
			if targetBase == nil {
				return
			}
			req.URL.Scheme = targetBase.Scheme
			req.URL.Host = targetBase.Host
			req.Host = targetBase.Host
			req.URL.Path = joinURLPath(targetBase.Path, req.URL.Path)
			if authToken != "" {
				req.Header.Set("Authorization", "Bearer "+authToken)
			}
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}
		},
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           proxy.dial,
			ForceAttemptHTTP2:     false,
			ResponseHeaderTimeout: 30 * time.Second,
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, fmt.Sprintf("bridge proxy error: %v", err), http.StatusBadGateway)
		},
	}
	proxy.server = &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			targetBase, _ := proxy.currentConfig()
			if targetBase == nil {
				http.Error(w, "bridge proxy is not configured", http.StatusServiceUnavailable)
				return
			}
			reverseProxy.ServeHTTP(w, r)
		}),
	}
	go func() {
		if err := proxy.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("codexmobile: local proxy stopped: %v", err)
		}
	}()
	return proxy, nil
}

func (p *localProxy) configure(targetBase *url.URL, authToken string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if targetBase != nil {
		copy := *targetBase
		p.targetBase = &copy
		p.lastEndpoint = copy.String()
	}
	p.authToken = strings.TrimSpace(authToken)
}

func (p *localProxy) currentConfig() (*url.URL, string) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.targetBase == nil {
		return nil, ""
	}
	copy := *p.targetBase
	return &copy, p.authToken
}

func (p *localProxy) close(ctx context.Context) error {
	if p == nil || p.server == nil {
		return nil
	}
	return p.server.Shutdown(ctx)
}

func normalizeBridgeHTTPBaseURL(endpoint string) (*url.URL, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return nil, errString("bridge endpoint is empty")
	}
	switch {
	case strings.HasPrefix(endpoint, "ws://"):
		endpoint = "http://" + strings.TrimPrefix(endpoint, "ws://")
	case strings.HasPrefix(endpoint, "wss://"):
		endpoint = "https://" + strings.TrimPrefix(endpoint, "wss://")
	case strings.HasPrefix(endpoint, "http://"), strings.HasPrefix(endpoint, "https://"):
	default:
		endpoint = "http://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return nil, errString("bridge endpoint is missing host")
	}
	if parsed.Path == "/" {
		parsed.Path = ""
	}
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed, nil
}

func joinURLPath(basePath, requestPath string) string {
	basePath = strings.TrimSuffix(strings.TrimSpace(basePath), "/")
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" {
		requestPath = "/"
	}
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}
	if basePath == "" {
		return requestPath
	}
	if requestPath == "/" {
		return basePath + "/"
	}
	return basePath + requestPath
}

func proxyURLFromSession(session *runtimeSession) string {
	if session == nil || session.proxy == nil {
		return ""
	}
	return strings.TrimSpace(session.proxy.baseURL)
}

func bridgeIDFromPayload(payload *enrollmentPayload) string {
	if payload == nil {
		return ""
	}
	return payload.BridgeID
}

func bridgeNameFromPayload(payload *enrollmentPayload) string {
	if payload == nil {
		return ""
	}
	return payload.BridgeName
}

func bridgeEndpointFromPayload(payload *enrollmentPayload) string {
	if payload == nil {
		return ""
	}
	return payload.BridgeServerEndpoint
}

func snapshotRuntimeState() runtimeState {
	stateMu.Lock()
	defer stateMu.Unlock()
	return currentState
}

func updateRuntimeState(update func(state *runtimeState)) {
	stateMu.Lock()
	defer stateMu.Unlock()
	update(&currentState)
	currentState.UpdatedAt = time.Now()
}

func parsePayload(raw *C.char) (*enrollmentPayload, error) {
	text := strings.TrimSpace(goString(raw))
	if text == "" {
		return nil, errString("enrollment payload is empty")
	}
	var payload enrollmentPayload
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		return nil, errString("enrollment payload is not valid JSON")
	}
	if payload.Type != "codex-mobile-enrollment" || payload.Version != 1 {
		return nil, errString("unsupported enrollment payload type")
	}
	if strings.TrimSpace(payload.BridgeServerEndpoint) == "" {
		return nil, errString("enrollment payload is missing bridgeServerEndpoint")
	}
	if strings.TrimSpace(payload.Tailnet.AuthKey) == "" {
		return nil, errString("enrollment payload is missing tailnet.authKey")
	}
	return &payload, nil
}

func closeTunFD(fd int) {
	if fd < 0 {
		return
	}
	file := os.NewFile(uintptr(fd), "codexmobile-tun")
	if file != nil {
		_ = file.Close()
	}
}

func jsonCString(response bridgeResponse) *C.char {
	if !response.OK && response.Error == "" {
		response.Error = "unknown error"
	}
	data, err := json.Marshal(response)
	if err != nil {
		data = []byte(`{"ok":false,"error":"failed to marshal response"}`)
	}
	return C.CString(string(data))
}

func goString(value *C.char) string {
	if value == nil {
		return ""
	}
	return C.GoString(value)
}

func errString(message string) error {
	return simpleError(message)
}

type simpleError string

func (e simpleError) Error() string {
	return string(e)
}
