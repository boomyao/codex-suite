package bridge

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type desktopVideoStreamer struct {
	logger *log.Logger
	cmd    *exec.Cmd

	mu                 sync.Mutex
	closed             bool
	ready              bool
	lastError          string
	latestWidth        int
	latestHeight       int
	latestScale        float64
	latestInputWidth   int
	latestInputHeight  int
	latestInputOriginX int
	latestInputOriginY int
	latestFormat       map[string]any
	latestSample       []byte
	nextSubscriberID   int64
	subscribers        map[int64]chan desktopVideoEvent

	doneCh chan struct{}
}

type desktopVideoEvent struct {
	Kind   string
	JSON   map[string]any
	Sample []byte
}

const (
	desktopVideoPacketTypeJSON   byte = 1
	desktopVideoPacketTypeSample byte = 2
)

var desktopVideoHelperBuildMu sync.Mutex

func startDesktopVideoStreamer(logger *log.Logger) (*desktopVideoStreamer, error) {
	binaryPath, err := ensureDesktopVideoHelperBinary()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(binaryPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open desktop video helper stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to open desktop video helper stderr: %w", err)
	}

	streamer := &desktopVideoStreamer{
		logger:      logger,
		cmd:         cmd,
		subscribers: map[int64]chan desktopVideoEvent{},
		doneCh:      make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start desktop video helper: %w", err)
	}

	go streamer.readStdoutLoop(stdout)
	go streamer.readStderrLoop(stderr)
	go streamer.waitLoop()
	return streamer, nil
}

func (s *desktopVideoStreamer) stop() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	subscribers := s.subscribers
	s.subscribers = map[int64]chan desktopVideoEvent{}
	cmd := s.cmd
	s.mu.Unlock()

	for _, ch := range subscribers {
		close(ch)
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	select {
	case <-s.doneCh:
	case <-time.After(2 * time.Second):
	}
}

func (s *desktopVideoStreamer) subscribe() (int64, <-chan desktopVideoEvent, map[string]any, []byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		errMessage := strings.TrimSpace(s.lastError)
		if errMessage == "" {
			errMessage = "desktop video stream closed"
		}
		return 0, nil, nil, nil, errors.New(errMessage)
	}

	s.nextSubscriberID++
	subscriberID := s.nextSubscriberID
	ch := make(chan desktopVideoEvent, 3)
	s.subscribers[subscriberID] = ch
	return subscriberID, ch, cloneJSONMap(s.latestFormat), cloneDesktopVideoBytes(s.latestSample), nil
}

func (s *desktopVideoStreamer) unsubscribe(subscriberID int64) {
	s.mu.Lock()
	ch := s.subscribers[subscriberID]
	delete(s.subscribers, subscriberID)
	s.mu.Unlock()
	if ch != nil {
		close(ch)
	}
}

func (s *desktopVideoStreamer) statusSnapshot() (ready bool, codec string, videoError string, width int, height int, scale float64, inputWidth int, inputHeight int, inputOriginX int, inputOriginY int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ready || s.latestFormat != nil {
		ready = true
	}
	if ready {
		codec = "h264"
	}
	videoError = strings.TrimSpace(s.lastError)
	width = s.latestWidth
	height = s.latestHeight
	scale = s.latestScale
	inputWidth = s.latestInputWidth
	inputHeight = s.latestInputHeight
	inputOriginX = s.latestInputOriginX
	inputOriginY = s.latestInputOriginY
	return ready, codec, videoError, width, height, scale, inputWidth, inputHeight, inputOriginX, inputOriginY
}

func (s *desktopVideoStreamer) readStdoutLoop(stdout io.ReadCloser) {
	reader := bufio.NewReader(stdout)
	header := make([]byte, 5)
	for {
		if _, err := io.ReadFull(reader, header); err != nil {
			if !errors.Is(err, io.EOF) {
				s.recordError(fmt.Sprintf("desktop video helper stdout failed: %v", err))
			}
			return
		}
		payloadLength := binary.BigEndian.Uint32(header[1:5])
		payload := make([]byte, payloadLength)
		if payloadLength != 0 {
			if _, err := io.ReadFull(reader, payload); err != nil {
				s.recordError(fmt.Sprintf("desktop video helper stdout payload failed: %v", err))
				return
			}
		}
		s.handleEventPacket(header[0], payload)
	}
}

