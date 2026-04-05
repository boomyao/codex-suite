package bridge

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/gorilla/websocket"
)

const (
	browserSessionCommandTimeout    = 12 * time.Second
	browserSessionLaunchTimeout     = 15 * time.Second
	browserSessionPollInterval      = 250 * time.Millisecond
	browserSessionFrameQuality      = 55
	browserSessionEditableSyncDelay = 45 * time.Millisecond
)

var browserAttachDefaultPorts = []int{9222, 9223, 9333}

type browserSessionManager struct {
	logger   *log.Logger
	mu       sync.Mutex
	sessions map[string]*browserSession
}

type browserSession struct {
	id          string
	logger      *log.Logger
	source      string
	browserPath string
	debugBase   string
	targetID    string
	userDataDir string
	ownsProcess bool
	cmd         *exec.Cmd
	conn        *websocket.Conn

	writeMu        sync.Mutex
	mu             sync.Mutex
	nextID         int64
	pending        map[int64]chan browserSessionCommandResult
	revisionSignal chan struct{}

	viewport                 browserSessionViewport
	currentURL               string
	title                    string
	loading                  bool
	canGoBack                bool
	canGoForward             bool
	revision                 int64
	consoleLines             []string
	consoleErrorCount        int
	consoleWarnCount         int
	networkInflight          map[string]struct{}
	networkFailed            []string
	networkFailedCount       int
	metadataRefreshScheduled bool
	latestFrameBase64        string
	latestFrameBytes         []byte
	latestFrameMimeType      string
	screencastActive         bool
	textInputActive          bool
	editableText             string
	editableSelectionStart   int
	editableSelectionEnd     int
	pointerActive            bool
	lastPointerX             float64
	lastPointerY             float64
	lastPointerValid         bool
	nextStreamSubscriberID   int64
	streamSubscribers        map[int64]chan browserSessionStreamMessage
	closed                   bool
	lastError                string
}

type browserSessionStreamMessage struct {
	Kind   string
	Status map[string]any
	Frame  []byte
}

type browserSessionViewport struct {
	Width  int
	Height int
	Scale  float64
	Mobile bool
}

type browserSessionCommandEnvelope struct {
	ID     int64            `json:"id,omitempty"`
	Method string           `json:"method,omitempty"`
	Params json.RawMessage  `json:"params,omitempty"`
	Result json.RawMessage  `json:"result,omitempty"`
	Error  *browserCDPError `json:"error,omitempty"`
}

type browserCDPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type browserSessionCommandResult struct {
	Result json.RawMessage
	Err    error
}

type browserEditableState struct {
	Active         bool
	Text           string
	SelectionStart int
	SelectionEnd   int
}

type browserEditableStateJSON struct {
	OK             bool   `json:"ok"`
	Text           string `json:"text"`
	SelectionStart int    `json:"selectionStart"`
	SelectionEnd   int    `json:"selectionEnd"`
}

func (s browserEditableStateJSON) toEditableState() browserEditableState {
	if s.OK == false {
		return browserEditableState{}
	}
	return browserEditableState{
		Active:         true,
		Text:           s.Text,
		SelectionStart: s.SelectionStart,
		SelectionEnd:   s.SelectionEnd,
	}
}

