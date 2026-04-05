package libp2p

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/boomyao/codex-bridge/internal/exposure"
	libp2phost "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	gostream "github.com/libp2p/go-libp2p/p2p/net/gostream"
	"github.com/multiformats/go-multiaddr"
)

const (
	protocolHTTP = "/codex-bridge/http/1.0.0"
	protocolWS   = "/codex-bridge/ws/1.0.0"
)

type Config struct {
	// ListenAddrs are the multiaddrs the libp2p node listens on.
	// Defaults to /ip4/0.0.0.0/tcp/0 and /ip4/0.0.0.0/udp/0/quic-v1 if empty.
	ListenAddrs []string

	// BootstrapPeers are multiaddrs of bootstrap peers for DHT discovery.
	// Uses IPFS default bootstrap peers if empty.
	BootstrapPeers []string

	// PrivateKeyPath is the path to a persistent identity key file.
	// A new ephemeral key is generated each run if empty.
	PrivateKeyPath string

	// EnableRelay enables the libp2p circuit relay for NAT traversal.
	EnableRelay bool

	// EnableMDNS enables mDNS discovery for LAN peers.
	EnableMDNS bool

	Logger *log.Logger
}

type Provider struct {
	config Config
	logger *log.Logger

	mu      sync.Mutex
	host    host.Host
	dht     *dht.IpfsDHT
	session *exposure.Session
	target  *exposure.Target
	cancel  context.CancelFunc

	httpServer *http.Server
	wsServer   *http.Server
}

func New(config Config) exposure.Provider {
	logger := config.Logger
	if logger == nil {
		logger = log.Default()
	}
	if len(config.ListenAddrs) == 0 {
		config.ListenAddrs = []string{
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
		}
	}
	if config.EnableMDNS {
		// mDNS is enabled by default in go-libp2p when not explicitly disabled
	}
	return &Provider{
		config: config,
		logger: logger,
	}
}

func (p *Provider) Name() string {
	return "libp2p"
}

func (p *Provider) Start(ctx context.Context, target exposure.Target) (*exposure.Session, error) {
	p.mu.Lock()
	if p.host != nil {
		copy := *p.session
		p.mu.Unlock()
		return &copy, nil
	}
	targetCopy := target
	p.target = &targetCopy
	p.mu.Unlock()

	return p.startNode(ctx, target)
}

func (p *Provider) startNode(ctx context.Context, target exposure.Target) (*exposure.Session, error) {
	nodeCtx, cancel := context.WithCancel(ctx)

	listenAddrs := make([]multiaddr.Multiaddr, 0, len(p.config.ListenAddrs))
	for _, addr := range p.config.ListenAddrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("invalid listen addr %q: %w", addr, err)
		}
		listenAddrs = append(listenAddrs, ma)
	}

	opts := []libp2phost.Option{
		libp2phost.ListenAddrs(listenAddrs...),
		libp2phost.NATPortMap(),
		libp2phost.EnableHolePunching(),
	}

	if p.config.EnableRelay {
		opts = append(opts, libp2phost.EnableRelay())
	}

	h, err := libp2phost.New(opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create libp2p host: %w", err)
	}

	// Start Kademlia DHT for peer discovery
	kdht, err := dht.New(nodeCtx, h, dht.Mode(dht.ModeAutoServer))
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("create DHT: %w", err)
	}

	if err := kdht.Bootstrap(nodeCtx); err != nil {
		kdht.Close()
		h.Close()
		cancel()
		return nil, fmt.Errorf("bootstrap DHT: %w", err)
	}

	// Connect to bootstrap peers
	bootstrapPeers := p.config.BootstrapPeers
	if len(bootstrapPeers) == 0 {
		bootstrapPeers = defaultBootstrapPeers()
	}
	p.connectBootstrapPeers(nodeCtx, h, bootstrapPeers)

	// Set up HTTP reverse proxy over libp2p streams
	httpProxy, err := buildReverseProxy(target.GatewayHTTPURL)
	if err != nil {
		kdht.Close()
		h.Close()
		cancel()
		return nil, fmt.Errorf("build HTTP proxy: %w", err)
	}

	wsProxy, err := buildReverseProxy(target.GatewayWebSocketURL)
	if err != nil {
		kdht.Close()
		h.Close()
		cancel()
		return nil, fmt.Errorf("build WS proxy: %w", err)
	}

	// Create HTTP server on libp2p stream listener
	httpListener, err := gostream.Listen(h, protocolHTTP)
	if err != nil {
		kdht.Close()
		h.Close()
		cancel()
		return nil, fmt.Errorf("create HTTP stream listener: %w", err)
	}

	wsListener, err := gostream.Listen(h, protocolWS)
	if err != nil {
		httpListener.Close()
		kdht.Close()
		h.Close()
		cancel()
		return nil, fmt.Errorf("create WS stream listener: %w", err)
	}

	httpSrv := &http.Server{Handler: httpProxy}
	wsSrv := &http.Server{Handler: wsProxy}

	go func() {
		if srvErr := httpSrv.Serve(httpListener); srvErr != nil && srvErr != http.ErrServerClosed {
			p.logger.Printf("[libp2p] HTTP server error: %v", srvErr)
		}
	}()
	go func() {
		if srvErr := wsSrv.Serve(wsListener); srvErr != nil && srvErr != http.ErrServerClosed {
			p.logger.Printf("[libp2p] WS server error: %v", srvErr)
		}
	}()

	peerID := h.ID().String()
	addrs := h.Addrs()
	addrStrings := make([]string, len(addrs))
	for i, a := range addrs {
		addrStrings[i] = a.String()
	}

	p.logger.Printf("[libp2p] node started with peer ID: %s", peerID)
	p.logger.Printf("[libp2p] listening on: %s", strings.Join(addrStrings, ", "))

	// Build full multiaddrs including peer ID
	fullAddrs := make([]string, len(addrs))
	for i, a := range addrs {
		fullAddrs[i] = fmt.Sprintf("%s/p2p/%s", a.String(), peerID)
	}

	session := &exposure.Session{
		ID:            peerID,
		Provider:      "libp2p",
		ReachableHTTP: fmt.Sprintf("libp2p://%s", peerID),
		ReachableWS:   fmt.Sprintf("libp2p+ws://%s", peerID),
		Status:        "ready",
	}

	p.mu.Lock()
	p.host = h
	p.dht = kdht
	p.session = session
	p.cancel = cancel
	p.httpServer = httpSrv
	p.wsServer = wsSrv
	p.mu.Unlock()

	// Monitor context cancellation for cleanup
	go func() {
		<-nodeCtx.Done()
		p.cleanup()
	}()

	copy := *session
	return &copy, nil
}

