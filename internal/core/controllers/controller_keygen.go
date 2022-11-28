package controllers

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/crypto"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type KeygenController struct {
	mu sync.Mutex
	wg *sync.WaitGroup

	data *LocalSessionData

	auth *core.RequestAuthorizer
	log  *logan.Entry

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
	k.log.Infof("Starting %s", k.Type().String())
	k.party.Run(ctx)
	k.wg.Add(1)
	go k.run(ctx)
}

func (k *KeygenController) WaitFor() {
	k.party.WaitFor()
	k.wg.Wait()
}

func (k *KeygenController) Next() IController {
	return k.factory.GetSignController(hexutil.Encode(crypto.GetPartiesHash(k.data.New.Parties)), k.data.New)
}

func (k *KeygenController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_KEYGEN
}

func (k *KeygenController) run(ctx context.Context) {
	defer func() {
		k.log.Infof("%s finished", k.Type().String())
		k.wg.Done()
	}()

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

	k.data.SessionType = types.SessionType_KeygenSession
	k.data.New.LocalTss.LocalData = k.storage.GetTssSecret().Data
	k.data.New.LocalPrivateKey = k.storage.GetTssSecret().Prv
	k.data.New.LocalPubKey = k.storage.GetTssSecret().PubKeyStr()
	k.data.New.GlobalPubKey = k.storage.GetTssSecret().GlobalPubKeyStr()
	k.data.New.T = ((k.data.New.N + 2) / 3) * 2
	k.data.NewGlobalPublicKey = k.data.New.GlobalPubKey
}
