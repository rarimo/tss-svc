package controllers

import (
	"context"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type ReshareController struct {
	data *LocalSessionData

	broadcast *connectors.BroadcastConnector
	auth      *core.RequestAuthorizer
	log       *logan.Entry

	party   *tss.ReshareParty
	storage secret.Storage
	factory *ControllerFactory
}

var _ IController = &ReshareController{}

func (r *ReshareController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := r.auth.Auth(request)
	if err != nil {
		return err
	}

	if _, ok := r.data.Acceptances[sender.Address]; !ok {
		return ErrSenderHasNotAccepted
	}

	if request.Type != types.RequestType_Reshare {
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

func (r *ReshareController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_RESHARE
}
