package libp2p

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	libp2phost "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	gostream "github.com/libp2p/go-libp2p/p2p/net/gostream"
	"github.com/multiformats/go-multiaddr"
)

// Dialer creates HTTP connections to a remote libp2p peer running the
// codex-bridge protocol. This is used by clients (e.g. mobile apps) that
// want to reach the bridge over the libp2p network.
type Dialer struct {
	host   host.Host
	peerID peer.ID
}

// NewDialer creates a new libp2p dialer that can connect to a bridge peer.
// The peerMultiaddr should include the /p2p/<peerID> component,
// e.g. "/ip4/1.2.3.4/tcp/4001/p2p/QmPeerID...".
func NewDialer(ctx context.Context, peerMultiaddr string) (*Dialer, error) {
	ma, err := multiaddr.NewMultiaddr(peerMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("invalid multiaddr %q: %w", peerMultiaddr, err)
	}

	pi, err := peer.AddrInfoFromP2pAddr(ma)
	if err != nil {
		return nil, fmt.Errorf("extract peer info: %w", err)
	}

	h, err := libp2phost.New(
		libp2phost.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
		libp2phost.NATPortMap(),
		libp2phost.EnableHolePunching(),
		libp2phost.EnableRelay(),
	)
	if err != nil {
		return nil, fmt.Errorf("create libp2p host: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := h.Connect(connectCtx, *pi); err != nil {
		h.Close()
		return nil, fmt.Errorf("connect to peer %s: %w", pi.ID.String(), err)
	}

	return &Dialer{
		host:   h,
		peerID: pi.ID,
	}, nil
}

// HTTPTransport returns an http.RoundTripper that routes requests
// over a libp2p stream to the bridge's HTTP protocol handler.
func (d *Dialer) HTTPTransport() http.RoundTripper {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return gostream.Dial(ctx, d.host, d.peerID, protocolHTTP)
		},
	}
}

// WSTransport returns an http.RoundTripper for WebSocket upgrade requests
// over a libp2p stream to the bridge's WebSocket protocol handler.
func (d *Dialer) WSTransport() http.RoundTripper {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return gostream.Dial(ctx, d.host, d.peerID, protocolWS)
		},
	}
}

// Close shuts down the dialer's libp2p host.
func (d *Dialer) Close() error {
	if d.host != nil {
		return d.host.Close()
	}
	return nil
}

// PeerID returns the remote peer ID this dialer connects to.
func (d *Dialer) PeerID() string {
	return d.peerID.String()
}
