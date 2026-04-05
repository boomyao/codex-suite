package bridge

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

type desktopSessionManager struct {
	logger      *log.Logger
	port        int
	mu          sync.Mutex
	sessions    map[string]*desktopSession
	webrtcOnce  sync.Once
	webrtc      *desktopWebRTCService
	webrtcError error
}

type desktopSession struct {
	id     string
	logger *log.Logger

	mu sync.Mutex

	title               string
	width               int
	height              int
	scale               float64
	inputWidth          int
	inputHeight         int
	inputOriginX        int
	inputOriginY        int
	textInputActive     bool
	editableText        string
	editablePlaceholder string
	editableRole        string
	selectionStart      int
	selectionEnd        int

	video       *desktopVideoStreamer
	webrtcPeers map[string]*desktopWebRTCPeer

	closed    bool
	lastError string
}

func newDesktopSessionManager(logger *log.Logger, port int) *desktopSessionManager {
	return &desktopSessionManager{
		logger:   logger,
		port:     port,
		sessions: map[string]*desktopSession{},
	}
}

func (m *desktopSessionManager) Close(ctx context.Context) {
	m.mu.Lock()
	sessions := make([]*desktopSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	m.sessions = map[string]*desktopSession{}
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

func (m *desktopSessionManager) Start(_ map[string]any) (map[string]any, error) {
	if runtime.GOOS != "darwin" {
		return nil, errors.New("desktop sessions are only supported on macOS")
	}

	sessionID, err := randomMobileResourceID()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate desktop session: %w", err)
	}

	videoStreamer, err := startDesktopVideoStreamer(m.logger)
	if err != nil {
		return nil, err
	}

	session := &desktopSession{
		id:          sessionID,
		logger:      m.logger,
		title:       "Desktop",
		scale:       1,
		video:       videoStreamer,
		webrtcPeers: map[string]*desktopWebRTCPeer{},
	}

	readyCtx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	if err := videoStreamer.waitReady(readyCtx, 1500*time.Millisecond); err != nil {
		videoStreamer.stop()
		return nil, err
	}

	m.mu.Lock()
	m.sessions[sessionID] = session
	m.mu.Unlock()

	return session.statusMap(), nil
}

func (m *desktopSessionManager) Session(sessionID string) (*desktopSession, error) {
	trimmedID := strings.TrimSpace(sessionID)
	if trimmedID == "" {
		return nil, errors.New("desktop session id is required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	session := m.sessions[trimmedID]
	if session == nil {
		return nil, errors.New("desktop session was not found")
	}
	return session, nil
}

func (m *desktopSessionManager) Remove(sessionID string) *desktopSession {
	trimmedID := strings.TrimSpace(sessionID)
	if trimmedID == "" {
		return nil
	}
	m.mu.Lock()
	session := m.sessions[trimmedID]
	delete(m.sessions, trimmedID)
	m.mu.Unlock()
	return session
}

func (s *desktopSession) stop() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	video := s.video
	s.video = nil
	webrtcPeers := s.webrtcPeers
	s.webrtcPeers = map[string]*desktopWebRTCPeer{}
	s.mu.Unlock()

	if video != nil {
		video.stop()
	}
	for _, peer := range webrtcPeers {
		if peer != nil {
			peer.close()
		}
	}
}

func (s *desktopSession) syncVideoMetadataLocked() {
	if s.video == nil {
		return
	}
	_, _, _, width, height, scale, inputWidth, inputHeight, inputOriginX, inputOriginY := s.video.statusSnapshot()
	if width > 0 {
		s.width = width
	}
	if height > 0 {
		s.height = height
	}
	if scale > 0 {
		s.scale = scale
	}
	if s.scale <= 0 {
		s.scale = 1
	}
	if inputWidth > 0 {
		s.inputWidth = inputWidth
	} else if s.inputWidth <= 0 && s.width > 0 {
		s.inputWidth = maxInt(int(math.Round(float64(s.width)/s.scale)), 1)
	}
	if inputHeight > 0 {
		s.inputHeight = inputHeight
	} else if s.inputHeight <= 0 && s.height > 0 {
		s.inputHeight = maxInt(int(math.Round(float64(s.height)/s.scale)), 1)
	}
	s.inputOriginX = inputOriginX
	s.inputOriginY = inputOriginY
}

func (s *desktopSession) statusMap() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.statusMapLocked()
}

func (s *desktopSession) statusMapLocked() map[string]any {
	s.syncVideoMetadataLocked()
	s.syncEditableStateLocked()
	return s.currentStatusMapLocked()
}

func (s *desktopSession) currentStatusMapLocked() map[string]any {
	return map[string]any{
		"sessionId":           s.id,
		"title":               s.title,
		"streaming":           true,
		"width":               s.width,
		"height":              s.height,
		"scale":               s.scale,
		"active":              !s.closed,
		"lastError":           emptyToNil(s.lastError),
		"preferredTransport":  "webrtc",
		"videoCodec":          s.videoCodecLocked(),
		"videoReady":          s.videoReadyLocked(),
		"videoError":          emptyToNil(s.videoErrorLocked()),
		"textInputActive":     s.textInputActive,
		"editableText":        s.editableText,
		"editablePlaceholder": s.editablePlaceholder,
		"editableRole":        emptyToNil(s.editableRole),
		"selectionStart":      s.selectionStart,
		"selectionEnd":        s.selectionEnd,
	}
}

func (s *desktopSession) syncEditableStateLocked() {
	state, err := desktopFocusedEditableState()
	if err != nil {
		s.textInputActive = false
		s.editableText = ""
		s.editablePlaceholder = ""
		s.editableRole = ""
		s.selectionStart = 0
		s.selectionEnd = 0
		return
	}
	s.applyEditableStateLocked(state)
}

func (s *desktopSession) applyEditableStateLocked(state desktopEditableState) {
	s.textInputActive = state.Active
	if state.Active {
		s.editableText = state.Text
		s.editablePlaceholder = state.Placeholder
		s.editableRole = state.Role
		s.selectionStart = maxInt(state.SelectionStart, 0)
		s.selectionEnd = maxInt(state.SelectionEnd, s.selectionStart)
		return
	}
	s.editableText = ""
	s.editablePlaceholder = ""
	s.editableRole = ""
	s.selectionStart = 0
	s.selectionEnd = 0
}

func (s *desktopSession) input(params map[string]any) (map[string]any, error) {
	inputType := strings.TrimSpace(paramStringValue(params, "type"))
	var status map[string]any
	switch inputType {
	case "tap":
		x, y, err := s.screenPointFromNormalizedParams(params)
		if err != nil {
			return nil, err
		}
		if err := runDesktopCommand("cliclick", fmt.Sprintf("c:%d,%d", x, y)); err != nil {
			return nil, err
		}
	case "pointerDown":
		x, y, err := s.screenPointFromNormalizedParams(params)
		if err != nil {
			return nil, err
		}
		if err := runDesktopCommand("cliclick", fmt.Sprintf("dd:%d,%d", x, y)); err != nil {
			return nil, err
		}
	case "pointerMove":
		x, y, err := s.screenPointFromNormalizedParams(params)
		if err != nil {
			return nil, err
		}
		if err := runDesktopCommand("cliclick", fmt.Sprintf("dm:%d,%d", x, y)); err != nil {
			return nil, err
		}
	case "pointerUp":
		x, y, err := s.screenPointFromNormalizedParams(params)
		if err != nil {
			return nil, err
		}
		if err := runDesktopCommand("cliclick", fmt.Sprintf("du:%d,%d", x, y)); err != nil {
			return nil, err
		}
	case "text":
		text := rawStringParamValue(params, "text")
		if text == "" {
			return s.statusMap(), nil
		}
		if err := runDesktopTextCommand(text); err != nil {
			return nil, err
		}
	case "textState":
		text := rawStringParamValue(params, "text")
		defaultSelection := float64(utf16Len(text))
		selectionStart := int(numericParamWithDefault(params, "selectionStart", defaultSelection))
		selectionEnd := int(numericParamWithDefault(params, "selectionEnd", float64(selectionStart)))
		state, err := desktopSetFocusedTextState(text, selectionStart, selectionEnd)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.syncVideoMetadataLocked()
		s.applyEditableStateLocked(state)
		status = s.currentStatusMapLocked()
		s.mu.Unlock()
	case "key":
		key := strings.TrimSpace(paramStringValue(params, "key"))
		if key == "" {
			return nil, errors.New("desktop key input requires a key")
		}
		if err := runDesktopKeyCommand(key, desktopModifierParams(params)); err != nil {
			return nil, err
		}
	case "scroll":
		deltaY := numericParamWithDefault(params, "deltaY", 0)
		key := "page-down"
		if deltaY < 0 {
			key = "page-up"
		}
		steps := clampInt(int(math.Round(math.Abs(deltaY)/140.0)), 1, 6)
		for i := 0; i < steps; i++ {
			if err := runDesktopKeyCommand(key, nil); err != nil {
				return nil, err
			}
		}
	default:
		return nil, fmt.Errorf("unsupported desktop input type: %s", inputType)
	}

	if status != nil {
		return status, nil
	}
	return s.statusMap(), nil
}

func (s *desktopSession) videoCodecLocked() any {
	if s.video == nil {
		return nil
	}
	_, codec, _, _, _, _, _, _, _, _ := s.video.statusSnapshot()
	if codec == "" {
		return "h264"
	}
	return codec
}

func (s *desktopSession) videoReadyLocked() bool {
	if s.video == nil {
		return false
	}
	ready, _, _, _, _, _, _, _, _, _ := s.video.statusSnapshot()
	return ready
}

func (s *desktopSession) videoErrorLocked() string {
	if s.video == nil {
		return ""
	}
	_, _, errText, _, _, _, _, _, _, _ := s.video.statusSnapshot()
	return errText
}

func (s *desktopSession) subscribeVideoStream() (int64, <-chan desktopVideoEvent, map[string]any, []byte, error) {
	s.mu.Lock()
	video := s.video
	s.mu.Unlock()
	if video == nil {
		return 0, nil, nil, nil, errors.New("desktop video stream is unavailable")
	}
	return video.subscribe()
}

func (s *desktopSession) unsubscribeVideoStream(subscriberID int64) {
	s.mu.Lock()
	video := s.video
	s.mu.Unlock()
	if video != nil {
		video.unsubscribe(subscriberID)
	}
}

func runDesktopKeyCommand(key string, modifiers []string) error {
	normalizedKey, keyMode, err := normalizeDesktopKey(key)
	if err != nil {
		return err
	}
	normalizedModifiers, err := normalizeDesktopModifiers(modifiers)
	if err != nil {
		return err
	}

	if keyMode == "character" {
		return runDesktopAppleScriptKeystroke(normalizedKey, normalizedModifiers)
	}

	if keyCode, ok := desktopAppleScriptKeyCodes[normalizedKey]; ok {
		return runDesktopAppleScriptKeyCode(keyCode, normalizedModifiers)
	}

	if desktopCliclickSupportedKey(normalizedKey) {
		commands := make([]string, 0, 3)
		if len(normalizedModifiers) > 0 {
			commands = append(commands, "kd:"+strings.Join(normalizedModifiers, ","))
		}
		commands = append(commands, "kp:"+normalizedKey)
		if len(normalizedModifiers) > 0 {
			commands = append(commands, "ku:"+strings.Join(normalizedModifiers, ","))
		}
		return runDesktopCommand("cliclick", commands...)
	}

	return fmt.Errorf("unsupported desktop key: %s", key)
}

func runDesktopCommand(name string, args ...string) error {
	command := exec.Command(name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return fmt.Errorf("%s failed: %w", name, err)
		}
		return fmt.Errorf("%s failed: %w (%s)", name, err, text)
	}
	return nil
}

