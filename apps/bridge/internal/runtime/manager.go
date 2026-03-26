package runtime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/boomyao/codex-bridge/internal/bridge"
)

type Command struct {
	Path string
	Args []string
}

type Info struct {
	BridgeHTTPURL         string
	BridgeWebSocketURL    string
	BridgeReadyURL        string
	AppServerWebSocketURL string
}

type ManagerConfig struct {
	Host                string
	BridgePort          int
	DesktopWebviewRoot  string
	UIPathPrefix        string
	AppServerPort       int
	ManageAppServer     bool
	ExternalUpstreamURL string
	AppServerCommand    Command
	AutoRestart         bool
	RestartDelay        time.Duration
	CWD                 string
	Logger              *log.Logger
}

type Manager struct {
	config        ManagerConfig
	logger        *log.Logger
	mu            sync.Mutex
	cancel        context.CancelFunc
	bridge        *bridge.Bridge
	appServerCmd  *exec.Cmd
	appServerDone chan error
	info          *Info
	restartTimer  *time.Timer
	stopping      bool
}

func New(config ManagerConfig) *Manager {
	logger := config.Logger
	if logger == nil {
		logger = log.Default()
	}
	if config.Host == "" {
		config.Host = "127.0.0.1"
	}
	if config.BridgePort == 0 {
		config.BridgePort = 8787
	}
	if config.UIPathPrefix == "" {
		config.UIPathPrefix = "/ui"
	}
	if config.AppServerPort == 0 {
		config.AppServerPort = 9876
	}
	if config.RestartDelay <= 0 {
		config.RestartDelay = 1500 * time.Millisecond
	}

	return &Manager{
		config: config,
		logger: logger,
	}
}

func (m *Manager) Start(ctx context.Context) (*Info, error) {
	m.mu.Lock()
	if m.info != nil {
		infoCopy := *m.info
		m.mu.Unlock()
		return &infoCopy, nil
	}
	startCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.stopping = false
	m.mu.Unlock()

	info, err := m.performStart(startCtx)
	if err != nil {
		cancel()
		return nil, err
	}

	m.mu.Lock()
	m.info = info
	m.mu.Unlock()
	return info, nil
}

func (m *Manager) performStart(ctx context.Context) (*Info, error) {
	host := m.config.Host
	bridgePort, err := findAvailablePort(host, m.config.BridgePort)
	if err != nil {
		return nil, err
	}

	upstreamURL := stringsTrimSpace(m.config.ExternalUpstreamURL)
	if m.config.ManageAppServer {
		appServerPort, err := findAvailablePort(host, m.config.AppServerPort)
		if err != nil {
			return nil, err
		}
		upstreamURL = fmt.Sprintf("ws://%s:%d", host, appServerPort)
		if err := m.startManagedAppServer(ctx, upstreamURL); err != nil {
			return nil, err
		}
	} else if upstreamURL == "" {
		return nil, errors.New("no upstream app-server URL configured")
	}

	bridgeServer := bridge.New(bridge.Config{
		Host:               host,
		Port:               bridgePort,
		UpstreamURL:        upstreamURL,
		HealthEnabled:      true,
		HealthPath:         "/healthz",
		ReadyPath:          "/readyz",
		ProbeTimeout:       2 * time.Second,
		ProbeCacheTTL:      5 * time.Second,
		DesktopWebviewRoot: m.config.DesktopWebviewRoot,
		UIPathPrefix:       m.config.UIPathPrefix,
	}, m.logger)

	bridgeInfo, err := bridgeServer.Start(ctx)
	if err != nil {
		if m.appServerCmd != nil {
			process, doneCh, stopErr := m.stopAppServerLocked()
			if stopErr == nil {
				_ = waitForStoppedProcess(context.Background(), process, doneCh)
			}
		}
		return nil, err
	}

	m.mu.Lock()
	m.bridge = bridgeServer
	m.mu.Unlock()

	info := &Info{
		BridgeHTTPURL:         bridgeInfo.BridgeHTTPURL,
		BridgeWebSocketURL:    bridgeInfo.BridgeWebSocketURL,
		BridgeReadyURL:        bridgeInfo.BridgeReadyURL,
		AppServerWebSocketURL: upstreamURL,
	}
	m.logger.Printf("%s [codex-bridge-runtime] bridge ready %s upstream %s", nowISO(), info.BridgeHTTPURL, upstreamURL)
	return info, nil
}

func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	m.stopping = true
	if m.restartTimer != nil {
		m.restartTimer.Stop()
		m.restartTimer = nil
	}
	cancel := m.cancel
	bridgeServer := m.bridge
	m.bridge = nil
	info := m.info
	m.info = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if bridgeServer != nil {
		_ = bridgeServer.Close(ctx)
	}
	if info != nil {
		m.logger.Printf("%s [codex-bridge-runtime] stopping runtime %s", nowISO(), info.BridgeHTTPURL)
	}
	return m.stopAppServer(ctx)
}

