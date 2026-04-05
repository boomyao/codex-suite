package libp2p

import (
	"context"
	"fmt"
	"io"
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
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	// protocolHTTP is the libp2p protocol ID for HTTP proxying.
	// Remote peers open a stream with this protocol and send raw HTTP requests;
	// the bridge forwards them to the local gateway.
	protocolHTTP = "/codex-bridge/http/1.0.0"

	// protocolWS is the libp2p protocol ID for WebSocket proxying.
	protocolWS = "/codex-bridge/ws/1.0.0"
)

// Config holds the settings for the libp2p exposure provider.
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

	// ProxyListenPort is the local TCP port for the HTTP/WS proxy that
	// tunnels traffic from libp2p peers to the gateway. If 0 (default),
	// a random available port is chosen.
	ProxyListenPort int

	Logger *log.Logger
}

// Provider implements exposure.Provider using libp2p for decentralized
// peer-to-peer connectivity. It creates a libp2p node, joins the DHT for
// discovery, and opens two stream-based protocol handlers that proxy
// HTTP and WebSocket traffic to the local bridge gateway.
//
// For clients that don't have a native libp2p stack (e.g. mobile apps),
// the provider also starts a local TCP proxy server. Remote peers that
// connect via libp2p get their traffic tunneled through this local proxy
// to the gateway. The exposure session reports standard http:// and ws://
// URLs so existing clients work without changes.
type Provider struct {
	config Config
	logger *log.Logger

	mu      sync.Mutex
	host    host.Host
	dht     *dht.IpfsDHT
	session *exposure.Session
	target  *exposure.Target
	cancel  context.CancelFunc

	proxyListener net.Listener
	proxyServer   *http.Server
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

	// Register libp2p stream handlers that proxy to the local gateway
	h.SetStreamHandler(protocolHTTP, func(s network.Stream) {
		p.handleStream(s, target.GatewayHTTPURL)
	})
	h.SetStreamHandler(protocolWS, func(s network.Stream) {
		p.handleStream(s, target.GatewayWebSocketURL)
	})

	// Start a local TCP proxy so that existing HTTP/WS clients can reach the
	// gateway through the libp2p network without needing a native libp2p stack.
	proxyListener, proxyServer, proxyPort, err := p.startLocalProxy(target)
	if err != nil {
		kdht.Close()
		h.Close()
		cancel()
		return nil, fmt.Errorf("start local proxy: %w", err)
	}

	peerID := h.ID().String()
	addrs := h.Addrs()
	addrStrings := make([]string, len(addrs))
	for i, a := range addrs {
		addrStrings[i] = a.String()
	}

	p.logger.Printf("[libp2p] node started with peer ID: %s", peerID)
	p.logger.Printf("[libp2p] listening on: %s", strings.Join(addrStrings, ", "))
	p.logger.Printf("[libp2p] local proxy on port %d", proxyPort)

	// Build full multiaddrs including peer ID
	fullAddrs := make([]string, len(addrs))
	for i, a := range addrs {
		fullAddrs[i] = fmt.Sprintf("%s/p2p/%s", a.String(), peerID)
	}

	// Report standard HTTP/WS URLs that clients can use directly.
	// The proxy port forwards to the gateway and is reachable by any
	// peer that has network-level access (LAN, tailnet, or libp2p relay).
	session := &exposure.Session{
		ID:            peerID,
		Provider:      "libp2p",
		ReachableHTTP: fmt.Sprintf("http://0.0.0.0:%d", proxyPort),
		ReachableWS:   fmt.Sprintf("ws://0.0.0.0:%d", proxyPort),
		Status:        "ready",
	}

	p.mu.Lock()
	p.host = h
	p.dht = kdht
	p.session = session
	p.cancel = cancel
	p.proxyListener = proxyListener
	p.proxyServer = proxyServer
	p.mu.Unlock()

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

	if p.proxyServer != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		p.proxyServer.Shutdown(shutdownCtx)
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

// handleStream proxies a libp2p stream to a local gateway URL by opening
// a raw TCP connection and copying data bidirectionally.
func (p *Provider) handleStream(s network.Stream, targetURL string) {
	defer s.Close()

	parsed, err := url.Parse(targetURL)
	if err != nil {
		p.logger.Printf("[libp2p] invalid target URL %q: %v", targetURL, err)
		return
	}

	host := parsed.Hostname()
	port := parsed.Port()
	if port == "" {
		switch parsed.Scheme {
		case "http", "ws":
			port = "80"
		case "https", "wss":
			port = "443"
		}
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 10*time.Second)
	if err != nil {
		p.logger.Printf("[libp2p] dial gateway %s:%s failed: %v", host, port, err)
		return
	}
	defer conn.Close()

	done := make(chan struct{}, 2)
	go func() {
		io.Copy(conn, s)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(s, conn)
		done <- struct{}{}
	}()
	<-done
}

// startLocalProxy starts a local HTTP reverse proxy that forwards requests
// to the bridge gateway. This gives standard HTTP/WS clients a way to reach
// the bridge without needing a native libp2p implementation.
func (p *Provider) startLocalProxy(target exposure.Target) (net.Listener, *http.Server, int, error) {
	listenAddr := fmt.Sprintf("0.0.0.0:%d", p.config.ProxyListenPort)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("listen on %s: %w", listenAddr, err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	proxy, err := buildReverseProxy(target.GatewayHTTPURL)
	if err != nil {
		listener.Close()
		return nil, nil, 0, err
	}

	srv := &http.Server{Handler: proxy}
	go func() {
		if srvErr := srv.Serve(listener); srvErr != nil && srvErr != http.ErrServerClosed {
			p.logger.Printf("[libp2p] local proxy server error: %v", srvErr)
		}
	}()

	return listener, srv, port, nil
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