func runDesktopTextCommand(text string) error {
	if text == "" {
		return nil
	}

	var chunk strings.Builder
	flushChunk := func() error {
		if chunk.Len() == 0 {
			return nil
		}
		value := chunk.String()
		chunk.Reset()
		return runDesktopAppleScriptPasteText(value)
	}

	for _, r := range text {
		switch r {
		case '\n', '\r':
			if err := flushChunk(); err != nil {
				return err
			}
			if err := runDesktopKeyCommand("Enter", nil); err != nil {
				return err
			}
		case '\t':
			if err := flushChunk(); err != nil {
				return err
			}
			if err := runDesktopKeyCommand("Tab", nil); err != nil {
				return err
			}
		default:
			chunk.WriteRune(r)
		}
	}

	return flushChunk()
}

func runDesktopAppleScriptPasteText(text string) error {
	lines := []string{
		"set previousClipboard to the clipboard",
		fmt.Sprintf("set the clipboard to %s", appleScriptQuotedString(text)),
		`tell application "System Events"`,
		`keystroke "v" using {command down}`,
		"end tell",
		"delay 0.05",
		"set the clipboard to previousClipboard",
	}
	return runDesktopAppleScript(lines...)
}

func runDesktopAppleScriptKeystroke(key string, modifiers []string) error {
	lines := []string{
		`tell application "System Events"`,
		fmt.Sprintf("keystroke %s%s", appleScriptQuotedString(key), desktopAppleScriptUsingClause(modifiers)),
		"end tell",
	}
	return runDesktopAppleScript(lines...)
}

