package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/boomyao/codex-bridge/internal/exposure"
)

type Config struct {
	SSHBinary      string
	SSHDestination string
	SSHPort        int
	RemotePort     int
	PublicHost     string
	PublicPort     int
	PublicScheme   string
	SSHArgs        []string
	AutoRestart    bool
	RestartDelay   time.Duration
	Logger         *log.Logger
}

type Provider struct {
	config Config
	logger *log.Logger

	mu      sync.Mutex
	cmd     *exec.Cmd
	session *exposure.Session
	target  *exposure.Target
	ctx     context.Context
	stopped bool
	timer   *time.Timer
}

func New(config Config) exposure.Provider {
	logger := config.Logger
	if logger == nil {
		logger = log.Default()
	}
	if strings.TrimSpace(config.SSHBinary) == "" {
		config.SSHBinary = "ssh"
	}
	if config.SSHPort <= 0 {
		config.SSHPort = 22
	}
	if strings.TrimSpace(config.PublicScheme) == "" {
		config.PublicScheme = "http"
	}
	if config.RestartDelay <= 0 {
		config.RestartDelay = 2 * time.Second
	}

	return &Provider{
		config: config,
		logger: logger,
	}
}

func (p *Provider) Name() string {
	return "tunnel"
}

func (p *Provider) Start(ctx context.Context, target exposure.Target) (*exposure.Session, error) {
	p.mu.Lock()
	if p.cmd != nil && p.cmd.Process != nil && p.session != nil {
		copy := *p.session
		p.mu.Unlock()
		return &copy, nil
	}
	p.ctx = ctx
	p.stopped = false
	targetCopy := target
	p.target = &targetCopy
	p.mu.Unlock()

	return p.startProcess(ctx, target)
}

func (p *Provider) startProcess(ctx context.Context, target exposure.Target) (*exposure.Session, error) {
	localHost, localPort, err := parseGatewayTarget(target.GatewayHTTPURL)
	if err != nil {
		return nil, err
	}

	publicHost := strings.TrimSpace(p.config.PublicHost)
	if publicHost == "" {
		publicHost = publicHostFromDestination(p.config.SSHDestination)
	}
	publicPort := p.config.PublicPort
	if publicPort <= 0 {
		publicPort = p.config.RemotePort
	}

	httpURL, wsURL := buildPublicURLs(p.config.PublicScheme, publicHost, publicPort)
	cmdArgs := append([]string{}, p.config.SSHArgs...)
	cmdArgs = append(
		cmdArgs,
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-o", "ServerAliveInterval=30",
		"-o", "ServerAliveCountMax=3",
		"-p", fmt.Sprintf("%d", p.config.SSHPort),
		"-R", fmt.Sprintf("%d:%s:%s", p.config.RemotePort, localHost, localPort),
		p.config.SSHDestination,
	)

	cmd := exec.CommandContext(ctx, p.config.SSHBinary, cmdArgs...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	session := &exposure.Session{
		ID:            "tunnel",
		Provider:      "tunnel",
		ReachableHTTP: httpURL,
		ReachableWS:   wsURL,
		Status:        "starting",
	}

	p.mu.Lock()
	p.cmd = cmd
	p.session = session
	p.mu.Unlock()

	go logStream(p.logger, "[tunnel]", stdout)
	go logStream(p.logger, "[tunnel]", stderr)

	exitCh := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		exitCh <- err
		p.handleExit(err)
	}()

	select {
	case err := <-exitCh:
		if err != nil {
			return nil, fmt.Errorf("tunnel process exited before becoming ready: %w", err)
		}
		return nil, fmt.Errorf("tunnel process exited unexpectedly")
	case <-time.After(1200 * time.Millisecond):
		p.mu.Lock()
		if p.session != nil {
			p.session.Status = "ready"
			p.session.Error = ""
			copy := *p.session
			p.mu.Unlock()
			return &copy, nil
		}
		p.mu.Unlock()
		return nil, fmt.Errorf("tunnel session became unavailable during startup")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *Provider) Status(context.Context) (*exposure.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.session == nil {
		return &exposure.Session{
			ID:       "tunnel",
			Provider: "tunnel",
			Status:   "idle",
		}, nil
	}
	copy := *p.session
	return &copy, nil
}

func (p *Provider) Stop(_ context.Context, _ string) error {
	p.mu.Lock()
	p.stopped = true
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
	cmd := p.cmd
	p.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

func (p *Provider) handleExit(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.cmd = nil
	if p.session == nil {
		return
	}

	if err == nil {
		p.session.Status = "stopped"
		p.session.Error = ""
		return
	}
	if strings.Contains(strings.ToLower(err.Error()), "killed") || strings.Contains(strings.ToLower(err.Error()), "signal: terminated") {
		p.session.Status = "stopped"
		p.session.Error = ""
		return
	}
	p.session.Status = "failed"
	p.session.Error = err.Error()
	if !p.config.AutoRestart || p.stopped || p.target == nil || p.ctx == nil {
		return
	}
	if p.timer != nil {
		p.timer.Stop()
	}
	restartCtx := p.ctx
	restartTarget := *p.target
	delay := p.config.RestartDelay
	p.logger.Printf("%s [tunnel] scheduling restart in %s", time.Now().UTC().Format(time.RFC3339), delay)
	p.timer = time.AfterFunc(delay, func() {
		if restartCtx.Err() != nil {
			return
		}
		if _, restartErr := p.Start(restartCtx, restartTarget); restartErr != nil {
			p.logger.Printf("%s [tunnel] restart failed: %v", time.Now().UTC().Format(time.RFC3339), restartErr)
		}
	})
}

func parseGatewayTarget(rawURL string) (string, string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", "", err
	}
	host := parsed.Hostname()
	port := parsed.Port()
	if strings.TrimSpace(host) == "" || strings.TrimSpace(port) == "" {
		return "", "", fmt.Errorf("invalid gateway URL %q", rawURL)
	}
	return host, port, nil
}

func publicHostFromDestination(destination string) string {
	trimmed := strings.TrimSpace(destination)
	if trimmed == "" {
		return ""
	}
	if index := strings.LastIndex(trimmed, "@"); index >= 0 {
		return trimmed[index+1:]
	}
	return trimmed
}

func buildPublicURLs(scheme, host string, port int) (string, string) {
	wsScheme := "ws"
	if scheme == "https" {
		wsScheme = "wss"
	}
	httpURL := fmt.Sprintf("%s://%s", scheme, host)
	wsURL := fmt.Sprintf("%s://%s", wsScheme, host)
	if port > 0 && !isDefaultPort(scheme, port) {
		httpURL = fmt.Sprintf("%s:%d", httpURL, port)
	}
	if port > 0 && !isDefaultPort(wsScheme, port) {
		wsURL = fmt.Sprintf("%s:%d", wsURL, port)
	}
	return httpURL, wsURL
}

func isDefaultPort(scheme string, port int) bool {
	return (scheme == "http" || scheme == "ws") && port == 80 || (scheme == "https" || scheme == "wss") && port == 443
}

func logStream(logger *log.Logger, prefix string, reader io.Reader) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			logger.Printf("%s %s %s", time.Now().UTC().Format(time.RFC3339), prefix, line)
		}
	}
}
