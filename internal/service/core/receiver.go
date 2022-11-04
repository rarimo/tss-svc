package core

import (
	"context"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type IGlobalReceiver interface {
	Receive(request *types.MsgSubmitRequest) error
}

type IReceive interface {
	ReceiveFromSender(sender rarimo.Party, request *types.MsgSubmitRequest)
}

type Msg struct {
	Request *types.MsgSubmitRequest
	Sender  rarimo.Party
}

type RequestQueue struct {
	Queue chan *Msg
}

func NewQueue(cap int) *RequestQueue {
	return &RequestQueue{Queue: make(chan *Msg, cap)}
}

func (r *RequestQueue) ProcessQueue(ctx context.Context, f IReceive) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-r.Queue:
			if !ok {
				return
			}
			f.ReceiveFromSender(msg.Sender, msg.Request)
		}
	}
}
