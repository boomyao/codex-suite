package libp2p

import (
	"net"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
)

// streamConn wraps a libp2p network.Stream to implement net.Conn.
type streamConn struct {
	network.Stream
}

var _ net.Conn = (*streamConn)(nil)

func (c *streamConn) LocalAddr() net.Addr {
	return &streamAddr{c.Stream.Conn().LocalMultiaddr().String()}
}

func (c *streamConn) RemoteAddr() net.Addr {
	return &streamAddr{c.Stream.Conn().RemoteMultiaddr().String()}
}

func (c *streamConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *streamConn) SetReadDeadline(t time.Time) error {
	c.Stream.SetReadDeadline(t)
	return nil
}

func (c *streamConn) SetWriteDeadline(t time.Time) error {
	c.Stream.SetWriteDeadline(t)
	return nil
}

// streamAddr implements net.Addr for libp2p multiaddresses.
type streamAddr struct {
	addr string
}

func (a *streamAddr) Network() string { return "libp2p" }
func (a *streamAddr) String() string  { return a.addr }
