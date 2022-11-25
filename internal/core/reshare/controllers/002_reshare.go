package controllers

import (
	"context"

	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/sign/controllers"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type ReshareController struct {
	*defaultController

	bounds    *core.Bounds
	sessionId uint64
	data      LocalAcceptanceData

	party   tss.ReshareParty
	factory *controllers.ControllerFactory
}

var _ IController = &ReshareController{}

func (r *ReshareController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := r.auth.Auth(request)
	if err != nil {
		return err
	}

	if _, ok := r.data.Accepted[sender.Address]; !ok {
		return ErrSenderHasNotAccepted
	}

	if request.Type != types.RequestType_Sign {
		return ErrInvalidRequestType
	}

	r.party.Receive(sender, request.IsBroadcast, request.Details.Value)
	return nil
}

func (r *ReshareController) Run(ctx context.Context) {
	r.party.Run(ctx)

	<-ctx.Done()
	r.party.Result()
}

func (r *ReshareController) WaitFor() {
	r.party.WaitFor()
}

func (r *ReshareController) Next() IController {
	return nil
}

func (r *ReshareController) Bounds() *core.Bounds {
	return r.bounds
}