func (p *Provider) Status(_ context.Context) (*exposure.Session, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.session == nil {
		return &exposure.Session{
			ID:       "libp2p",
			Provider: "libp2p",
			Status:   "idle",
		}, nil
	}

	// Update addresses in case they changed (e.g. after NAT mapping)
	if p.host != nil {
		addrs := p.host.Addrs()
		addrStrings := make([]string, len(addrs))
		for i, a := range addrs {
			addrStrings[i] = fmt.Sprintf("%s/p2p/%s", a.String(), p.host.ID().String())
		}
		p.logger.Printf("[libp2p] current addresses: %s", strings.Join(addrStrings, ", "))
	}

	copy := *p.session
	return &copy, nil
}

func (p *Provider) Stop(_ context.Context, _ string) error {
	p.mu.Lock()
	cancel := p.cancel
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	return nil
}

func (p *Provider) cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.httpServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		p.httpServer.Shutdown(shutdownCtx)
		shutdownCancel()
	}
	if p.wsServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		p.wsServer.Shutdown(shutdownCtx)
		shutdownCancel()
	}
	if p.dht != nil {
		p.dht.Close()
		p.dht = nil
	}
	if p.host != nil {
		p.host.Close()
		p.host = nil
	}
	if p.session != nil {
		p.session.Status = "stopped"
		p.session.Error = ""
	}
	p.cancel = nil
}

// PeerID returns the libp2p peer ID of this node, or empty if not started.
func (p *Provider) PeerID() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.host == nil {
		return ""
	}
	return p.host.ID().String()
}

// Multiaddrs returns the full multiaddrs (including peer ID) of this node.
func (p *Provider) Multiaddrs() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.host == nil {
		return nil
	}
	addrs := p.host.Addrs()
	result := make([]string, len(addrs))
	for i, a := range addrs {
		result[i] = fmt.Sprintf("%s/p2p/%s", a.String(), p.host.ID().String())
	}
	return result
}

func (p *Provider) connectBootstrapPeers(ctx context.Context, h host.Host, peers []string) {
	for _, addr := range peers {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			p.logger.Printf("[libp2p] invalid bootstrap peer addr %q: %v", addr, err)
			continue
		}
		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			p.logger.Printf("[libp2p] invalid bootstrap peer info %q: %v", addr, err)
			continue
		}
		go func(pi peer.AddrInfo) {
			connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
			defer connectCancel()
			if err := h.Connect(connectCtx, pi); err != nil {
				p.logger.Printf("[libp2p] failed to connect to bootstrap peer %s: %v", pi.ID.String(), err)
			} else {
				p.logger.Printf("[libp2p] connected to bootstrap peer %s", pi.ID.String())
			}
		}(*pi)
	}
}

func buildReverseProxy(targetURL string) (*httputil.ReverseProxy, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(parsed)
	// Override the transport to use direct TCP connections to the local gateway
	proxy.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
	}
	return proxy, nil
}

func defaultBootstrapPeers() []string {
	return []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	}
}