func (m *Manager) startManagedAppServer(ctx context.Context, upstreamURL string) error {
	command := m.config.AppServerCommand
	if command.Path == "" {
		command.Path = "codex"
	}
	args := append([]string{}, command.Args...)
	args = append(args, "--listen", upstreamURL)
	m.logger.Printf("%s [codex-bridge-runtime] starting app-server %s %s", nowISO(), command.Path, stringsJoin(args, " "))

	cmd := exec.CommandContext(ctx, command.Path, args...)
	cmd.Dir = m.config.CWD
	cmd.Env = append(os.Environ(), "CODEX_INTERNAL_ORIGINATOR_OVERRIDE=codex-bridge")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	m.mu.Lock()
	m.appServerCmd = cmd
	m.appServerDone = make(chan error, 1)
	m.mu.Unlock()

	go logStream(m.logger, "[app-server]", stdout)
	go logStream(m.logger, "[app-server]", stderr)
	go m.waitForAppServerExit(cmd)

	if err := waitForWebSocketReady(ctx, upstreamURL, 15*time.Second); err != nil {
		return fmt.Errorf("app-server exited before becoming ready: %w", err)
	}

	m.logger.Printf("%s [codex-bridge-runtime] app-server ready %s", nowISO(), upstreamURL)
	return nil
}

func (m *Manager) waitForAppServerExit(cmd *exec.Cmd) {
	err := cmd.Wait()

	m.mu.Lock()
	doneCh := m.appServerDone
	if doneCh != nil {
		doneCh <- err
		close(doneCh)
	}
	if m.appServerCmd == cmd {
		m.appServerCmd = nil
	}
	m.appServerDone = nil
	shouldRestart := !m.stopping && m.config.ManageAppServer && m.config.AutoRestart
	stopping := m.stopping
	m.mu.Unlock()

	if stopping {
		return
	}

	if err == nil || errors.Is(err, context.Canceled) {
		return
	}

	m.logger.Printf("%s [codex-bridge-runtime] runtime failed %v", nowISO(), err)
	if shouldRestart {
		m.scheduleRestart()
	}
}

func (m *Manager) scheduleRestart() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.restartTimer != nil || m.stopping {
		return
	}

	delay := m.config.RestartDelay
	m.logger.Printf("%s [codex-bridge-runtime] scheduling runtime restart %s", nowISO(), delay)
	m.restartTimer = time.AfterFunc(delay, func() {
		m.mu.Lock()
		m.restartTimer = nil
		m.mu.Unlock()

		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = m.Stop(stopCtx)

		startCtx := context.Background()
		if _, err := m.Start(startCtx); err != nil {
			m.logger.Printf("%s [codex-bridge-runtime] restart failed %v", nowISO(), err)
		}
	})
}

func (m *Manager) stopAppServer(ctx context.Context) error {
	m.mu.Lock()
	process, doneCh, err := m.stopAppServerLocked()
	m.mu.Unlock()
	if err != nil {
		return err
	}
	return waitForStoppedProcess(ctx, process, doneCh)
}

func (m *Manager) stopAppServerLocked() (*os.Process, chan error, error) {
	if m.appServerCmd == nil || m.appServerCmd.Process == nil {
		return nil, nil, nil
	}

	process := m.appServerCmd.Process
	m.appServerCmd = nil
	doneCh := m.appServerDone
	return process, doneCh, nil
}

func waitForStoppedProcess(ctx context.Context, process *os.Process, doneCh chan error) error {
	if process == nil {
		return nil
	}
	_ = process.Signal(syscall.SIGTERM)

	select {
	case err, ok := <-doneCh:
		if ok && err != nil && !errors.Is(err, os.ErrProcessDone) && !errors.Is(err, context.Canceled) {
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				return nil
			}
			return err
		}
		return nil
	case <-ctx.Done():
		_ = process.Kill()
		return ctx.Err()
	case <-time.After(5 * time.Second):
		_ = process.Kill()
		if doneCh != nil {
			<-doneCh
		}
		return nil
	}
}

func findAvailablePort(host string, preferred int) (int, error) {
	for port := preferred; port < preferred+200; port++ {
		address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
		listener, err := net.Listen("tcp", address)
		if err != nil {
			continue
		}
		_ = listener.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no available port found from %d", preferred)
}

func waitForWebSocketReady(ctx context.Context, rawURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		err := probeHTTPReady(ctx, rawURL)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(120 * time.Millisecond):
		}
	}
	return fmt.Errorf("timed out waiting for websocket %s", rawURL)
}

func probeHTTPReady(ctx context.Context, rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	scheme := "http"
	if parsed.Scheme == "wss" {
		scheme = "https"
	}
	readyURL := fmt.Sprintf("%s://%s/readyz", scheme, parsed.Host)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, readyURL, nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("readyz probe failed with status %d", response.StatusCode)
}

func logStream(logger *log.Logger, prefix string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := stringsTrimSpace(scanner.Text())
		if line != "" {
			logger.Printf("%s %s %s", nowISO(), prefix, line)
		}
	}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func stringsJoin(values []string, separator string) string {
	if len(values) == 0 {
		return ""
	}
	result := values[0]
	for _, value := range values[1:] {
		result += separator + value
	}
	return result
}

func stringsTrimSpace(value string) string {
	return strings.TrimSpace(value)
}
