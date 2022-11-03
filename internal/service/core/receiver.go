package core

import (
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type IGlobalReceiver interface {
	Receive(request *types.MsgSubmitRequest) error
}

type IReceiver interface {
	Receive(sender rarimo.Party, request *types.MsgSubmitRequest)
}

type OrderMsg struct {
	Request *types.MsgSubmitRequest
	Sender  rarimo.Party
}

var _ IReceiver = &Receiver{}

type Receiver struct {
	Order chan *OrderMsg
}

func NewReceiver(cap int) *Receiver {
	return &Receiver{Order: make(chan *OrderMsg, cap)}
}

func (r *Receiver) Receive(sender rarimo.Party, request *types.MsgSubmitRequest) {
	r.Order <- &OrderMsg{
		Request: request,
		Sender:  sender,
	}
}
