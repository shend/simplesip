package message

import (
	"github.com/jart/gosip/sip"
)

type RequestMiddleware func(req *Message)

type ResponseMiddleware func(res *Message) bool

type RequestHandler func(req *Message) *Message

type RequestMethod string

func (r RequestMethod) String() string { return string(r) }

func FromString(r string) RequestMethod { return RequestMethod(r) }

// StatusCode - 响应状态码: 1xx - 6xx
type StatusCode int

type Message struct {
	Msg       *sip.Msg
	Transport string

	Source      string
	Destination string
}

// 请求方法常量
const (
	INVITE    RequestMethod = "INVITE"
	ACK       RequestMethod = "ACK"
	CANCEL    RequestMethod = "CANCEL"
	BYE       RequestMethod = "BYE"
	REGISTER  RequestMethod = "REGISTER"
	OPTIONS   RequestMethod = "OPTIONS"
	SUBSCRIBE RequestMethod = "SUBSCRIBE"
	NOTIFY    RequestMethod = "NOTIFY"
	REFER     RequestMethod = "REFER"
	INFO      RequestMethod = "INFO"
	MESSAGE   RequestMethod = "MESSAGE"
	PRACK     RequestMethod = "PRACK"
	UPDATE    RequestMethod = "UPDATE"
	PUBLISH   RequestMethod = "PUBLISH"
)

func (m *Message) Clone() Message {
	return Message{
		Msg:         m.Msg.Copy(),
		Transport:   m.Transport,
		Source:      m.Source,
		Destination: m.Destination,
	}
}
