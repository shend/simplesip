package parser

import (
	"errors"

	"github.com/jart/gosip/sip"

	"github.com/shend/simplesip/message"
)

// Parser is SIP parser implementation
type Parser struct {
}

func (p *Parser) ParseMsg(data []byte) (msg *message.Message, err error) {
	msg0, err := sip.ParseMsg(data)
	if err != nil {
		return nil, err
	}
	if msg0.Via == nil {
		return nil, errors.New("invalid SIP: \"Via\" header field is mandatory")
	}
	msg1 := &message.Message{
		Msg:       msg0.Copy(),
		Transport: msg0.Via.Transport,
	}
	return msg1, nil
}

// NewParser creates a new Parser.
func NewParser() Parser {
	p := Parser{}
	return p
}
