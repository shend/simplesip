package transport

import (
	"github.com/shend/simplesip/message"
)

type Connection interface {
	// WriteMsg marshals message and sends to socket
	WriteMsg(msg *message.Message) error

	// Close the collection
	Close() error
}
