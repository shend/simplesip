package message

import (
	"fmt"
	"strings"

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

type RespondFunc func(*Message) error

type Message struct {
	Msg       *sip.Msg
	Transport string

	Source      string
	Destination string

	Respond RespondFunc
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
		Respond:     m.Respond,
	}
}

func (m *Message) GetBranch() string {
	return m.Msg.Via.Param.Get("branch").Value
}

func (m *Message) GetCallID() string {
	return m.Msg.CallID
}

func (m *Message) GetToTag() string {
	toParam := m.Msg.To.Param
	if toParam == nil {
		return ""
	}
	return toParam.Get("tag").Value
}

func (m *Message) GetFromTag() string {
	fromParam := m.Msg.From.Param
	if fromParam == nil {
		return ""
	}
	return fromParam.Get("tag").Value
}

// MakeDialogIDFromMessage creates dialog ID of message.
// returns error if CallID or To tag or From tag does not exist
func (m *Message) MakeDialogIDFromMessage() (string, error) {
	callID := m.GetCallID()
	if callID == "" {
		return "", fmt.Errorf("missing Call-ID header")
	}

	toTag := m.GetToTag()
	if toTag == "" {
		return "", fmt.Errorf("missing tag param in To header")
	}

	fromTag := m.GetFromTag()
	if fromTag == "" {
		return "", fmt.Errorf("missing tag param in From header")
	}

	return MakeDialogID(callID, toTag, fromTag), nil
}

func MakeDialogID(callID, innerID, externalID string) string {
	return strings.Join([]string{callID, innerID, externalID}, "__")
}
