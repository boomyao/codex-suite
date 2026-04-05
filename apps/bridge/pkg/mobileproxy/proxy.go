// Package mobileproxy provides a gomobile-compatible API for mobile clients
// to connect to a codex-bridge peer over the libp2p network.
//
// Mobile apps (Android/iOS) call StartProxy with the bridge's peer ID and
// optional bootstrap peers. The proxy creates a local libp2p node, connects
// to the bridge peer via DHT discovery / relay / hole punching, and opens
// a local HTTP listener on 127.0.0.1. The mobile app then connects its
// OkHttp / URLSession to this local address, just like a normal HTTP bridge.
//
// # Usage from Kotlin (Android)
//
//	val proxy = Mobileproxy.startProxy(peerID, bootstrapPeers)
//	val localBaseURL = proxy.httpBaseURL()   // e.g. "http://127.0.0.1:54321"
//	// use localBaseURL with existing BridgeApi / WebSocket code
//	proxy.stop()
//
// # Usage from Swift (iOS)
//
//	let proxy = MobileproxyStartProxy(peerID, bootstrapPeers)
//	let baseURL = proxy.httpBaseURL()
//	// use baseURL with existing BridgeAPI / URLSession code
//	proxy.stop()
//
// Build with gomobile:
//
//	gomobile bind -target=android -o mobileproxy.aar ./pkg/mobileproxy
//	gomobile bind -target=ios -o Mobileproxy.xcframework ./pkg/mobileproxy
package mobileproxy

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	libp2phost "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

const (
	protocolHTTP = "/codex-bridge/http/1.0.0"
	protocolWS   = "/codex-bridge/ws/1.0.0"
)

// Proxy holds a running libp2p-to-HTTP proxy. It creates a libp2p node,
// connects to the bridge peer, and starts a local HTTP server that tunnels
// all requests through libp2p streams to the bridge.
type Proxy struct {
	mu       sync.Mutex
	host     host.Host
	dht      *dht.IpfsDHT
	peerID   peer.ID
	listener net.Listener
	server   *http.Server
	cancel   context.CancelFunc
	port     int
}

// StartProxy creates a libp2p node, discovers and connects to the bridge
// peer identified by peerIDStr, and starts a local HTTP proxy on 127.0.0.1.
//
// bootstrapPeers is a comma-separated list of multiaddrs for DHT bootstrap.
// If empty, IPFS default bootstrap peers are used.
//
// Returns a Proxy handle that provides the local URL and a Stop method.
// gomobile exports this as MobileproxyStartProxy on iOS / Mobileproxy.startProxy on Android.
func StartProxy(peerIDStr string, bootstrapPeers string) (*Proxy, error) {
	ctx, cancel := context.WithCancel(context.Background())

	targetPeerID, err := peer.Decode(strings.TrimSpace(peerIDStr))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("invalid peer ID %q: %w", peerIDStr, err)
	}

	// Create a lightweight libp2p node for the mobile client
	h, err := libp2phost.New(
		libp2phost.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/0",
			"/ip4/0.0.0.0/udp/0/quic-v1",
		),
		libp2phost.NATPortMap(),
		libp2phost.EnableHolePunching(),
		libp2phost.EnableRelay(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create libp2p host: %w", err)
	}

	// Start DHT for peer discovery
	kdht, err := dht.New(ctx, h, dht.Mode(dht.ModeClient))
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("create DHT: %w", err)
	}

	if err := kdht.Bootstrap(ctx); err != nil {
		kdht.Close()
		h.Close()
		cancel()
		return nil, fmt.Errorf("bootstrap DHT: %w", err)
	}

	// Connect to bootstrap peers
	bsPeers := parseBootstrapPeers(bootstrapPeers)
	connectBootstrapPeers(ctx, h, bsPeers)

	// Wait for peer discovery and connect to the bridge
	if err := discoverAndConnect(ctx, h, kdht, targetPeerID); err != nil {
		kdht.Close()
		h.Close()
		cancel()
		return nil, fmt.Errorf("connect to bridge peer: %w", err)
	}

	// Start local HTTP proxy on 127.0.0.1
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		kdht.Close()
		h.Close()
		cancel()
		return nil, fmt.Errorf("listen on loopback: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port

	proxy := &Proxy{
		host:     h,
		dht:      kdht,
		peerID:   targetPeerID,
		listener: listener,
		cancel:   cancel,
		port:     port,
	}

	// The HTTP server handles all incoming local requests by opening a
	// libp2p stream to the bridge peer and proxying the raw TCP data.
	srv := &http.Server{
		Handler: http.HandlerFunc(proxy.handleRequest),
	}
	proxy.server = srv

	go func() {
		if srvErr := srv.Serve(listener); srvErr != nil && srvErr != http.ErrServerClosed {
			log.Printf("[mobileproxy] server error: %v", srvErr)
		}
	}()

	log.Printf("[mobileproxy] started on 127.0.0.1:%d, bridge peer: %s", port, targetPeerID.String())

	return proxy, nil
}

