package simplesip

import "github.com/shend/simplesip/message"

func NewResponseFromRequest(req *message.Message, status int, phrase string) *message.Message {
	res := req.Clone()
	res.Msg.Status = status
	res.Msg.Phrase = phrase
	res.Destination, res.Source = res.Source, res.Destination
	return &res
}