type browserTargetInfo struct {
	ID                   string `json:"id"`
	Title                string `json:"title"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

func newBrowserSessionManager(logger *log.Logger) *browserSessionManager {
	return &browserSessionManager{
		logger:   logger,
		sessions: map[string]*browserSession{},
	}
}

func (m *browserSessionManager) Close(ctx context.Context) {
	m.mu.Lock()
	sessions := make([]*browserSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.sessions = map[string]*browserSession{}
	m.mu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, session := range sessions {
			session.stop()
		}
	}()

	select {
	case <-ctx.Done():
	case <-done:
	}
}

func (m *browserSessionManager) Start(params map[string]any) (map[string]any, error) {
	sessionID, err := randomMobileResourceID()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate browser session: %w", err)
	}
	source := strings.ToLower(strings.TrimSpace(paramStringValue(params, "source")))
	if source == "" {
		source = "preview"
	}

	var session *browserSession
	switch source {
	case "preview":
		browserPath, err := findBrowserExecutable()
		if err != nil {
			return nil, err
		}
		viewport := browserSessionViewportFromParams(params)
		initialURL := normalizedBrowserSessionURL(paramStringValue(params, "url"))
		port, err := allocateBrowserDebugPort()
		if err != nil {
			return nil, fmt.Errorf("failed to allocate browser debug port: %w", err)
		}
		userDataDir, err := os.MkdirTemp("", "codex-mobile-browser-"+sessionID+"-")
		if err != nil {
			return nil, fmt.Errorf("failed to prepare browser session data dir: %w", err)
		}
		session, err = startBrowserSession(sessionID, browserPath, port, userDataDir, initialURL, viewport, m.logger)
		if err != nil {
			_ = os.RemoveAll(userDataDir)
			return nil, err
		}
	case "attach":
		targetID := paramStringValue(params, "targetId")
		if targetID == "" {
			return nil, errors.New("browser attach mode requires a targetId")
		}
		viewport := browserSessionViewportFromParams(params)
		debugBase, err := discoverBrowserDebugBase(paramStringValue(params, "debugBaseURL"))
		if err != nil {
			return nil, err
		}
		session, err = attachBrowserSession(sessionID, debugBase, targetID, viewport, m.logger)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported browser session source: %s", source)
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return session.statusMap(), nil
}

func (m *browserSessionManager) ListTargets(params map[string]any) (map[string]any, error) {
	debugBase, err := discoverBrowserDebugBase(paramStringValue(params, "debugBaseURL"))
	if err != nil {
		return map[string]any{
			"available": false,
			"targets":   []any{},
			"error":     err.Error(),
		}, nil
	}
	client := &http.Client{Timeout: 2 * time.Second}
	targets, err := loadBrowserTargets(client, debugBase)
	if err != nil {
		return map[string]any{
			"available":    false,
			"debugBaseURL": debugBase,
			"targets":      []any{},
			"error":        err.Error(),
		}, nil
	}

	filtered := make([]any, 0, len(targets))
	for _, target := range targets {
		if target.Type != "page" || strings.TrimSpace(target.WebSocketDebuggerURL) == "" {
			continue
		}
		filtered = append(filtered, map[string]any{
			"id":           target.ID,
			"title":        emptyToNil(strings.TrimSpace(target.Title)),
			"url":          emptyToNil(strings.TrimSpace(target.URL)),
			"type":         target.Type,
			"debugBaseURL": debugBase,
		})
	}

	return map[string]any{
		"available":    true,
		"debugBaseURL": debugBase,
		"targets":      filtered,
	}, nil
}

func (m *browserSessionManager) Session(sessionID string) (*browserSession, error) {
	normalizedID := strings.TrimSpace(sessionID)
	if normalizedID == "" {
		return nil, errors.New("browser session id is required")
	}
	m.mu.Lock()
	session := m.sessions[normalizedID]
	m.mu.Unlock()
	if session == nil {
		return nil, errors.New("browser session was not found")
	}
	return session, nil
}

func (m *browserSessionManager) Remove(sessionID string) *browserSession {
	m.mu.Lock()
	session := m.sessions[sessionID]
	delete(m.sessions, sessionID)
	m.mu.Unlock()
	return session
}

func startBrowserSession(
	sessionID string,
	browserPath string,
	port int,
	userDataDir string,
	initialURL string,
	viewport browserSessionViewport,
	logger *log.Logger,
) (*browserSession, error) {
	debugBase := fmt.Sprintf("http://127.0.0.1:%d", port)
	args := []string{
		"--headless=new",
		"--disable-gpu",
		"--no-first-run",
		"--no-default-browser-check",
		"--disable-background-networking",
		"--disable-extensions",
		"--disable-sync",
		"--disable-component-update",
		"--disable-renderer-backgrounding",
		"--hide-crash-restore-bubble",
		"--mute-audio",
		"--remote-debugging-address=127.0.0.1",
		fmt.Sprintf("--remote-debugging-port=%d", port),
		fmt.Sprintf("--user-data-dir=%s", userDataDir),
		fmt.Sprintf("--window-size=%d,%d", viewport.Width, viewport.Height),
		initialURL,
	}

	cmd := exec.Command(browserPath, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	target, err := waitForBrowserTarget(debugBase, browserSessionLaunchTimeout)
	if err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.Dial(target.WebSocketDebuggerURL, nil)
	if err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return nil, fmt.Errorf("failed to connect to browser session: %w", err)
	}

	session := &browserSession{
		id:                sessionID,
		logger:            logger,
		source:            "preview",
		browserPath:       browserPath,
		debugBase:         debugBase,
		userDataDir:       userDataDir,
		ownsProcess:       true,
		cmd:               cmd,
		conn:              conn,
		pending:           map[int64]chan browserSessionCommandResult{},
		revisionSignal:    make(chan struct{}),
		viewport:          viewport,
		currentURL:        target.URL,
		title:             target.Title,
		loading:           true,
		networkInflight:   map[string]struct{}{},
		streamSubscribers: map[int64]chan browserSessionStreamMessage{},
	}
	go session.readLoop()
	go session.waitForExit()

	if err := session.configure(); err != nil {
		session.stop()
		return nil, err
	}
	session.refreshSessionMetadata()

	if initialURL != "" && initialURL != target.URL && initialURL != "about:blank" {
		if _, err := session.navigateToURL(initialURL); err != nil {
			session.stop()
			return nil, err
		}
	}

	return session, nil
}

func attachBrowserSession(
	sessionID string,
	debugBase string,
	targetID string,
	viewport browserSessionViewport,
	logger *log.Logger,
) (*browserSession, error) {
	target, err := loadBrowserTargetByID(debugBase, targetID)
	if err != nil {
		return nil, err
	}
	conn, _, err := websocket.DefaultDialer.Dial(target.WebSocketDebuggerURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to attach browser tab: %w", err)
	}

	session := &browserSession{
		id:                sessionID,
		logger:            logger,
		source:            "attach",
		debugBase:         debugBase,
		targetID:          targetID,
		conn:              conn,
		pending:           map[int64]chan browserSessionCommandResult{},
		revisionSignal:    make(chan struct{}),
		viewport:          viewport,
		currentURL:        target.URL,
		title:             target.Title,
		networkInflight:   map[string]struct{}{},
		streamSubscribers: map[int64]chan browserSessionStreamMessage{},
	}
	go session.readLoop()
	if err := session.configure(); err != nil {
		session.stop()
		return nil, err
	}
	session.refreshSessionMetadata()
	return session, nil
}

func (s *browserSession) configure() error {
	commands := []struct {
		Method string
		Params map[string]any
	}{
		{Method: "Page.enable"},
		{Method: "Runtime.enable"},
		{Method: "Log.enable"},
		{Method: "Network.enable"},
		{Method: "Page.setLifecycleEventsEnabled", Params: map[string]any{"enabled": true}},
		{Method: "Emulation.setDeviceMetricsOverride", Params: map[string]any{
			"width":             s.viewport.Width,
			"height":            s.viewport.Height,
			"deviceScaleFactor": s.viewport.Scale,
			"mobile":            s.viewport.Mobile,
		}},
		{Method: "Emulation.setTouchEmulationEnabled", Params: map[string]any{
			"enabled":        s.viewport.Mobile,
			"maxTouchPoints": 1,
		}},
	}

	for _, command := range commands {
		if _, err := s.command(command.Method, command.Params); err != nil {
			return fmt.Errorf("%s failed: %w", command.Method, err)
		}
	}
	if err := s.restartScreencast(); err != nil {
		s.logger.Printf("browser session screencast unavailable, falling back to on-demand screenshots: %v", err)
	}
	return nil
}

func (s *browserSession) waitForExit() {
	err := s.cmd.Wait()
	message := ""
	if err != nil {
		message = err.Error()
	}
	s.closeWithError(message)
}

func (s *browserSession) closeWithError(message string) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.lastError = strings.TrimSpace(message)
	pending := s.pending
	s.pending = map[int64]chan browserSessionCommandResult{}
	revisionSignal := s.revisionSignal
	s.revisionSignal = nil
	streamSubscribers := s.streamSubscribers
	s.streamSubscribers = map[int64]chan browserSessionStreamMessage{}
	s.mu.Unlock()

	if revisionSignal != nil {
		close(revisionSignal)
	}
	for _, ch := range streamSubscribers {
		close(ch)
	}
	for _, ch := range pending {
		ch <- browserSessionCommandResult{Err: errors.New("browser session closed")}
	}
	_ = s.conn.Close()
}

func (s *browserSession) stop() {
	s.closeWithError("")
	if s.ownsProcess && s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	if strings.TrimSpace(s.userDataDir) != "" {
		_ = os.RemoveAll(s.userDataDir)
	}
}

func (s *browserSession) readLoop() {
	for {
		var envelope browserSessionCommandEnvelope
		if err := s.conn.ReadJSON(&envelope); err != nil {
			s.closeWithError(err.Error())
			return
		}

		if envelope.ID != 0 {
			s.mu.Lock()
			responseCh := s.pending[envelope.ID]
			delete(s.pending, envelope.ID)
			s.mu.Unlock()
			if responseCh != nil {
				if envelope.Error != nil {
					responseCh <- browserSessionCommandResult{
						Err: fmt.Errorf("%s", envelope.Error.Message),
					}
				} else {
					responseCh <- browserSessionCommandResult{Result: envelope.Result}
				}
			}
			continue
		}

		s.handleEvent(envelope.Method, envelope.Params)
	}
}

func (s *browserSession) handleEvent(method string, params json.RawMessage) {
	shouldBroadcastStatus := false
	switch method {
	case "Page.screencastFrame":
		var payload struct {
			Data      string `json:"data"`
			SessionID int    `json:"sessionId"`
		}
		if json.Unmarshal(params, &payload) == nil {
			var frameBytes []byte
			if payload.Data != "" {
				if decoded, err := base64.StdEncoding.DecodeString(payload.Data); err == nil {
					frameBytes = decoded
				}
			}
			s.mu.Lock()
			if payload.Data != "" {
				s.latestFrameBase64 = payload.Data
				s.latestFrameBytes = cloneBytes(frameBytes)
				s.latestFrameMimeType = "image/jpeg"
				s.screencastActive = true
				s.advanceRevisionLocked()
			}
			s.mu.Unlock()
			if len(frameBytes) != 0 {
				s.broadcastStreamFrame(frameBytes)
			}
			if payload.SessionID != 0 {
				s.fireAndForgetCommand("Page.screencastFrameAck", map[string]any{"sessionId": payload.SessionID})
			}
		}
	case "Page.frameStartedLoading":
		s.mu.Lock()
		s.loading = true
		s.advanceRevisionLocked()
		s.textInputActive = false
		s.pointerActive = false
		s.mu.Unlock()
		shouldBroadcastStatus = true
	case "Page.frameStoppedLoading", "Page.loadEventFired":
		s.mu.Lock()
		s.loading = false
		s.advanceRevisionLocked()
		s.mu.Unlock()
		shouldBroadcastStatus = true
		s.queueMetadataRefresh()
	case "Page.frameNavigated":
		var payload struct {
			Frame struct {
				URL string `json:"url"`
			} `json:"frame"`
		}
		if json.Unmarshal(params, &payload) == nil {
			s.mu.Lock()
			if payload.Frame.URL != "" {
				s.currentURL = payload.Frame.URL
			}
			s.loading = true
			s.advanceRevisionLocked()
			s.textInputActive = false
			s.pointerActive = false
			s.mu.Unlock()
			shouldBroadcastStatus = true
			s.queueMetadataRefresh()
		}
	case "Page.navigatedWithinDocument":
		var payload struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(params, &payload) == nil {
			s.mu.Lock()
			if payload.URL != "" {
				s.currentURL = payload.URL
			}
			s.advanceRevisionLocked()
			s.textInputActive = false
			s.pointerActive = false
			s.mu.Unlock()
			shouldBroadcastStatus = true
			s.queueMetadataRefresh()
		}
	case "Runtime.consoleAPICalled":
		var payload struct {
			Type string `json:"type"`
			Args []struct {
				Value       any    `json:"value"`
				Description string `json:"description"`
			} `json:"args"`
		}
		if json.Unmarshal(params, &payload) == nil {
			line := browserConsoleLine(payload.Args)
			s.mu.Lock()
			switch payload.Type {
			case "warning":
				s.consoleWarnCount++
			case "error", "assert":
				s.consoleErrorCount++
			}
			if line != "" {
				s.consoleLines = appendLimitedLines(s.consoleLines, line, 6)
			}
			s.advanceRevisionLocked()
			s.mu.Unlock()
		}
	case "Log.entryAdded":
		var payload struct {
			Entry struct {
				Level string `json:"level"`
				Text  string `json:"text"`
			} `json:"entry"`
		}
		if json.Unmarshal(params, &payload) == nil {
			text := strings.TrimSpace(payload.Entry.Text)
			s.mu.Lock()
			switch payload.Entry.Level {
			case "warning":
				s.consoleWarnCount++
			case "error":
				s.consoleErrorCount++
			}
			if text != "" {
				s.consoleLines = appendLimitedLines(s.consoleLines, text, 6)
			}
			s.advanceRevisionLocked()
			s.mu.Unlock()
		}
	case "Network.requestWillBeSent":
		var payload struct {
			RequestID string `json:"requestId"`
		}
		if json.Unmarshal(params, &payload) == nil && payload.RequestID != "" {
			s.mu.Lock()
			s.networkInflight[payload.RequestID] = struct{}{}
			s.advanceRevisionLocked()
			s.mu.Unlock()
		}
	case "Network.loadingFinished":
		var payload struct {
			RequestID string `json:"requestId"`
		}
		if json.Unmarshal(params, &payload) == nil && payload.RequestID != "" {
			s.mu.Lock()
			delete(s.networkInflight, payload.RequestID)
			s.advanceRevisionLocked()
			s.mu.Unlock()
		}
	case "Network.loadingFailed":
		var payload struct {
			RequestID string `json:"requestId"`
			ErrorText string `json:"errorText"`
		}
		if json.Unmarshal(params, &payload) == nil {
			s.mu.Lock()
			delete(s.networkInflight, payload.RequestID)
			s.networkFailedCount++
			if payload.ErrorText != "" {
				s.networkFailed = appendLimitedLines(s.networkFailed, payload.ErrorText, 4)
			}
			s.advanceRevisionLocked()
			s.mu.Unlock()
		}
	}
	if shouldBroadcastStatus {
		s.broadcastStreamStatus()
	}
}

func (s *browserSession) command(method string, params map[string]any) (json.RawMessage, error) {
	s.mu.Lock()
	if s.closed {
		lastError := s.lastError
		s.mu.Unlock()
		if lastError == "" {
			return nil, errors.New("browser session closed")
		}
		return nil, fmt.Errorf("browser session closed: %s", lastError)
	}
	s.nextID++
	commandID := s.nextID
	responseCh := make(chan browserSessionCommandResult, 1)
	s.pending[commandID] = responseCh
	s.mu.Unlock()

	request := map[string]any{
		"id":     commandID,
		"method": method,
	}
	if params != nil {
		request["params"] = params
	}

	s.writeMu.Lock()
	writeErr := s.conn.WriteJSON(request)
	s.writeMu.Unlock()
	if writeErr != nil {
		s.mu.Lock()
		delete(s.pending, commandID)
		s.mu.Unlock()
		return nil, writeErr
	}

	timer := time.NewTimer(browserSessionCommandTimeout)
	defer timer.Stop()

	select {
	case result := <-responseCh:
		return result.Result, result.Err
	case <-timer.C:
		s.mu.Lock()
		delete(s.pending, commandID)
		s.mu.Unlock()
		return nil, fmt.Errorf("%s timed out", method)
	}
}

func (s *browserSession) advanceRevisionLocked() {
	s.revision++
	if s.revisionSignal == nil {
		s.revisionSignal = make(chan struct{})
		return
	}
	previousSignal := s.revisionSignal
	s.revisionSignal = make(chan struct{})
	close(previousSignal)
}

func (s *browserSession) currentRevision() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revision
}

func (s *browserSession) waitForRevision(afterRevision int64, timeout time.Duration) error {
	for {
		s.mu.Lock()
		if s.closed {
			errMessage := strings.TrimSpace(s.lastError)
			s.mu.Unlock()
			if errMessage == "" {
				errMessage = "browser session closed"
			}
			return errors.New(errMessage)
		}
		if s.revision > afterRevision {
			s.mu.Unlock()
			return nil
		}
		signal := s.revisionSignal
		s.mu.Unlock()

		if timeout <= 0 {
			<-signal
			continue
		}

		timer := time.NewTimer(timeout)
		select {
		case <-signal:
			if !timer.Stop() {
				<-timer.C
			}
		case <-timer.C:
			return nil
		}
	}
}

func (s *browserSession) fireAndForgetCommand(method string, params map[string]any) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.nextID++
	commandID := s.nextID
	s.mu.Unlock()

	request := map[string]any{
		"id":     commandID,
		"method": method,
	}
	if params != nil {
		request["params"] = params
	}

	s.writeMu.Lock()
	_ = s.conn.WriteJSON(request)
	s.writeMu.Unlock()
}

func (s *browserSession) subscribeStream() (int64, <-chan browserSessionStreamMessage, map[string]any, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		errMessage := strings.TrimSpace(s.lastError)
		if errMessage == "" {
			errMessage = "browser session closed"
		}
		return 0, nil, nil, nil, errors.New(errMessage)
	}
	s.nextStreamSubscriberID++
	subscriberID := s.nextStreamSubscriberID
	ch := make(chan browserSessionStreamMessage, 4)
	if s.streamSubscribers == nil {
		s.streamSubscribers = map[int64]chan browserSessionStreamMessage{}
	}
	s.streamSubscribers[subscriberID] = ch
	return subscriberID, ch, s.statusMapLocked(), cloneBytes(s.latestFrameBytes), nil
}

func (s *browserSession) unsubscribeStream(subscriberID int64) {
	s.mu.Lock()
	ch := s.streamSubscribers[subscriberID]
	delete(s.streamSubscribers, subscriberID)
	s.mu.Unlock()
	if ch != nil {
		close(ch)
	}
}

func (s *browserSession) broadcastStreamStatus() {
	s.mu.Lock()
	status := s.statusMapLocked()
	subscribers := appendStreamSubscriberChannels(nil, s.streamSubscribers)
	s.mu.Unlock()
	if len(subscribers) == 0 {
		return
	}
	message := browserSessionStreamMessage{
		Kind:   "status",
		Status: status,
	}
	for _, ch := range subscribers {
		deliverBrowserSessionStreamMessage(ch, message)
	}
}

func (s *browserSession) broadcastStreamFrame(frame []byte) {
	if len(frame) == 0 {
		return
	}
	s.mu.Lock()
	subscribers := appendStreamSubscriberChannels(nil, s.streamSubscribers)
	s.mu.Unlock()
	if len(subscribers) == 0 {
		return
	}
	message := browserSessionStreamMessage{
		Kind:  "frame",
		Frame: cloneBytes(frame),
	}
	for _, ch := range subscribers {
		deliverBrowserSessionStreamMessage(ch, message)
	}
}

func (s *browserSession) statusMap() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.statusMapLocked()
}

func (s *browserSession) statusMapLocked() map[string]any {
	if s.streamSubscribers == nil {
		s.streamSubscribers = map[int64]chan browserSessionStreamMessage{}
	}

	return map[string]any{
		"sessionId":       s.id,
		"source":          s.source,
		"url":             emptyToNil(s.currentURL),
		"title":           emptyToNil(s.title),
		"loading":         s.loading,
		"canGoBack":       s.canGoBack,
		"canGoForward":    s.canGoForward,
		"viewport":        browserViewportMap(s.viewport),
		"revision":        s.revision,
		"browserPath":     s.browserPath,
		"debugBaseURL":    emptyToNil(s.debugBase),
		"targetId":        emptyToNil(s.targetID),
		"streaming":       s.screencastActive,
		"textInputActive": s.textInputActive,
		"editableText":    s.editableText,
		"selectionStart":  s.editableSelectionStart,
		"selectionEnd":    s.editableSelectionEnd,
		"active":          !s.closed,
		"lastError":       emptyToNil(s.lastError),
	}
}

func (s *browserSession) snapshotMap(quality int) (map[string]any, error) {
	if quality <= 0 {
		quality = browserSessionFrameQuality
	}

	s.mu.Lock()
	imageBase64 := s.latestFrameBase64
	mimeType := s.latestFrameMimeType
	s.mu.Unlock()

	if imageBase64 == "" {
		result, err := s.command("Page.captureScreenshot", map[string]any{
			"format":           "jpeg",
			"quality":          quality,
			"fromSurface":      true,
			"optimizeForSpeed": true,
		})
		if err != nil {
			return nil, err
		}
		var payload struct {
			Data string `json:"data"`
		}
		if err := json.Unmarshal(result, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse screenshot payload: %w", err)
		}
		if payload.Data == "" {
			return nil, errors.New("browser session returned an empty screenshot")
		}
		imageBase64 = payload.Data
		mimeType = "image/jpeg"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return map[string]any{
		"sessionId":       s.id,
		"source":          s.source,
		"revision":        s.revision,
		"url":             emptyToNil(s.currentURL),
		"title":           emptyToNil(s.title),
		"loading":         s.loading,
		"canGoBack":       s.canGoBack,
		"canGoForward":    s.canGoForward,
		"width":           s.viewport.Width,
		"height":          s.viewport.Height,
		"mimeType":        emptyToNil(mimeType),
		"imageBase64":     imageBase64,
		"streaming":       s.screencastActive,
		"textInputActive": s.textInputActive,
		"editableText":    s.editableText,
		"selectionStart":  s.editableSelectionStart,
		"selectionEnd":    s.editableSelectionEnd,
		"console": map[string]any{
			"errorCount": s.consoleErrorCount,
			"warnCount":  s.consoleWarnCount,
			"lastLines":  append([]string(nil), s.consoleLines...),
		},
		"network": map[string]any{
			"inflight":    len(s.networkInflight),
			"failedCount": s.networkFailedCount,
			"lastFailed":  append([]string(nil), s.networkFailed...),
		},
	}, nil
}

func (s *browserSession) queryTitle() (string, error) {
	result, err := s.command("Runtime.evaluate", map[string]any{
		"expression":    "document.title",
		"returnByValue": true,
	})
	if err != nil {
		return "", err
	}
	var payload struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.Result.Value), nil
}

func (s *browserSession) navigationCapabilities() (bool, bool, error) {
	result, err := s.command("Page.getNavigationHistory", nil)
	if err != nil {
		return false, false, err
	}
	var payload struct {
		CurrentIndex int `json:"currentIndex"`
		Entries      []struct {
			ID int `json:"id"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return false, false, err
	}
	if len(payload.Entries) == 0 {
		return false, false, nil
	}
	return payload.CurrentIndex > 0, payload.CurrentIndex < len(payload.Entries)-1, nil
}

