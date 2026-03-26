package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var DirectRPCMethods = map[string]struct{}{
	"account/read":            {},
	"account/rateLimits/read": {},
	"app/list":                {},
	"collaborationMode/list":  {},
	"config/read":             {},
	"mcpServerStatus/list":    {},
	"model/list":              {},
	"skills/list":             {},
	"thread/list":             {},
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
}

type Info struct {
	BridgeHTTPURL      string
	BridgeWebSocketURL string
	BridgeReadyURL     string
	AppServerWebSocket string
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
	config      Config
	logger      *log.Logger
	server      *http.Server
	healthState *healthState
	upgrader    websocket.Upgrader
	closeOnce   sync.Once
	mu          sync.Mutex
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
	if websocket.IsWebSocketUpgrade(r) {
		if b.config.HealthEnabled && (r.URL.Path == b.config.HealthPath || r.URL.Path == b.config.ReadyPath) {
			http.NotFound(w, r)
			return
		}
		b.handleWebSocketProxy(w, r)
		return
	}

	if r.URL.Path == "/codex-mobile/rpc" {
		b.handleDirectRPC(w, r)
		return
	}
	if b.handleMobilePreload(w, r) {
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
		sendJSON(w, http.StatusOK, map[string]any{
			"ok": true,
			"bridge": map[string]any{
				"host":          b.config.Host,
				"port":          b.config.Port,
				"uptimeMs":      time.Since(b.healthState.startedAt).Milliseconds(),
				"websocketPath": "/",
			},
			"upstream": map[string]any{
				"url":        b.config.UpstreamURL,
				"ready":      readiness.OK,
				"checkedAt":  readiness.CheckedAt,
				"durationMs": readiness.DurationMS,
				"error":      emptyToNil(readiness.Error),
			},
			"healthPaths": map[string]any{
				"health": b.config.HealthPath,
				"ready":  b.config.ReadyPath,
			},
			"ui": b.uiStatus(),
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
		sendJSON(w, status, map[string]any{
			"ok": readiness.OK,
			"bridge": map[string]any{
				"host":     b.config.Host,
				"port":     b.config.Port,
				"uptimeMs": time.Since(b.healthState.startedAt).Milliseconds(),
			},
			"upstream": map[string]any{
				"url":        b.config.UpstreamURL,
				"ready":      readiness.OK,
				"checkedAt":  readiness.CheckedAt,
				"durationMs": readiness.DurationMS,
				"error":      emptyToNil(readiness.Error),
			},
		})
		return
	}

	sendText(w, http.StatusNotFound, "Not Found\n")
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
	if _, ok := DirectRPCMethods[payload.Method]; !ok {
		sendJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": "Unsupported direct RPC method."})
		return
	}

	startedAt := time.Now()
	result, err := performUpstreamRPC(r.Context(), b.config, payload.Method, payload.Params)
	if err != nil {
		b.logger.Printf("%s [codex-bridge] direct rpc failed %s %v", nowISO(), payload.Method, err)
		sendJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	b.logger.Printf("%s [codex-bridge] direct rpc %s %dms", nowISO(), payload.Method, time.Since(startedAt).Milliseconds())
	sendJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
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
					return nil, fmt.Errorf(message)
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
