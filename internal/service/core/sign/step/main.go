package step

import (
	"context"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type IController interface {
	Run(ctx context.Context)
	WaitFinish()
	Receive(party rarimo.Party, request types.MsgSubmitRequest) error
}