func runDesktopAppleScriptKeyCode(keyCode int, modifiers []string) error {
	lines := []string{
		`tell application "System Events"`,
		fmt.Sprintf("key code %d%s", keyCode, desktopAppleScriptUsingClause(modifiers)),
		"end tell",
	}
	return runDesktopAppleScript(lines...)
}

func runDesktopAppleScript(lines ...string) error {
	args := make([]string, 0, len(lines)*2)
	for _, line := range lines {
		args = append(args, "-e", line)
	}
	return runDesktopCommand("osascript", args...)
}

func normalizeDesktopKey(key string) (string, string, error) {
	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return "", "", errors.New("desktop key input requires a key")
	}

	switch trimmedKey {
	case " ":
		return "space", "named", nil
	}

	loweredKey := strings.ToLower(trimmedKey)
	if alias, ok := desktopKeyAliases[loweredKey]; ok {
		return alias, "named", nil
	}

	if len([]rune(trimmedKey)) == 1 {
		return trimmedKey, "character", nil
	}

	return "", "", fmt.Errorf("unsupported desktop key: %s", key)
}

func normalizeDesktopModifiers(modifiers []string) ([]string, error) {
	if len(modifiers) == 0 {
		return nil, nil
	}
	normalized := make([]string, 0, len(modifiers))
	for _, modifier := range modifiers {
		trimmedModifier := strings.ToLower(strings.TrimSpace(modifier))
		if trimmedModifier == "" {
			continue
		}
		normalizedModifier, ok := desktopModifierAliases[trimmedModifier]
		if !ok {
			return nil, fmt.Errorf("unsupported desktop modifier: %s", modifier)
		}
		if !slices.Contains(normalized, normalizedModifier) {
			normalized = append(normalized, normalizedModifier)
		}
	}
	slices.SortFunc(normalized, func(lhs, rhs string) int {
		return desktopModifierOrderIndex(lhs) - desktopModifierOrderIndex(rhs)
	})
	return normalized, nil
}

