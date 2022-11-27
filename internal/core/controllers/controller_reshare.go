package controllers

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type ReshareController struct {
	mu   *sync.Mutex
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

	r.mu.Lock()
	defer r.mu.Unlock()

	result := r.party.Result()
	if result == nil {
		r.data.Processing = false
		return
	}

	err := r.storage.SetTssSecret(secret.NewTssSecret(result, r.storage.GetTssSecret().Params, r.storage.GetTssSecret()))
	if err != nil {
		panic(err)
	}

	r.data.New.LocalTss.LocalData = r.storage.GetTssSecret().Data
	r.data.New.LocalPrivateKey = r.storage.GetTssSecret().Prv
	r.data.New.LocalPubKey = r.storage.GetTssSecret().PubKeyStr()
	r.data.New.GlobalPubKey = r.storage.GetTssSecret().GlobalPubKeyStr()
	r.data.NewGlobalPublicKey = r.data.New.GlobalPubKey
}

func (r *ReshareController) WaitFor() {
	r.party.WaitFor()
}

func (r *ReshareController) Next() IController {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.data.Processing {
		return r.factory.GetSignKeyController()
	}

	return r.factory.GetFinishController()
}

func (r *ReshareController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_RESHARE
}