func (s *desktopVideoStreamer) readStderrLoop(stderr io.ReadCloser) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		s.logger.Printf("desktop video helper: %s", text)
	}
	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		s.logger.Printf("desktop video helper stderr failed: %v", err)
	}
}

func (s *desktopVideoStreamer) waitLoop() {
	defer close(s.doneCh)
	err := s.cmd.Wait()
	s.mu.Lock()
	alreadyClosed := s.closed
	if !s.closed {
		s.closed = true
	}
	if err != nil && strings.TrimSpace(s.lastError) == "" {
		s.lastError = err.Error()
	}
	subscribers := s.subscribers
	s.subscribers = map[int64]chan desktopVideoEvent{}
	s.mu.Unlock()

	if err != nil && !alreadyClosed {
		s.logger.Printf("desktop video helper exited: %v", err)
	}
	for _, ch := range subscribers {
		close(ch)
	}
}

func (s *desktopVideoStreamer) handleEventPacket(packetType byte, payload []byte) {
	switch packetType {
	case desktopVideoPacketTypeJSON:
		s.handleJSONEventPacket(payload)
	case desktopVideoPacketTypeSample:
		s.handleSampleEventPacket(payload)
	default:
		s.recordError(fmt.Sprintf("desktop video helper emitted an unknown packet type: %d", packetType))
	}
}

func (s *desktopVideoStreamer) handleJSONEventPacket(payload []byte) {
	var event map[string]any
	if err := json.Unmarshal(payload, &event); err != nil {
		s.recordError(fmt.Sprintf("desktop video helper emitted invalid JSON: %v", err))
		return
	}

	eventType := strings.TrimSpace(stringValueFromJSON(event, "type"))
	var subscribers []chan desktopVideoEvent
	var outbound desktopVideoEvent

	s.mu.Lock()
	switch eventType {
	case "ready":
		s.ready = true
		s.latestWidth = maxInt(int(float64Value(event["width"])), 0)
		s.latestHeight = maxInt(int(float64Value(event["height"])), 0)
		s.latestScale = maxFloat(float64Value(event["scale"]), 1)
		s.latestInputWidth = maxInt(int(float64Value(event["inputWidth"])), 0)
		s.latestInputHeight = maxInt(int(float64Value(event["inputHeight"])), 0)
		s.latestInputOriginX = int(float64Value(event["inputOriginX"]))
		s.latestInputOriginY = int(float64Value(event["inputOriginY"]))
	case "format":
		s.ready = true
		s.latestFormat = cloneJSONMap(event)
		payload, _ := event["payload"].(map[string]any)
		s.latestWidth = maxInt(int(float64Value(payload["width"])), s.latestWidth)
		s.latestHeight = maxInt(int(float64Value(payload["height"])), s.latestHeight)
		s.latestScale = maxFloat(float64Value(payload["scale"]), s.latestScale)
		s.latestInputWidth = maxInt(int(float64Value(payload["inputWidth"])), s.latestInputWidth)
		s.latestInputHeight = maxInt(int(float64Value(payload["inputHeight"])), s.latestInputHeight)
		if originX, ok := payload["inputOriginX"]; ok {
			s.latestInputOriginX = int(float64Value(originX))
		}
		if originY, ok := payload["inputOriginY"]; ok {
			s.latestInputOriginY = int(float64Value(originY))
		}
		outbound = desktopVideoEvent{
			Kind: "json",
			JSON: cloneJSONMap(s.latestFormat),
		}
	case "sample":
		s.mu.Unlock()
		s.recordError("desktop video helper emitted an unexpected JSON sample event")
		return
	case "error":
		s.lastError = strings.TrimSpace(stringValueFromJSON(event, "message"))
	}
	if outbound.Kind != "" {
		subscribers = appendDesktopVideoSubscriberChannels(nil, s.subscribers)
	}
	s.mu.Unlock()

	if outbound.Kind != "" {
		for _, ch := range subscribers {
			deliverDesktopVideoEvent(ch, outbound)
		}
	}
}