// HTTPBaseURL returns the local HTTP base URL, e.g. "http://127.0.0.1:54321".
// Use this as the bridge endpoint in BridgeApi / BridgeAPI.
func (p *Proxy) HTTPBaseURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", p.port)
}

// WSBaseURL returns the local WebSocket base URL, e.g. "ws://127.0.0.1:54321".
func (p *Proxy) WSBaseURL() string {
	return fmt.Sprintf("ws://127.0.0.1:%d", p.port)
}

// Port returns the local port the proxy is listening on.
func (p *Proxy) Port() int {
	return p.port
}

// PeerID returns the bridge peer ID string this proxy is connected to.
func (p *Proxy) PeerIDStr() string {
	return p.peerID.String()
}

// IsConnected returns true if the libp2p host has an active connection
// to the bridge peer.
func (p *Proxy) IsConnected() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.host == nil {
		return false
	}
	return p.host.Network().Connectedness(p.peerID) == network.Connected
}

// Stop shuts down the proxy, closing the local listener and the libp2p node.
func (p *Proxy) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	if p.server != nil {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
		p.server.Shutdown(shutdownCtx)
		shutdownCancel()
		p.server = nil
	}
	if p.dht != nil {
		p.dht.Close()
		p.dht = nil
	}
	if p.host != nil {
		p.host.Close()
		p.host = nil
	}
}

// handleRequest proxies an HTTP request by opening a libp2p stream to the
// bridge peer, sending the raw HTTP request, and copying the response back.
// This uses HTTP/1.1 connection hijacking for WebSocket upgrade support.
func (p *Proxy) handleRequest(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	h := p.host
	pid := p.peerID
	p.mu.Unlock()

	if h == nil {
		http.Error(w, "proxy stopped", http.StatusServiceUnavailable)
		return
	}

	// Determine protocol based on whether this is a WebSocket upgrade
	protocol := protocolHTTP
	if isWebSocketUpgrade(r) {
		protocol = protocolWS
	}

	// Open a libp2p stream to the bridge
	streamCtx, streamCancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer streamCancel()

	s, err := h.NewStream(streamCtx, pid, protocol)
	if err != nil {
		http.Error(w, fmt.Sprintf("stream error: %v", err), http.StatusBadGateway)
		return
	}
	defer s.Close()

	// Hijack the connection to get raw TCP access for bidirectional proxying
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}

	clientConn, clientBuf, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, fmt.Sprintf("hijack error: %v", err), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Write the original request to the libp2p stream
	if err := r.Write(s); err != nil {
		log.Printf("[mobileproxy] write request error: %v", err)
		return
	}

	// Bidirectional copy between client connection and libp2p stream
	done := make(chan struct{}, 2)

	// Bridge -> Client
	go func() {
		io.Copy(clientConn, s)
		done <- struct{}{}
	}()

	// Client -> Bridge (use buffered reader for any data already read)
	go func() {
		if clientBuf.Reader.Buffered() > 0 {
			io.Copy(s, clientBuf)
		}
		io.Copy(s, clientConn)
		done <- struct{}{}
	}()

	<-done
}

func isWebSocketUpgrade(r *http.Request) bool {
	for _, val := range r.Header["Upgrade"] {
		if strings.EqualFold(val, "websocket") {
			return true
		}
	}
	return false
}

// discoverAndConnect uses the DHT to find the bridge peer and connect to it.
// It first checks if the peer is already known, then falls back to DHT lookup.
func discoverAndConnect(ctx context.Context, h host.Host, kdht *dht.IpfsDHT, target peer.ID) error {
	// First try direct connection if addresses are already known
	if h.Network().Connectedness(target) == network.Connected {
		return nil
	}

	// Use DHT to discover the peer's addresses
	findCtx, findCancel := context.WithTimeout(ctx, 30*time.Second)
	defer findCancel()

	peerInfo, err := kdht.FindPeer(findCtx, target)
	if err != nil {
		return fmt.Errorf("peer not found via DHT: %w", err)
	}

	connectCtx, connectCancel := context.WithTimeout(ctx, 15*time.Second)
	defer connectCancel()

	if err := h.Connect(connectCtx, peerInfo); err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}

	return nil
}

func parseBootstrapPeers(csv string) []string {
	if strings.TrimSpace(csv) == "" {
		return defaultBootstrapPeers()
	}
	parts := strings.Split(csv, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return defaultBootstrapPeers()
	}
	return result
}

func connectBootstrapPeers(ctx context.Context, h host.Host, peers []string) {
	for _, addr := range peers {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			continue
		}
		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			continue
		}
		go func(pi peer.AddrInfo) {
			connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			h.Connect(connectCtx, pi)
		}(*pi)
	}
}

func defaultBootstrapPeers() []string {
	return []string{
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
	}
}