func desktopModifierParams(params map[string]any) []string {
	items := anySliceParam(params, "modifiers")
	if len(items) == 0 {
		return nil
	}
	modifiers := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if ok {
			modifiers = append(modifiers, text)
		}
	}
	return modifiers
}

func desktopModifierOrderIndex(value string) int {
	for index, modifier := range desktopModifierOrder {
		if modifier == value {
			return index
		}
	}
	return len(desktopModifierOrder)
}

func desktopAppleScriptUsingClause(modifiers []string) string {
	if len(modifiers) == 0 {
		return ""
	}
	parts := make([]string, 0, len(modifiers))
	for _, modifier := range modifiers {
		term, ok := desktopAppleScriptModifierTerms[modifier]
		if ok {
			parts = append(parts, term)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " using {" + strings.Join(parts, ", ") + "}"
}

func appleScriptQuotedString(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func desktopCliclickSupportedKey(key string) bool {
	_, ok := desktopCliclickNamedKeys[key]
	return ok
}

var desktopModifierAliases = map[string]string{
	"alt":      "alt",
	"option":   "alt",
	"cmd":      "cmd",
	"command":  "cmd",
	"control":  "ctrl",
	"ctrl":     "ctrl",
	"fn":       "fn",
	"function": "fn",
	"shift":    "shift",
}

var desktopModifierOrder = []string{"cmd", "shift", "alt", "ctrl", "fn"}

var desktopAppleScriptModifierTerms = map[string]string{
	"alt":   "option down",
	"cmd":   "command down",
	"ctrl":  "control down",
	"shift": "shift down",
}

var desktopKeyAliases = map[string]string{
	"arrowdown":      "arrow-down",
	"arrowleft":      "arrow-left",
	"arrowright":     "arrow-right",
	"arrowup":        "arrow-up",
	"backspace":      "delete",
	"delete":         "fwd-delete",
	"down":           "arrow-down",
	"end":            "end",
	"enter":          "return",
	"esc":            "esc",
	"escape":         "esc",
	"forwarddelete":  "fwd-delete",
	"forward-delete": "fwd-delete",
	"home":           "home",
	"left":           "arrow-left",
	"pagedown":       "page-down",
	"page-down":      "page-down",
	"pageup":         "page-up",
	"page-up":        "page-up",
	"return":         "return",
	"right":          "arrow-right",
	"space":          "space",
	"tab":            "tab",
	"up":             "arrow-up",
}

var desktopAppleScriptKeyCodes = map[string]int{
	"arrow-down":  125,
	"arrow-left":  123,
	"arrow-right": 124,
	"arrow-up":    126,
	"delete":      51,
	"end":         119,
	"esc":         53,
	"fwd-delete":  117,
	"home":        115,
	"page-down":   121,
	"page-up":     116,
	"return":      36,
	"space":       49,
	"tab":         48,
}

var desktopCliclickNamedKeys = map[string]struct{}{
	"arrow-down":  {},
	"arrow-left":  {},
	"arrow-right": {},
	"arrow-up":    {},
	"delete":      {},
	"end":         {},
	"enter":       {},
	"esc":         {},
	"fwd-delete":  {},
	"home":        {},
	"page-down":   {},
	"page-up":     {},
	"return":      {},
	"space":       {},
	"tab":         {},
}

func (s *desktopSession) screenPointFromNormalizedParams(params map[string]any) (int, int, error) {
	xNorm := clamp(numericParamWithDefault(params, "xNorm", 0.5), 0, 1)
	yNorm := clamp(numericParamWithDefault(params, "yNorm", 0.5), 0, 1)

	s.mu.Lock()
	s.syncVideoMetadataLocked()
	inputWidth := s.inputWidth
	inputHeight := s.inputHeight
	inputOriginX := s.inputOriginX
	inputOriginY := s.inputOriginY
	s.mu.Unlock()
	if inputWidth <= 0 || inputHeight <= 0 {
		return 0, 0, errors.New("desktop stream dimensions are not ready yet")
	}

	x := inputOriginX + int(math.Round(xNorm*float64(maxInt(inputWidth-1, 0))))
	y := inputOriginY + int(math.Round(yNorm*float64(maxInt(inputHeight-1, 0))))
	return x, y, nil
}

func maxInt(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a >= b {
		return a
	}
	return b
}

func (b *Bridge) startDesktopSession(params map[string]any) map[string]any {
	if b.desktopSessions == nil {
		return map[string]any{"error": "Desktop sessions are unavailable on this bridge."}
	}
	result, err := b.desktopSessions.Start(params)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (b *Bridge) desktopSessionStatus(params map[string]any) map[string]any {
	session, err := b.desktopSessions.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return session.statusMap()
}

func (b *Bridge) desktopSessionInput(params map[string]any) map[string]any {
	session, err := b.desktopSessions.Session(paramStringValue(params, "sessionId"))
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	result, err := session.input(params)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return result
}

func (b *Bridge) stopDesktopSession(params map[string]any) map[string]any {
	session := b.desktopSessions.Remove(paramStringValue(params, "sessionId"))
	if session == nil {
		return map[string]any{"stopped": false, "error": "desktop session was not found"}
	}
	session.stop()
	return map[string]any{"stopped": true}
}
