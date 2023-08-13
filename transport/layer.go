package transport

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/shend/simplesip/message"
	"github.com/shend/simplesip/parser"
	"github.com/shend/simplesip/util"
)

var (
	ErrNetworkNotSupported = errors.New("protocol not supported")
)

// Layer implementation.
type Layer struct {
	udp *UDPTransport
	//tcp           *TCPTransport

	transports map[string]Transport

	listenPorts   map[string][]int
	listenPortsMu sync.Mutex

	handlers []message.RequestHandler

	// Parser used by transport layer. It can be overridden before setting up network transports
	Parser parser.Parser
}

// NewLayer creates transport layer.
func NewLayer(parser parser.Parser) *Layer {
	l := &Layer{
		transports:  make(map[string]Transport),
		listenPorts: make(map[string][]int),
		Parser:      parser,
	}

	// Make some default transports available.
	l.udp = NewUDPTransport(parser)

	// Fill map for fast access
	l.transports["udp"] = l.udp

	return l
}

func ParseAddr(addr string) (host string, port int, err error) {
	host, pstr, err := net.SplitHostPort(addr)
	if err != nil {
		return host, port, err
	}

	port, err = strconv.Atoi(pstr)
	return host, port, err
}

// AppendHandlers appends handlers to current handlers
func (l *Layer) AppendHandlers(handlers ...message.RequestHandler) {
	l.handlers = append(l.handlers, handlers...)
}

// OnMessage is main function which will be called on any new message by transport layer
func (l *Layer) OnMessage(h message.RequestHandler) {
	l.handlers = append(l.handlers, h)
}

// handleMessage is transport layer for handling messages
func (l *Layer) handleMessage(msg *message.Message) {
	for _, h := range l.handlers {
		h(msg)
	}
}

// ServeUDP will listen on udp connection
func (l *Layer) ServeUDP(c net.PacketConn) error {
	_, port, err := ParseAddr(c.LocalAddr().String())
	if err != nil {
		return err
	}

	l.addListenPort("udp", port)

	return l.udp.Serve(c, l.handleMessage)
}

// ServeTCP will listen on tcp connection
func (l *Layer) ServeTCP(c net.Listener) error {
	_, port, err := ParseAddr(c.Addr().String())
	if err != nil {
		return err
	}

	l.addListenPort("tcp", port)

	// TODO 实现TCP层
	return nil
}

// ListenAndServe serve on any network. This function will block
// Network supported: udp, tcp
func (l *Layer) ListenAndServe(network string, addr string) error {
	network = strings.ToLower(network)
	switch network {
	case "udp":
		// resolve local UDP endpoint
		laddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			return fmt.Errorf("fail to resolve address. err=%w", err)
		}

		conn, err := net.ListenUDP("udp", laddr)
		if err != nil {
			return fmt.Errorf("listen udp error. err=%w", err)
		}

		return l.ServeUDP(conn)
	case "tcp":
		// TODO: 实现TCP
		return ErrNetworkNotSupported
	}

	return ErrNetworkNotSupported
}

func (l *Layer) addListenPort(network string, port int) {
	l.listenPortsMu.Lock()
	defer l.listenPortsMu.Unlock()

	if _, ok := l.listenPorts[network]; !ok {
		if l.listenPorts[network] == nil {
			l.listenPorts[network] = make([]int, 0)
		}
		l.listenPorts[network] = append(l.listenPorts[network], port)
	}
}

func (l *Layer) WriteMsg(msg *message.Message) error {
	network := msg.Transport
	addr := msg.Destination
	return l.WriteMsgTo(msg, addr, network)
}

func (l *Layer) WriteMsgTo(msg *message.Message, addr string, network string) error {
	var conn Connection
	var err error

	if msg.Msg.IsResponse() {
		conn, err = l.GetConnection(network, addr)
		if err != nil {
			return err
		}
	} else {
		conn, err = l.GetConnection(network, addr)
		if err != nil {
			return err
		}
	}

	if err := conn.WriteMsg(msg); err != nil {
		return err
	}

	return nil
}

// GetConnection gets existing or creates new connection based on addr
func (l *Layer) GetConnection(network, addr string) (Connection, error) {
	network = NetworkToLower(network)
	return l.getConnection(network, addr)
}

func (l *Layer) getConnection(network, addr string) (Connection, error) {
	transport, ok := l.transports[network]
	if !ok {
		return nil, fmt.Errorf("transport %s is not supported", network)
	}

	c, err := transport.GetConnection(addr)
	if err == nil && c == nil {
		return nil, fmt.Errorf("connection does not exist")
	}

	return c, err
}

func (l *Layer) Close() error {
	var werr error
	for _, t := range l.transports {
		if err := t.Close(); err != nil {
			// For now dump last error
			werr = err
		}
	}
	return werr
}

// NetworkToLower is faster function converting UDP, TCP to udp, tcp
func NetworkToLower(network string) string {
	// Switch is faster then lower
	switch network {
	case "UDP":
		return "udp"
	case "TCP":
		return "tcp"
	case "TLS":
		return "tls"
	case "WS":
		return "ws"
	default:
		return util.ASCIIToLower(network)
	}
}
