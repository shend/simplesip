package transport

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/shend/simplesip/message"
	"github.com/shend/simplesip/parser"
)

var (
	UDPMTUSize = 1500

	ErrUDPMTUCongestion = errors.New("size of packet larger than MTU")
)

// UDPTransport implements Transport interface
type UDPTransport struct {
	listener net.PacketConn
	parser   parser.Parser
	conn     *UDPConnection

	pool ConnectionPool
}

func NewUDPTransport(parser parser.Parser) *UDPTransport {
	p := &UDPTransport{
		parser: parser,
		conn:   nil, // Making sure interface is nil in returns
		pool:   NewConnectionPool(),
	}
	return p
}

func (t *UDPTransport) Network() string {
	return TransportUDP
}

func (t *UDPTransport) String() string {
	return "transport<UDP>"
}

func (t *UDPTransport) GetConnection(addr string) (Connection, error) {
	return t.conn, nil
}

func (t *UDPTransport) Close() error {
	t.pool.RLock()
	defer t.pool.RUnlock()
	var err error
	return err
}

func (t *UDPTransport) ListenAndServe(addr string, handler func(msg *message.Message)) error {
	// resolve local UDP endpoint
	laddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve IP address. err=%w", err)
	}
	conn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		return fmt.Errorf("failed to listen udp address. err=%w", err)
	}

	return t.Serve(conn, handler)
}

func (t *UDPTransport) Serve(conn net.PacketConn, handler func(msg *message.Message)) error {
	slog.Debug(fmt.Sprintf("begin listening on %s %s", t.Network(), conn.LocalAddr().String()))

	c := &UDPConnection{PacketConn: conn}

	// In case single connection avoid pool
	if t.pool.Size() == 0 {
		t.conn = c
	} else {
		t.conn = nil
	}

	t.pool.Add(conn.LocalAddr().String(), c)

	t.readConnection(c, handler)

	return nil
}

func (t *UDPTransport) readConnection(conn *UDPConnection, handler func(*message.Message)) {
	buf := make([]byte, transportBufferSize)
	defer conn.Close()
	for {
		num, raddr, err := conn.ReadFrom(buf)

		if err != nil {
			slog.Error("read udp error", "err", err)
			return
		}

		data := buf[:num]
		if len(bytes.Trim(data, "\x00")) == 0 {
			continue
		}

		t.parseAndHandle(data, raddr.String(), handler)
	}
}

func (t *UDPTransport) readConnectedConnection(conn *UDPConnection, handler func(*message.Message)) {
	buf := make([]byte, transportBufferSize)
	raddr := conn.Conn.RemoteAddr().String()
	defer func() {
		// Delete connection from pool only when closed
		// TODO does this makes sense closing if reading fails
		t.pool.Del(raddr)
	}()
	for {
		num, err := conn.Read(buf)

		if err != nil {
			slog.Error("error occurred while reading data from connection", "err", err)
			return
		}

		data := buf[:num]
		if len(bytes.Trim(data, "\x00")) == 0 {
			continue
		}

		t.parseAndHandle(data, raddr, handler)
	}
}

func (t *UDPTransport) parseAndHandle(data []byte, src string, handler func(*message.Message)) {
	// Check is keep alive
	if len(data) <= 4 {
		// One or 2 CRLF
		if len(bytes.Trim(data, "\r\n")) == 0 {
			slog.Debug("Keep alive CRLF received")
			return
		}
	}

	msg, err := t.parser.ParseMsg(data)
	if err != nil {
		slog.Debug("failed to parse", slog.String("data", string(data)), slog.Any("err", err))
		return
	}

	if msg.Msg.Via == nil {
		slog.Warn("invalid SIP: \"Via\" header field is mandatory")
		return
	}

	if msg.Msg.Via.Transport != "UDP" {
		slog.Debug("transport mismatch", slog.String("transport", msg.Msg.Via.Transport))
		return
	}

	msg.Transport = "UDP"
	msg.Source = src

	handler(msg)
}

type UDPConnection struct {
	PacketConn net.PacketConn
	Conn       net.Conn

	mu sync.RWMutex
}

func (c *UDPConnection) Close() error {
	if c.Conn == nil {
		return nil
	}
	c.mu.Lock()
	c.mu.Unlock()
	return c.Conn.Close()
}

func (c *UDPConnection) Read(b []byte) (n int, err error) {
	if SIPDebug {
		slog.Debug(fmt.Sprintf("UDP read %s <- %s:\n%s", c.Conn.LocalAddr().String(), c.Conn.RemoteAddr().String(), string(b)))
	}
	return c.Conn.Read(b)
}

func (c *UDPConnection) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, addr, err = c.PacketConn.ReadFrom(b)
	if SIPDebug {
		slog.Debug("UDP read %s <- %s:\n%s", c.PacketConn.LocalAddr().String(), addr.String(), string(b[:n]))
	}
	return n, addr, err
}

func (c *UDPConnection) WriteTo(b []byte, addr net.Addr) (n int, err error) {
	n, err = c.PacketConn.WriteTo(b, addr)
	if SIPDebug {
		slog.Debug(fmt.Sprintf("UDP write %s -> %s:\n%s", c.PacketConn.LocalAddr().String(), addr.String(), string(b)))
	}
	return n, err
}

func (c *UDPConnection) WriteMsg(msg *message.Message) error {
	var buf bytes.Buffer
	msg.Msg.Append(&buf)
	data := buf.Bytes()

	if len(data) > UDPMTUSize-200 {
		return ErrUDPMTUCongestion
	}

	dst := msg.Destination
	raddr, err := net.ResolveUDPAddr("udp", dst)
	if err != nil {
		return err
	}

	n, err := c.WriteTo(data, raddr)
	if err != nil {
		return fmt.Errorf("udp conn %s err. %w", c.PacketConn.LocalAddr().String(), err)
	}

	if n == 0 {
		return fmt.Errorf("wrote 0 bytes")
	}

	if n != len(data) {
		return fmt.Errorf("fail to write full message")
	}

	return nil
}