func (s *browserSession) navigate(params map[string]any) (map[string]any, error) {
	action := strings.TrimSpace(paramStringValue(params, "action"))
	switch action {
	case "", "navigate":
		urlValue := normalizedBrowserSessionURL(paramStringValue(params, "url"))
		if urlValue == "" {
			return nil, errors.New("browser navigation url is required")
		}
		return s.navigateToURL(urlValue)
	case "reload":
		if _, err := s.command("Page.reload", map[string]any{"ignoreCache": true}); err != nil {
			return nil, err
		}
	case "back", "forward":
		entryID, err := s.historyEntryID(action == "back")
		if err != nil {
			return nil, err
		}
		if entryID == 0 {
			return s.statusMap(), nil
		}
		if _, err := s.command("Page.navigateToHistoryEntry", map[string]any{"entryId": entryID}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported browser navigation action: %s", action)
	}

	s.mu.Lock()
	s.loading = true
	s.advanceRevisionLocked()
	s.textInputActive = false
	s.mu.Unlock()
	s.queueMetadataRefresh()
	return s.statusMap(), nil
}

func (s *browserSession) navigateToURL(urlValue string) (map[string]any, error) {
	if _, err := s.command("Page.navigate", map[string]any{"url": urlValue}); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.currentURL = urlValue
	s.loading = true
	s.advanceRevisionLocked()
	s.textInputActive = false
	s.mu.Unlock()
	s.queueMetadataRefresh()
	return s.statusMap(), nil
}

func (s *browserSession) historyEntryID(back bool) (int, error) {
	result, err := s.command("Page.getNavigationHistory", nil)
	if err != nil {
		return 0, err
	}
	var payload struct {
		CurrentIndex int `json:"currentIndex"`
		Entries      []struct {
			ID int `json:"id"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return 0, err
	}
	targetIndex := payload.CurrentIndex
	if back {
		targetIndex--
	} else {
		targetIndex++
	}
	if targetIndex < 0 || targetIndex >= len(payload.Entries) {
		return 0, nil
	}
	return payload.Entries[targetIndex].ID, nil
}

func (s *browserSession) input(params map[string]any) (map[string]any, error) {
	inputType := strings.TrimSpace(paramStringValue(params, "type"))
	switch inputType {
	case "tap", "doubleTap":
		xNorm := numericParamWithDefault(params, "xNorm", 0.5)
		yNorm := numericParamWithDefault(params, "yNorm", 0.5)
		x, y := s.normalizedPoint(xNorm, yNorm)
		clickCount := 1
		if inputType == "doubleTap" {
			clickCount = 2
		}
		s.rememberPointer(x, y)
		for i := 0; i < clickCount; i++ {
			s.dispatchTap(x, y)
		}
		editableState, err := s.focusEditableNearPointer(x, y)
		if err != nil {
			s.logger.Printf("browser session focus hint failed: %v", err)
		}
		s.applyEditableState(editableState)
	case "pointerDown":
		xNorm := numericParamWithDefault(params, "xNorm", 0.5)
		yNorm := numericParamWithDefault(params, "yNorm", 0.5)
		x, y := s.normalizedPoint(xNorm, yNorm)
		s.rememberPointer(x, y)
		s.dispatchPointerDown(x, y)
	case "pointerMove":
		xNorm := numericParamWithDefault(params, "xNorm", 0.5)
		yNorm := numericParamWithDefault(params, "yNorm", 0.5)
		x, y := s.normalizedPoint(xNorm, yNorm)
		s.rememberPointer(x, y)
		s.dispatchPointerMove(x, y)
	case "pointerUp":
		xNorm := numericParamWithDefault(params, "xNorm", 0.5)
		yNorm := numericParamWithDefault(params, "yNorm", 0.5)
		x, y := s.normalizedPoint(xNorm, yNorm)
		s.rememberPointer(x, y)
		s.dispatchPointerUp(x, y)
		editableState, err := s.focusEditableNearPointer(x, y)
		if err != nil {
			s.logger.Printf("browser session focus hint failed after pointer up: %v", err)
		}
		s.applyEditableState(editableState)
	case "scroll":
		xNorm := numericParamWithDefault(params, "xNorm", 0.5)
		yNorm := numericParamWithDefault(params, "yNorm", 0.5)
		x, y := s.normalizedPoint(xNorm, yNorm)
		s.rememberPointer(x, y)
		s.fireAndForgetCommand("Input.dispatchMouseEvent", map[string]any{
			"type":   "mouseWheel",
			"x":      x,
			"y":      y,
			"deltaX": numericParamWithDefault(params, "deltaX", 0),
			"deltaY": numericParamWithDefault(params, "deltaY", 120),
		})
	case "text":
		text := rawStringParamValue(params, "text")
		if text == "" {
			return nil, errors.New("browser text input requires text")
		}
		if err := s.insertText(text); err != nil {
			return nil, err
		}
		editableState, err := s.activeEditableState(browserSessionEditableSyncDelay)
		if err != nil {
			s.logger.Printf("browser session editable sync failed after text input: %v", err)
			editableState = browserEditableState{
				Active:         true,
				Text:           s.nextEditableTextAfterInsert(text),
				SelectionStart: s.nextEditableSelectionAfterInsert(text),
				SelectionEnd:   s.nextEditableSelectionAfterInsert(text),
			}
		}
		s.applyEditableState(editableState)
	case "textState":
		text := rawStringParamValue(params, "text")
		defaultSelection := float64(utf16Len(text))
		selectionStart := int(numericParamWithDefault(params, "selectionStart", defaultSelection))
		selectionEnd := int(numericParamWithDefault(params, "selectionEnd", float64(selectionStart)))
		if err := s.setEditableState(text, selectionStart, selectionEnd); err != nil {
			return nil, err
		}
		editableState, err := s.activeEditableState(browserSessionEditableSyncDelay)
		if err != nil {
			s.logger.Printf("browser session editable sync failed after text state update: %v", err)
			editableState = browserEditableState{
				Active:         true,
				Text:           text,
				SelectionStart: selectionStart,
				SelectionEnd:   selectionEnd,
			}
		}
		s.applyEditableState(editableState)
	case "key":
		key := strings.TrimSpace(paramStringValue(params, "key"))
		if key == "" {
			return nil, errors.New("browser key input requires key")
		}
		if err := s.dispatchKey(key); err != nil {
			return nil, err
		}
		if strings.EqualFold(key, "escape") {
			s.applyEditableState(browserEditableState{})
		} else if browserKeyAffectsEditableState(key) || s.textInputEnabled() {
			editableState, err := s.activeEditableState(browserSessionEditableSyncDelay)
			if err != nil {
				s.logger.Printf("browser session editable sync failed after key input %q: %v", key, err)
			} else {
				s.applyEditableState(editableState)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported browser input type: %s", inputType)
	}

	s.mu.Lock()
	s.advanceRevisionLocked()
	s.mu.Unlock()
	return s.statusMap(), nil
}

func (s *browserSession) textInputEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.textInputActive
}

func (s *browserSession) dispatchKey(key string) error {
	keyPayloads := browserKeyPayloads(key)
	if len(keyPayloads) == 0 {
		return fmt.Errorf("unsupported browser key: %s", key)
	}
	for _, payload := range keyPayloads {
		s.fireAndForgetCommand("Input.dispatchKeyEvent", payload)
	}
	return nil
}

func (s *browserSession) resize(params map[string]any) (map[string]any, error) {
	viewport := browserSessionViewportFromParams(params)
	if _, err := s.command("Emulation.setDeviceMetricsOverride", map[string]any{
		"width":             viewport.Width,
		"height":            viewport.Height,
		"deviceScaleFactor": viewport.Scale,
		"mobile":            viewport.Mobile,
	}); err != nil {
		return nil, err
	}
	if _, err := s.command("Emulation.setTouchEmulationEnabled", map[string]any{
		"enabled":        viewport.Mobile,
		"maxTouchPoints": 1,
	}); err != nil {
		return nil, err
	}
	if err := s.restartScreencast(); err != nil {
		s.logger.Printf("browser session screencast restart failed, keeping on-demand screenshots: %v", err)
	}
	s.mu.Lock()
	s.viewport = viewport
	s.advanceRevisionLocked()
	s.textInputActive = false
	s.pointerActive = false
	s.mu.Unlock()
	s.queueMetadataRefresh()
	return s.statusMap(), nil
}

func (s *browserSession) restartScreencast() error {
	s.mu.Lock()
	viewport := s.viewport
	s.screencastActive = false
	s.latestFrameBase64 = ""
	s.latestFrameBytes = nil
	s.latestFrameMimeType = ""
	s.mu.Unlock()

	s.fireAndForgetCommand("Page.stopScreencast", nil)
	if _, err := s.command("Page.startScreencast", map[string]any{
		"format":        "jpeg",
		"quality":       browserSessionFrameQuality,
		"maxWidth":      viewport.Width,
		"maxHeight":     viewport.Height,
		"everyNthFrame": 1,
	}); err != nil {
		return fmt.Errorf("Page.startScreencast failed: %w", err)
	}
	s.mu.Lock()
	s.latestFrameMimeType = "image/jpeg"
	s.screencastActive = true
	s.mu.Unlock()
	return nil
}

func (s *browserSession) dispatchTap(x, y float64) {
	s.dispatchPointerDown(x, y)
	s.dispatchPointerUp(x, y)
}

func (s *browserSession) dispatchPointerDown(x, y float64) {
	s.mu.Lock()
	mobile := s.viewport.Mobile
	wasActive := s.pointerActive
	s.pointerActive = true
	s.mu.Unlock()
	if wasActive {
		s.dispatchPointerMove(x, y)
		return
	}
	if mobile {
		s.fireAndForgetCommand("Input.dispatchTouchEvent", map[string]any{
			"type": "touchStart",
			"touchPoints": []map[string]any{{
				"x":       x,
				"y":       y,
				"radiusX": 1,
				"radiusY": 1,
				"force":   1,
				"id":      1,
			}},
		})
		return
	}
	s.fireAndForgetCommand("Input.dispatchMouseEvent", map[string]any{
		"type":    "mouseMoved",
		"x":       x,
		"y":       y,
		"button":  "none",
		"buttons": 0,
	})
	s.fireAndForgetCommand("Input.dispatchMouseEvent", map[string]any{
		"type":       "mousePressed",
		"x":          x,
		"y":          y,
		"button":     "left",
		"buttons":    1,
		"clickCount": 1,
	})
}

func (s *browserSession) dispatchPointerMove(x, y float64) {
	s.mu.Lock()
	mobile := s.viewport.Mobile
	active := s.pointerActive
	s.mu.Unlock()
	if !active {
		return
	}
	if mobile {
		s.fireAndForgetCommand("Input.dispatchTouchEvent", map[string]any{
			"type": "touchMove",
			"touchPoints": []map[string]any{{
				"x":       x,
				"y":       y,
				"radiusX": 1,
				"radiusY": 1,
				"force":   1,
				"id":      1,
			}},
		})
		return
	}
	s.fireAndForgetCommand("Input.dispatchMouseEvent", map[string]any{
		"type":    "mouseMoved",
		"x":       x,
		"y":       y,
		"button":  "left",
		"buttons": 1,
	})
}

func (s *browserSession) dispatchPointerUp(x, y float64) {
	s.mu.Lock()
	mobile := s.viewport.Mobile
	active := s.pointerActive
	s.pointerActive = false
	s.mu.Unlock()
	if !active {
		return
	}
	if mobile {
		s.fireAndForgetCommand("Input.dispatchTouchEvent", map[string]any{
			"type":        "touchEnd",
			"touchPoints": []map[string]any{},
		})
		return
	}
	s.fireAndForgetCommand("Input.dispatchMouseEvent", map[string]any{
		"type":       "mouseReleased",
		"x":          x,
		"y":          y,
		"button":     "left",
		"buttons":    0,
		"clickCount": 1,
	})
}

func (s *browserSession) insertText(text string) error {
	x, y := s.textInsertionPoint()
	if _, err := s.focusEditableNearPointer(x, y); err != nil {
		s.logger.Printf("browser session focus before text failed: %v", err)
	}
	expression, err := browserInsertTextExpression(text, x, y)
	if err != nil {
		return err
	}
	result, err := s.command("Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
		"userGesture":   true,
	})
	if err != nil {
		return err
	}
	var payload struct {
		Result struct {
			Value struct {
				OK     bool   `json:"ok"`
				Reason string `json:"reason"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return fmt.Errorf("failed to parse browser text insertion result: %w", err)
	}
	if payload.Result.Value.OK {
		return nil
	}
	if _, err := s.command("Input.insertText", map[string]any{"text": text}); err == nil {
		return nil
	}
	if payload.Result.Value.Reason != "" {
		return fmt.Errorf("browser text input failed: %s", payload.Result.Value.Reason)
	}
	return errors.New("browser text input failed")
}

func (s *browserSession) rememberPointer(x, y float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastPointerX = x
	s.lastPointerY = y
	s.lastPointerValid = true
}

func (s *browserSession) textInsertionPoint() (float64, float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastPointerValid {
		return s.lastPointerX, s.lastPointerY
	}
	return float64(s.viewport.Width) / 2, float64(s.viewport.Height) / 2
}

func (s *browserSession) focusEditableNearPointer(x, y float64) (browserEditableState, error) {
	expression := browserFocusEditableExpression(x, y)
	result, err := s.command("Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
		"userGesture":   true,
	})
	if err != nil {
		return browserEditableState{}, err
	}
	var payload struct {
		Result struct {
			Value browserEditableStateJSON `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return browserEditableState{}, fmt.Errorf("failed to parse browser focus result: %w", err)
	}
	return payload.Result.Value.toEditableState(), nil
}

func (s *browserSession) activeEditableState(delay time.Duration) (browserEditableState, error) {
	expression := browserActiveEditableStateExpression(delay)
	result, err := s.command("Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
		"userGesture":   true,
	})
	if err != nil {
		return browserEditableState{}, err
	}
	var payload struct {
		Result struct {
			Value browserEditableStateJSON `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return browserEditableState{}, fmt.Errorf("failed to parse browser editable state result: %w", err)
	}
	return payload.Result.Value.toEditableState(), nil
}

func (s *browserSession) applyEditableState(state browserEditableState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.textInputActive = state.Active
	s.editableText = state.Text
	textLength := utf16Len(state.Text)
	s.editableSelectionStart = clampInt(state.SelectionStart, 0, textLength)
	s.editableSelectionEnd = clampInt(state.SelectionEnd, s.editableSelectionStart, textLength)
}

func (s *browserSession) nextEditableTextAfterInsert(text string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	start := clampInt(s.editableSelectionStart, 0, utf16Len(s.editableText))
	end := clampInt(s.editableSelectionEnd, start, utf16Len(s.editableText))
	return replaceStringByUTF16Range(s.editableText, start, end, text)
}

func (s *browserSession) nextEditableSelectionAfterInsert(text string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	start := clampInt(s.editableSelectionStart, 0, utf16Len(s.editableText))
	return start + utf16Len(text)
}

func (s *browserSession) setEditableState(text string, selectionStart, selectionEnd int) error {
	x, y := s.textInsertionPoint()
	expression, err := browserSetEditableStateExpression(text, selectionStart, selectionEnd, x, y)
	if err != nil {
		return err
	}
	result, err := s.command("Runtime.evaluate", map[string]any{
		"expression":    expression,
		"returnByValue": true,
		"awaitPromise":  true,
		"userGesture":   true,
	})
	if err != nil {
		return err
	}
	var payload struct {
		Result struct {
			Value struct {
				OK     bool   `json:"ok"`
				Reason string `json:"reason"`
			} `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return fmt.Errorf("failed to parse browser text state result: %w", err)
	}
	if payload.Result.Value.OK {
		return nil
	}
	if payload.Result.Value.Reason != "" {
		return fmt.Errorf("browser text state sync failed: %s", payload.Result.Value.Reason)
	}
	return nil
}

func (s *browserSession) syncEditableState(delay time.Duration) (map[string]any, error) {
	editableState, err := s.activeEditableState(delay)
	if err != nil {
		return nil, err
	}
	s.applyEditableState(editableState)
	s.mu.Lock()
	s.advanceRevisionLocked()
	s.mu.Unlock()
	return s.statusMap(), nil
}

func (s *browserSession) queueMetadataRefresh() {
	s.mu.Lock()
	if s.closed || s.metadataRefreshScheduled {
		s.mu.Unlock()
		return
	}
	s.metadataRefreshScheduled = true
	s.mu.Unlock()

	go func() {
		time.Sleep(120 * time.Millisecond)
		s.refreshSessionMetadata()
		s.mu.Lock()
		s.metadataRefreshScheduled = false
		s.mu.Unlock()
	}()
}

func (s *browserSession) refreshSessionMetadata() {
	title, titleErr := s.queryTitle()
	canGoBack, canGoForward, historyErr := s.navigationCapabilities()

	s.mu.Lock()
	changed := false
	if titleErr == nil && title != "" {
		if s.title != title {
			s.title = title
			changed = true
		}
	}
	if historyErr == nil {
		if s.canGoBack != canGoBack || s.canGoForward != canGoForward {
			s.canGoBack = canGoBack
			s.canGoForward = canGoForward
			changed = true
		}
	}
	if changed {
		s.advanceRevisionLocked()
	}
	s.mu.Unlock()
	if changed {
		s.broadcastStreamStatus()
	}
}

func (s *browserSession) normalizedPoint(xNorm, yNorm float64) (float64, float64) {
	s.mu.Lock()
	viewport := s.viewport
	s.mu.Unlock()

	xNorm = clamp(xNorm, 0, 1)
	yNorm = clamp(yNorm, 0, 1)
	return xNorm * float64(viewport.Width), yNorm * float64(viewport.Height)
}

func browserViewportMap(viewport browserSessionViewport) map[string]any {
	return map[string]any{
		"width":  viewport.Width,
		"height": viewport.Height,
		"scale":  viewport.Scale,
		"mobile": viewport.Mobile,
	}
}

func browserSessionViewportFromParams(params map[string]any) browserSessionViewport {
	viewport := browserSessionViewport{
		Width:  1280,
		Height: 900,
		Scale:  1,
		Mobile: false,
	}

	preset := strings.ToLower(strings.TrimSpace(paramStringValue(params, "preset")))
	switch preset {
	case "mobile":
		viewport = browserSessionViewport{Width: 430, Height: 932, Scale: 2, Mobile: true}
	case "tablet":
		viewport = browserSessionViewport{Width: 1024, Height: 1366, Scale: 2, Mobile: true}
	case "desktop":
		viewport = browserSessionViewport{Width: 1440, Height: 960, Scale: 1, Mobile: false}
	}

	rawViewport, _ := params["viewport"].(map[string]any)
	if width := int(numericParamWithDefault(rawViewport, "width", float64(viewport.Width))); width > 0 {
		viewport.Width = width
	}
	if height := int(numericParamWithDefault(rawViewport, "height", float64(viewport.Height))); height > 0 {
		viewport.Height = height
	}
	if scale := numericParamWithDefault(rawViewport, "scale", viewport.Scale); scale > 0 {
		viewport.Scale = scale
	}
	if mobile, ok := rawViewport["mobile"].(bool); ok {
		viewport.Mobile = mobile
	}
	return viewport
}

func findBrowserExecutable() (string, error) {
	var candidates []string
	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	case "linux":
		candidates = []string{
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/usr/bin/microsoft-edge",
		}
	case "windows":
		candidates = []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	for _, candidate := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "msedge", "microsoft-edge"} {
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, nil
		}
	}
	return "", errors.New("could not find Chrome, Chromium, or Edge on the bridge host")
}

func allocateBrowserDebugPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func waitForBrowserTarget(debugBase string, timeout time.Duration) (browserTargetInfo, error) {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		targets, err := loadBrowserTargets(client, debugBase)
		if err == nil {
			for _, target := range targets {
				if target.Type == "page" && strings.TrimSpace(target.WebSocketDebuggerURL) != "" {
					return target, nil
				}
			}
		}
		time.Sleep(browserSessionPollInterval)
	}
	return browserTargetInfo{}, errors.New("browser session did not expose a debuggable page in time")
}

func loadBrowserTargetByID(debugBase string, targetID string) (browserTargetInfo, error) {
	client := &http.Client{Timeout: 2 * time.Second}
	targets, err := loadBrowserTargets(client, debugBase)
	if err != nil {
		return browserTargetInfo{}, err
	}
	for _, target := range targets {
		if target.ID == targetID && target.Type == "page" && strings.TrimSpace(target.WebSocketDebuggerURL) != "" {
			return target, nil
		}
	}
	return browserTargetInfo{}, errors.New("browser target was not found")
}

func discoverBrowserDebugBase(preferred string) (string, error) {
	candidates := []string{}
	if normalized := normalizeBrowserDebugBase(preferred); normalized != "" {
		candidates = append(candidates, normalized)
	}
	for _, port := range browserAttachDefaultPorts {
		candidates = append(candidates, fmt.Sprintf("http://127.0.0.1:%d", port))
		candidates = append(candidates, fmt.Sprintf("http://localhost:%d", port))
	}

	client := &http.Client{Timeout: 1500 * time.Millisecond}
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		request, err := http.NewRequest(http.MethodGet, candidate+"/json/version", nil)
		if err != nil {
			continue
		}
		response, err := client.Do(request)
		if err != nil {
			continue
		}
		_ = response.Body.Close()
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return candidate, nil
		}
	}
	if preferred != "" {
		return "", errors.New("could not reach the requested Chrome remote debugging endpoint")
	}
	return "", errors.New("no attachable Chrome remote debugging session was found on localhost. Start Chrome with remote debugging first")
}

func normalizeBrowserDebugBase(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "ws://") {
		return "http://" + strings.TrimPrefix(trimmed, "ws://")
	}
	if strings.HasPrefix(trimmed, "wss://") {
		return "https://" + strings.TrimPrefix(trimmed, "wss://")
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return strings.TrimRight(trimmed, "/")
	}
	return "http://" + strings.TrimRight(trimmed, "/")
}

func loadBrowserTargets(client *http.Client, debugBase string) ([]browserTargetInfo, error) {
	request, err := http.NewRequest(http.MethodGet, debugBase+"/json/list", nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("browser target listing failed with HTTP %d", response.StatusCode)
	}

	var targets []browserTargetInfo
	if err := json.NewDecoder(response.Body).Decode(&targets); err != nil {
		return nil, err
	}
	return targets, nil
}

func browserConsoleLine(args []struct {
	Value       any    `json:"value"`
	Description string `json:"description"`
}) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		switch value := arg.Value.(type) {
		case string:
			parts = append(parts, value)
		case nil:
			if arg.Description != "" {
				parts = append(parts, arg.Description)
			}
		default:
			if encoded, err := json.Marshal(value); err == nil {
				parts = append(parts, string(encoded))
			} else if arg.Description != "" {
				parts = append(parts, arg.Description)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func appendLimitedLines(lines []string, next string, limit int) []string {
	normalized := strings.TrimSpace(next)
	if normalized == "" {
		return lines
	}
	lines = append(lines, normalized)
	if limit > 0 && len(lines) > limit {
		lines = append([]string(nil), lines[len(lines)-limit:]...)
	}
	return lines
}

func browserKeyPayloads(key string) []map[string]any {
	switch strings.TrimSpace(strings.ToLower(key)) {
	case "enter", "return":
		return keyEventTriplet("Enter", "\r", 13)
	case "tab":
		return keyEventTriplet("Tab", "\t", 9)
	case "escape":
		return keyEventTriplet("Escape", "", 27)
	case "backspace":
		return keyEventTriplet("Backspace", "", 8)
	case "arrowleft":
		return keyEventTriplet("ArrowLeft", "", 37)
	case "arrowright":
		return keyEventTriplet("ArrowRight", "", 39)
	case "arrowup":
		return keyEventTriplet("ArrowUp", "", 38)
	case "arrowdown":
		return keyEventTriplet("ArrowDown", "", 40)
	default:
		if len(key) == 1 {
			r := []rune(key)[0]
			return []map[string]any{
				{"type": "keyDown", "text": key, "unmodifiedText": key, "key": key, "windowsVirtualKeyCode": int(r), "nativeVirtualKeyCode": int(r)},
				{"type": "char", "text": key, "unmodifiedText": key, "key": key, "windowsVirtualKeyCode": int(r), "nativeVirtualKeyCode": int(r)},
				{"type": "keyUp", "key": key, "windowsVirtualKeyCode": int(r), "nativeVirtualKeyCode": int(r)},
			}
		}
	}
	return nil
}

func keyEventTriplet(key string, text string, code int) []map[string]any {
	events := []map[string]any{
		{"type": "keyDown", "key": key, "code": key, "windowsVirtualKeyCode": code, "nativeVirtualKeyCode": code},
	}
	if text != "" {
		events = append(events, map[string]any{
			"type":                  "char",
			"key":                   key,
			"code":                  key,
			"text":                  text,
			"unmodifiedText":        text,
			"windowsVirtualKeyCode": code,
			"nativeVirtualKeyCode":  code,
		})
	}
	events = append(events, map[string]any{
		"type":                  "keyUp",
		"key":                   key,
		"code":                  key,
		"windowsVirtualKeyCode": code,
		"nativeVirtualKeyCode":  code,
	})
	return events
}

func browserFocusEditableExpression(x, y float64) string {
	return fmt.Sprintf(`(() => {
  const x = %.4f;
  const y = %.4f;
  const selector = 'input:not([type="hidden"]):not([disabled]), textarea:not([disabled]), [contenteditable=""], [contenteditable="true"], [role="textbox"]';
  const isEditable = (el) => {
    if (!el || !(el instanceof HTMLElement)) return false;
    if (el instanceof HTMLInputElement) return el.type !== 'hidden' && el.disabled === false && el.readOnly === false;
    if (el instanceof HTMLTextAreaElement) return el.disabled === false && el.readOnly === false;
    return el.isContentEditable || el.getAttribute('role') === 'textbox';
  };
  const findEditable = (el) => {
    if (!el || !(el instanceof HTMLElement)) return null;
    if (isEditable(el)) return el;
    return el.closest?.(selector) || el.querySelector?.(selector) || null;
  };
  const selectionOffsetsForContentEditable = (target) => {
    const text = target.textContent ?? '';
    const fallback = { start: text.length, end: text.length };
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) return fallback;
    const range = selection.getRangeAt(0);
    if (!target.contains(range.startContainer) || !target.contains(range.endContainer)) return fallback;
    const offsetWithinTarget = (container, offset) => {
      const probe = document.createRange();
      probe.selectNodeContents(target);
      try {
        probe.setEnd(container, offset);
      } catch {
        return text.length;
      }
      return probe.toString().length;
    };
    return {
      start: offsetWithinTarget(range.startContainer, range.startOffset),
      end: offsetWithinTarget(range.endContainer, range.endOffset),
    };
  };
  const target = findEditable(document.activeElement) || findEditable(document.elementFromPoint(x, y));
  if (!target) return { ok: false, text: '', selectionStart: 0, selectionEnd: 0 };
  target.focus?.({ preventScroll: true });
  if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement) {
    const text = target.value ?? '';
    return {
      ok: true,
      text,
      selectionStart: typeof target.selectionStart === 'number' ? target.selectionStart : text.length,
      selectionEnd: typeof target.selectionEnd === 'number' ? target.selectionEnd : text.length,
    };
  }
  if (target.isContentEditable || target.getAttribute('role') === 'textbox') {
    const text = target.textContent ?? '';
    const selection = selectionOffsetsForContentEditable(target);
    return {
      ok: true,
      text,
      selectionStart: selection.start,
      selectionEnd: selection.end,
    };
  }
  return { ok: false, text: '', selectionStart: 0, selectionEnd: 0 };
})()`, x, y)
}

func browserActiveEditableStateExpression(delay time.Duration) string {
	delayMs := int(delay / time.Millisecond)
	if delayMs < 0 {
		delayMs = 0
	}
	return fmt.Sprintf(`(async () => {
  const delayMs = %d;
  if (delayMs > 0) {
    await new Promise((resolve) => setTimeout(resolve, delayMs));
  }
  const isEditable = (el) => {
    if (!el || !(el instanceof HTMLElement)) return false;
    if (el instanceof HTMLInputElement) return el.type !== 'hidden' && el.disabled === false && el.readOnly === false;
    if (el instanceof HTMLTextAreaElement) return el.disabled === false && el.readOnly === false;
    return el.isContentEditable || el.getAttribute('role') === 'textbox';
  };
  const selectionOffsetsForContentEditable = (target) => {
    const text = target.textContent ?? '';
    const fallback = { start: text.length, end: text.length };
    const selection = window.getSelection();
    if (!selection || selection.rangeCount === 0) return fallback;
    const range = selection.getRangeAt(0);
    if (!target.contains(range.startContainer) || !target.contains(range.endContainer)) return fallback;
    const offsetWithinTarget = (container, offset) => {
      const probe = document.createRange();
      probe.selectNodeContents(target);
      try {
        probe.setEnd(container, offset);
      } catch {
        return text.length;
      }
      return probe.toString().length;
    };
    return {
      start: offsetWithinTarget(range.startContainer, range.startOffset),
      end: offsetWithinTarget(range.endContainer, range.endOffset),
    };
  };
  const target = document.activeElement;
  if (!isEditable(target)) return { ok: false, text: '', selectionStart: 0, selectionEnd: 0 };
  if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement) {
    const text = target.value ?? '';
    return {
      ok: true,
      text,
      selectionStart: typeof target.selectionStart === 'number' ? target.selectionStart : text.length,
      selectionEnd: typeof target.selectionEnd === 'number' ? target.selectionEnd : text.length,
    };
  }
  if (target.isContentEditable || target.getAttribute('role') === 'textbox') {
    const text = target.textContent ?? '';
    const selection = selectionOffsetsForContentEditable(target);
    return {
      ok: true,
      text,
      selectionStart: selection.start,
      selectionEnd: selection.end,
    };
  }
  return { ok: false, text: '', selectionStart: 0, selectionEnd: 0 };
})()`, delayMs)
}

func browserInsertTextExpression(text string, x, y float64) (string, error) {
	encodedText, err := json.Marshal(text)
	if err != nil {
		return "", fmt.Errorf("failed to encode browser text: %w", err)
	}
	return fmt.Sprintf(`(() => {
  const text = %s;
  const x = %.4f;
  const y = %.4f;
  const selector = 'input:not([type="hidden"]):not([disabled]), textarea:not([disabled]), [contenteditable=""], [contenteditable="true"], [role="textbox"]';
  const isEditable = (el) => {
    if (!el || !(el instanceof HTMLElement)) return false;
    if (el instanceof HTMLInputElement) return el.type !== 'hidden' && el.disabled === false && el.readOnly === false;
    if (el instanceof HTMLTextAreaElement) return el.disabled === false && el.readOnly === false;
    return el.isContentEditable || el.getAttribute('role') === 'textbox';
  };
  const findEditable = (el) => {
    if (!el || !(el instanceof HTMLElement)) return null;
    if (isEditable(el)) return el;
    return el.closest?.(selector) || el.querySelector?.(selector) || null;
  };
  const target = findEditable(document.activeElement) || findEditable(document.elementFromPoint(x, y));
  if (!target) return { ok: false, reason: 'no-editable-target' };
  target.focus?.({ preventScroll: true });
  if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement) {
    const prototype = target instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    const descriptor = Object.getOwnPropertyDescriptor(prototype, 'value');
    const setter = descriptor?.set;
    const currentValue = target.value ?? '';
    const selectionStart = typeof target.selectionStart === 'number' ? target.selectionStart : currentValue.length;
    const selectionEnd = typeof target.selectionEnd === 'number' ? target.selectionEnd : selectionStart;
    const nextValue = currentValue.slice(0, selectionStart) + text + currentValue.slice(selectionEnd);
    if (setter) {
      setter.call(target, nextValue);
    } else {
      target.value = nextValue;
    }
    const nextCaret = selectionStart + text.length;
    target.setSelectionRange?.(nextCaret, nextCaret);
    target.dispatchEvent(new InputEvent('input', { bubbles: true, data: text, inputType: 'insertText' }));
    return { ok: true, mode: 'form-control' };
  }
  if (target.isContentEditable || target.getAttribute('role') === 'textbox') {
    const selection = window.getSelection();
    let range = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null;
    if (!range || !target.contains(range.commonAncestorContainer)) {
      range = document.createRange();
      range.selectNodeContents(target);
      range.collapse(false);
      selection?.removeAllRanges();
      selection?.addRange(range);
    }
    range.deleteContents();
    const node = document.createTextNode(text);
    range.insertNode(node);
    range.setStartAfter(node);
    range.collapse(true);
    selection?.removeAllRanges();
    selection?.addRange(range);
    target.dispatchEvent(new InputEvent('input', { bubbles: true, data: text, inputType: 'insertText' }));
    return { ok: true, mode: 'contenteditable' };
  }
  return { ok: false, reason: 'unsupported-editable-target' };
})()`, encodedText, x, y), nil
}

func browserSetEditableStateExpression(text string, selectionStart, selectionEnd int, x, y float64) (string, error) {
	encodedText, err := json.Marshal(text)
	if err != nil {
		return "", fmt.Errorf("failed to encode browser text state: %w", err)
	}
	return fmt.Sprintf(`(() => {
  const text = %s;
  const requestedSelectionStart = %d;
  const requestedSelectionEnd = %d;
  const x = %.4f;
  const y = %.4f;
  const selector = 'input:not([type="hidden"]):not([disabled]), textarea:not([disabled]), [contenteditable=""], [contenteditable="true"], [role="textbox"]';
  const isEditable = (el) => {
    if (!el || !(el instanceof HTMLElement)) return false;
    if (el instanceof HTMLInputElement) return el.type !== 'hidden' && el.disabled === false && el.readOnly === false;
    if (el instanceof HTMLTextAreaElement) return el.disabled === false && el.readOnly === false;
    return el.isContentEditable || el.getAttribute('role') === 'textbox';
  };
  const findEditable = (el) => {
    if (!el || !(el instanceof HTMLElement)) return null;
    if (isEditable(el)) return el;
    return el.closest?.(selector) || el.querySelector?.(selector) || null;
  };
  const clamp = (value, min, max) => Math.min(Math.max(value, min), max);
  const target = findEditable(document.activeElement) || findEditable(document.elementFromPoint(x, y));
  if (!target) return { ok: false, reason: 'no-editable-target' };
  target.focus?.({ preventScroll: true });
  const textLength = text.length;
  const selectionStart = clamp(requestedSelectionStart, 0, textLength);
  const selectionEnd = clamp(requestedSelectionEnd, selectionStart, textLength);
  if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement) {
    const prototype = target instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    const descriptor = Object.getOwnPropertyDescriptor(prototype, 'value');
    const setter = descriptor?.set;
    if (setter) {
      setter.call(target, text);
    } else {
      target.value = text;
    }
    target.setSelectionRange?.(selectionStart, selectionEnd);
    target.dispatchEvent(new InputEvent('input', { bubbles: true, data: text, inputType: 'insertReplacementText' }));
    return { ok: true };
  }
  if (target.isContentEditable || target.getAttribute('role') === 'textbox') {
    const node = document.createTextNode(text);
    target.replaceChildren(node);
    const selection = window.getSelection();
    const range = document.createRange();
    range.setStart(node, selectionStart);
    range.setEnd(node, selectionEnd);
    selection?.removeAllRanges();
    selection?.addRange(range);
    target.dispatchEvent(new InputEvent('input', { bubbles: true, data: text, inputType: 'insertReplacementText' }));
    return { ok: true };
  }
  return { ok: false, reason: 'unsupported-editable-target' };
})()`, encodedText, selectionStart, selectionEnd, x, y), nil
}

func browserKeyAffectsEditableState(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "enter", "backspace", "delete", "tab", "arrowleft", "arrowright", "arrowup", "arrowdown", "home", "end":
		return true
	default:
		return false
	}
}

func normalizedBrowserSessionURL(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "about:blank"
	}
	if strings.Contains(trimmed, "://") || strings.HasPrefix(trimmed, "about:") {
		return trimmed
	}
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "localhost") || strings.HasPrefix(lower, "127.0.0.1") || strings.HasPrefix(lower, "[::1]") {
		return "http://" + trimmed
	}
	if strings.Contains(trimmed, ":") {
		return "http://" + trimmed
	}
	return "https://" + trimmed
}

func numericParamWithDefault(params map[string]any, key string, fallback float64) float64 {
	if params == nil {
		return fallback
	}
	value, ok := params[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		parsed, err := typed.Float64()
		if err == nil {
			return parsed
		}
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func paramStringValue(params map[string]any, key string) string {
	value, _ := stringParam(params, key)
	return strings.TrimSpace(value)
}

func rawStringParamValue(params map[string]any, key string) string {
	value, _ := stringParam(params, key)
	return value
}

func clamp(value, minimum, maximum float64) float64 {
	return math.Min(math.Max(value, minimum), maximum)
}

func clampInt(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

func utf16Len(value string) int {
	return len(utf16.Encode([]rune(value)))
}

func cloneBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}
	return append([]byte(nil), value...)
}

func appendStreamSubscriberChannels(
	into []chan browserSessionStreamMessage,
	subscribers map[int64]chan browserSessionStreamMessage,
) []chan browserSessionStreamMessage {
	for _, ch := range subscribers {
		into = append(into, ch)
	}
	return into
}

func deliverBrowserSessionStreamMessage(ch chan browserSessionStreamMessage, message browserSessionStreamMessage) {
	select {
	case ch <- message:
		return
	default:
	}

	select {
	case <-ch:
	default:
	}

	select {
	case ch <- message:
	default:
	}
}

func replaceStringByUTF16Range(value string, start, end int, replacement string) string {
	runes := []rune(value)
	startIndex := utf16OffsetToRuneIndex(runes, start)
	endIndex := utf16OffsetToRuneIndex(runes, end)
	return string(runes[:startIndex]) + replacement + string(runes[endIndex:])
}

func utf16OffsetToRuneIndex(runes []rune, offset int) int {
	if offset <= 0 {
		return 0
	}
	units := 0
	for index, r := range runes {
		if units == offset {
			return index
		}
		if r > 0xFFFF {
			units += 2
		} else {
			units++
		}
		if units >= offset {
			return index + 1
		}
	}
	return len(runes)
}

func (b *Bridge) startBrowserSession(params map[string]any) map[string]any {
	if b.browserSessions == nil {
		return map[string]any{"error": "Browser sessions are unavailable on this bridge."}
	}
	result, err := b.browserSessions.Start(params)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (b *Bridge) listBrowserSessionTargets(params map[string]any) map[string]any {
	if b.browserSessions == nil {
		return map[string]any{
			"available": false,
			"targets":   []any{},
			"error":     "Browser sessions are unavailable on this bridge.",
		}
	}
	result, err := b.browserSessions.ListTargets(params)
	if err != nil {
		return map[string]any{
			"available": false,
			"targets":   []any{},
			"error":     err.Error(),
		}
	}
	return result
}

func (b *Bridge) browserSessionStatus(params map[string]any) map[string]any {
	session, err := b.browserSessions.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return session.statusMap()
}

func (b *Bridge) browserSessionSnapshot(params map[string]any) map[string]any {
	session, err := b.browserSessions.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	quality := int(numericParamWithDefault(params, "quality", 65))
	result, err := session.snapshotMap(quality)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (b *Bridge) browserSessionNextFrame(params map[string]any) map[string]any {
	session, err := b.browserSessions.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	afterRevision := int64(numericParamWithDefault(params, "afterRevision", float64(session.currentRevision())))
	timeoutMs := int(numericParamWithDefault(params, "timeoutMs", 8000))
	if timeoutMs < 0 {
		timeoutMs = 0
	}
	if err := session.waitForRevision(afterRevision, time.Duration(timeoutMs)*time.Millisecond); err != nil {
		return map[string]any{"error": err.Error()}
	}
	quality := int(numericParamWithDefault(params, "quality", 55))
	result, err := session.snapshotMap(quality)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (b *Bridge) navigateBrowserSession(params map[string]any) map[string]any {
	session, err := b.browserSessions.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	result, err := session.navigate(params)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (b *Bridge) browserSessionInput(params map[string]any) map[string]any {
	session, err := b.browserSessions.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	result, err := session.input(params)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (b *Bridge) browserSessionSyncEditable(params map[string]any) map[string]any {
	session, err := b.browserSessions.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	delayMs := int(numericParamWithDefault(params, "delayMs", 0))
	if delayMs < 0 {
		delayMs = 0
	}
	result, err := session.syncEditableState(time.Duration(delayMs) * time.Millisecond)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (b *Bridge) resizeBrowserSession(params map[string]any) map[string]any {
	session, err := b.browserSessions.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	result, err := session.resize(params)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (b *Bridge) stopBrowserSession(params map[string]any) map[string]any {
	sessionID := paramStringValue(params, "sessionId")
	session := b.browserSessions.Remove(sessionID)
	if session == nil {
		return map[string]any{"stopped": false, "error": "browser session was not found"}
	}
	session.stop()
	return map[string]any{"stopped": true}
}

func (b *Bridge) handleBrowserSessionStreamSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendText(w, http.StatusMethodNotAllowed, "Method Not Allowed\n")
		return
	}

	session, err := b.browserSessions.Session(strings.TrimSpace(r.URL.Query().Get("sessionId")))
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	conn, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	subscriberID, subscriber, initialStatus, initialFrame, err := session.subscribeStream()
	if err != nil {
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()),
			time.Now().Add(time.Second),
		)
		_ = conn.Close()
		return
	}
	defer session.unsubscribeStream(subscriberID)
	defer conn.Close()

	if err := writeBrowserSessionStreamStatus(conn, initialStatus); err != nil {
		return
	}
	if len(initialFrame) != 0 {
		if err := conn.WriteMessage(websocket.BinaryMessage, initialFrame); err != nil {
			return
		}
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	pingTicker := time.NewTicker(15 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case message, ok := <-subscriber:
			if !ok {
				return
			}
			var writeErr error
			switch message.Kind {
			case "frame":
				writeErr = conn.WriteMessage(websocket.BinaryMessage, message.Frame)
			case "status":
				writeErr = writeBrowserSessionStreamStatus(conn, message.Status)
			}
			if writeErr != nil {
				return
			}
		case <-pingTicker.C:
			if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(time.Second)); err != nil {
				return
			}
		case <-done:
			return
		case <-r.Context().Done():
			return
		}
	}
}

func writeBrowserSessionStreamStatus(conn *websocket.Conn, status map[string]any) error {
	return conn.WriteJSON(map[string]any{
		"type":    "status",
		"payload": status,
	})
}

func _browserSessionSnapshotBytesForTest(base64Text string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(base64Text)
}
