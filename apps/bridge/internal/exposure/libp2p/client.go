package libp2p

import (
	"context"
	"fmt"
	"net"
	"time"

	libp2phost "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Dialer creates TCP-like connections to a remote libp2p peer running the
// codex-bridge protocol. Remote peers open streams that get proxied to the
// bridge's local gateway, so callers can treat the resulting connection like
// a regular net.Conn.
type Dialer struct {
	host   host.Host
	peerID peer.ID
}

// NewDialer connects to the bridge peer identified by peerMultiaddr.
// The address must include /p2p/<peerID>, e.g.
// "/ip4/1.2.3.4/tcp/4001/p2p/QmPeerID...".
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

// DialHTTP opens a libp2p stream using the bridge HTTP protocol and returns
// it as a net.Conn. The caller can write raw HTTP requests to this connection.
func (d *Dialer) DialHTTP(ctx context.Context) (net.Conn, error) {
	s, err := d.host.NewStream(ctx, d.peerID, protocolHTTP)
	if err != nil {
		return nil, fmt.Errorf("open HTTP stream: %w", err)
	}
	return &streamConn{Stream: s}, nil
}

// DialWS opens a libp2p stream using the bridge WebSocket protocol.
func (d *Dialer) DialWS(ctx context.Context) (net.Conn, error) {
	s, err := d.host.NewStream(ctx, d.peerID, protocolWS)
	if err != nil {
		return nil, fmt.Errorf("open WS stream: %w", err)
	}
	return &streamConn{Stream: s}, nil
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
