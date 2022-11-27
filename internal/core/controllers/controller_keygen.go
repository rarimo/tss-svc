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

type KeygenController struct {
	mu *sync.Mutex

	data *LocalSessionData

	broadcast *connectors.BroadcastConnector
	auth      *core.RequestAuthorizer
	log       *logan.Entry

	storage secret.Storage
	party   *tss.KeygenParty
	factory *ControllerFactory
}

var _ IController = &KeygenController{}

func (k *KeygenController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := k.auth.Auth(request)
	if err != nil {
		return err
	}

	if request.Type != types.RequestType_Keygen {
		return ErrInvalidRequestType
	}

	k.party.Receive(sender, request.IsBroadcast, request.Details.Value)

	return nil
}

func (k *KeygenController) Run(ctx context.Context) {
	k.party.Run(ctx)

	<-ctx.Done()

	k.mu.Lock()
	defer k.mu.Unlock()

	result := k.party.Result()
	if result == nil {
		k.data.Processing = false
		return
	}

	err := k.storage.SetTssSecret(secret.NewTssSecret(result, k.storage.GetTssSecret().Params, k.storage.GetTssSecret()))
	if err != nil {
		panic(err)
	}

	k.data.New.LocalTss.LocalData = k.storage.GetTssSecret().Data
	k.data.New.LocalPrivateKey = k.storage.GetTssSecret().Prv
	k.data.New.LocalPubKey = k.storage.GetTssSecret().PubKeyStr()
	k.data.New.GlobalPubKey = k.storage.GetTssSecret().GlobalPubKeyStr()
	k.data.NewGlobalPublicKey = k.data.New.GlobalPubKey
}

func (k *KeygenController) WaitFor() {
	k.party.WaitFor()
}

func (k *KeygenController) Next() IController { // TODO

	return nil
}

func (k *KeygenController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_KEYGEN
}
