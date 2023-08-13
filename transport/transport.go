package transport

var (
	SIPDebug bool
)

const (
	TransportUDP = "UDP"
	TransportTCP = "TCP"

	transportBufferSize uint16 = 65535
)

// Transport implements network specific features.
type Transport interface {
	Network() string
	String() string
	GetConnection(addr string) (Connection, error)
	Close() error
}
