package tailnet

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"sync"

	"github.com/boomyao/codex-bridge/internal/exposure"
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
)

type Config struct {
	Socket          string
	Hostname        string
	AddressStrategy string
	Logger          *log.Logger
}

type Provider struct {
	config Config
	logger *log.Logger

	mu      sync.Mutex
	target  *exposure.Target
	session *exposure.Session
}

func New(config Config) exposure.Provider {
	logger := config.Logger
	if logger == nil {
		logger = log.Default()
	}
	if strings.TrimSpace(config.AddressStrategy) == "" {
		config.AddressStrategy = "auto"
	}
	return &Provider{
		config: config,
		logger: logger,
	}
}

func (p *Provider) Name() string {
	return "tailnet"
}

func (p *Provider) Start(ctx context.Context, target exposure.Target) (*exposure.Session, error) {
	p.mu.Lock()
	targetCopy := target
	p.target = &targetCopy
	p.mu.Unlock()
	return p.refreshSession(ctx)
}

func (p *Provider) Status(ctx context.Context) (*exposure.Session, error) {
	p.mu.Lock()
	hasTarget := p.target != nil
	var current *exposure.Session
	if p.session != nil {
		copy := *p.session
		current = &copy
	}
	p.mu.Unlock()

	if !hasTarget {
		if current != nil {
			return current, nil
		}
		return &exposure.Session{
			ID:       "tailnet",
			Provider: "tailnet",
			Status:   "idle",
		}, nil
	}
	return p.refreshSession(ctx)
}

func (p *Provider) Stop(context.Context, string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.session != nil {
		p.session.Status = "stopped"
		p.session.Error = ""
	}
	return nil
}

func (p *Provider) refreshSession(ctx context.Context) (*exposure.Session, error) {
	p.mu.Lock()
	if p.target == nil {
		p.mu.Unlock()
		return &exposure.Session{
			ID:       "tailnet",
			Provider: "tailnet",
			Status:   "idle",
		}, nil
	}
	target := *p.target
	p.mu.Unlock()

	session, err := p.buildSession(ctx, target)
	p.mu.Lock()
	p.session = session
	copy := *session
	p.mu.Unlock()
	if err != nil {
		return &copy, err
	}
	return &copy, nil
}

func (p *Provider) buildSession(ctx context.Context, target exposure.Target) (*exposure.Session, error) {
	httpPort, err := gatewayPort(target.GatewayHTTPURL)
	if err != nil {
		return failedSession(fmt.Errorf("parse gateway HTTP URL: %w", err)), err
	}
	wsPort, err := gatewayPort(target.GatewayWebSocketURL)
	if err != nil {
		return failedSession(fmt.Errorf("parse gateway WebSocket URL: %w", err)), err
	}
	host, err := gatewayHost(target.GatewayHTTPURL)
	if err != nil {
		return failedSession(fmt.Errorf("parse gateway host: %w", err)), err
	}
	if isLoopbackGateway(host) {
		err := fmt.Errorf("tailnet exposure requires gateway.host to be non-loopback, got %q", host)
		return failedSession(err), err
	}

	status, err := p.readStatus(ctx)
	if err != nil {
		return failedSession(fmt.Errorf("read tailscale status: %w", err)), err
	}
	reachableHost, err := p.reachableHost(status)
	if err != nil {
		return failedSession(err), err
	}

	return &exposure.Session{
		ID:            "tailnet",
		Provider:      "tailnet",
		ReachableHTTP: buildURL("http", reachableHost, httpPort),
		ReachableWS:   buildURL("ws", reachableHost, wsPort),
		Status:        "ready",
	}, nil
}

func (p *Provider) readStatus(ctx context.Context) (*ipnstate.Status, error) {
	client := &local.Client{}
	if socket := strings.TrimSpace(p.config.Socket); socket != "" {
		client.Socket = socket
		client.UseSocketOnly = true
	}
	return client.StatusWithoutPeers(ctx)
}

func (p *Provider) reachableHost(status *ipnstate.Status) (string, error) {
	if host := normalizeDNSName(p.config.Hostname); host != "" {
		return host, nil
	}
	if status == nil || status.Self == nil {
		return "", fmt.Errorf("tailscale is not ready")
	}

	strategy := strings.TrimSpace(strings.ToLower(p.config.AddressStrategy))
	dnsName := normalizeDNSName(status.Self.DNSName)
	switch strategy {
	case "dns":
		if dnsName == "" {
			return "", fmt.Errorf("tailscale did not report a MagicDNS hostname")
		}
		return dnsName, nil
	case "ipv4":
		if ip, ok := preferredIP(status.Self.TailscaleIPs, true); ok {
			return ip.String(), nil
		}
		return "", fmt.Errorf("tailscale did not report an IPv4 address")
	case "ipv6":
		if ip, ok := preferredIP(status.Self.TailscaleIPs, false); ok {
			return ip.String(), nil
		}
		return "", fmt.Errorf("tailscale did not report an IPv6 address")
	case "auto", "":
		if magicDNSEnabled(status) && dnsName != "" {
			return dnsName, nil
		}
		if ip, ok := preferredIP(status.Self.TailscaleIPs, true); ok {
			return ip.String(), nil
		}
		if ip, ok := preferredIP(status.Self.TailscaleIPs, false); ok {
			return ip.String(), nil
		}
		return "", fmt.Errorf("tailscale did not report a usable address")
	default:
		return "", fmt.Errorf("unsupported tailnet address strategy %q", strategy)
	}
}

func failedSession(err error) *exposure.Session {
	return &exposure.Session{
		ID:       "tailnet",
		Provider: "tailnet",
		Status:   "failed",
		Error:    strings.TrimSpace(err.Error()),
	}
}

func gatewayPort(rawURL string) (int, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return 0, err
	}
	portText := strings.TrimSpace(parsed.Port())
	if portText == "" {
		switch parsed.Scheme {
		case "http", "ws":
			return 80, nil
		case "https", "wss":
			return 443, nil
		default:
			return 0, fmt.Errorf("missing port for scheme %q", parsed.Scheme)
		}
	}
	port, err := net.LookupPort("tcp", portText)
	if err != nil {
		return 0, err
	}
	return port, nil
}

func gatewayHost(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", fmt.Errorf("missing host")
	}
	return host, nil
}

func isLoopbackGateway(host string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" || host == "localhost" {
		return true
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	return ip.IsLoopback()
}

func buildURL(scheme, host string, port int) string {
	hostPort := host
	if port > 0 && !isDefaultPort(scheme, port) {
		hostPort = net.JoinHostPort(host, fmt.Sprintf("%d", port))
	}
	return (&url.URL{Scheme: scheme, Host: hostPort}).String()
}

func isDefaultPort(scheme string, port int) bool {
	return (scheme == "http" || scheme == "ws") && port == 80 || (scheme == "https" || scheme == "wss") && port == 443
}

func normalizeDNSName(name string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
}

func magicDNSEnabled(status *ipnstate.Status) bool {
	return status != nil && status.CurrentTailnet != nil && status.CurrentTailnet.MagicDNSEnabled
}

func preferredIP(addrs []netip.Addr, preferIPv4 bool) (netip.Addr, bool) {
	var fallback netip.Addr
	for _, addr := range addrs {
		if !addr.IsValid() {
			continue
		}
		if preferIPv4 && addr.Is4() {
			return addr, true
		}
		if !preferIPv4 && addr.Is6() {
			return addr, true
		}
		if !fallback.IsValid() {
			fallback = addr
		}
	}
	return fallback, fallback.IsValid()
}