func (s *desktopVideoStreamer) handleSampleEventPacket(payload []byte) {
	if len(payload) == 0 {
		return
	}
	sample := cloneDesktopVideoBytes(payload)
	var subscribers []chan desktopVideoEvent

	s.mu.Lock()
	s.latestSample = cloneDesktopVideoBytes(sample)
	subscribers = appendDesktopVideoSubscriberChannels(nil, s.subscribers)
	s.mu.Unlock()

	outbound := desktopVideoEvent{
		Kind:   "sample",
		Sample: sample,
	}
	for _, ch := range subscribers {
		deliverDesktopVideoEvent(ch, outbound)
	}
}

func (s *desktopVideoStreamer) recordError(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	s.mu.Lock()
	s.lastError = message
	s.mu.Unlock()
	s.logger.Printf("%s", message)
}

func appendDesktopVideoSubscriberChannels(dst []chan desktopVideoEvent, src map[int64]chan desktopVideoEvent) []chan desktopVideoEvent {
	for _, ch := range src {
		if ch != nil {
			dst = append(dst, ch)
		}
	}
	return dst
}

func deliverDesktopVideoEvent(ch chan desktopVideoEvent, event desktopVideoEvent) {
	if ch == nil || event.Kind == "" {
		return
	}
	clonedEvent := cloneDesktopVideoEvent(event)
	select {
	case ch <- clonedEvent:
	default:
		select {
		case <-ch:
		default:
		}
		select {
		case ch <- clonedEvent:
		default:
		}
	}
}

func cloneDesktopVideoEvent(src desktopVideoEvent) desktopVideoEvent {
	return desktopVideoEvent{
		Kind:   src.Kind,
		JSON:   cloneJSONMap(src.JSON),
		Sample: cloneDesktopVideoBytes(src.Sample),
	}
}

func cloneDesktopVideoBytes(src []byte) []byte {
	if len(src) == 0 {
		return nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func cloneJSONMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	encoded, err := json.Marshal(src)
	if err != nil {
		return nil
	}
	var dst map[string]any
	if err := json.Unmarshal(encoded, &dst); err != nil {
		return nil
	}
	return dst
}

func stringValueFromJSON(payload map[string]any, key string) string {
	if payload == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", payload[key]))
}

func ensureDesktopVideoHelperBinary() (string, error) {
	desktopVideoHelperBuildMu.Lock()
	defer desktopVideoHelperBuildMu.Unlock()

	sourcePath, err := desktopVideoHelperSourcePath()
	if err != nil {
		return "", err
	}
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return "", fmt.Errorf("desktop video helper source is unavailable: %w", err)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to resolve the user cache directory: %w", err)
	}
	binaryPath := filepath.Join(cacheDir, "codex-bridge", "desktop_video_streamer")
	if binaryInfo, err := os.Stat(binaryPath); err == nil &&
		binaryInfo.ModTime().After(sourceInfo.ModTime()) {
		return binaryPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to prepare the desktop video helper cache directory: %w", err)
	}

	tmpPath := binaryPath + ".tmp"
	_ = os.Remove(tmpPath)
	command := exec.Command(
		"swiftc",
		sourcePath,
		"-O",
		"-framework", "ScreenCaptureKit",
		"-framework", "VideoToolbox",
		"-framework", "CoreMedia",
		"-framework", "CoreVideo",
		"-framework", "CoreGraphics",
		"-o", tmpPath,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to compile the desktop video helper: %w (%s)", err, strings.TrimSpace(string(output)))
	}
	if err := os.Rename(tmpPath, binaryPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to finalize the desktop video helper binary: %w", err)
	}
	return binaryPath, nil
}

func desktopVideoHelperSourcePath() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to locate the desktop video helper source")
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(currentFile)))
	sourcePath := filepath.Join(root, "helpers", "desktop_video_streamer.swift")
	if _, err := os.Stat(sourcePath); err != nil {
		return "", err
	}
	return sourcePath, nil
}

func (s *desktopVideoStreamer) waitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		s.mu.Lock()
		ready := s.ready || s.latestFormat != nil
		lastError := strings.TrimSpace(s.lastError)
		closed := s.closed
		s.mu.Unlock()

		if ready {
			return nil
		}
		if lastError != "" {
			return errors.New(lastError)
		}
		if closed {
			return errors.New("desktop video helper closed before producing a format description")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return nil
		case <-ticker.C:
		}
	}
}
