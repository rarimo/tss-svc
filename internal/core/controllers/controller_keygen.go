package controllers

import (
	"context"
	"crypto/elliptic"
	"database/sql"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/data"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
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
	pg      *pg.Storage
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
	if k.storage.GetTssSecret().Prv != nil {
		panic(ErrSecretDataAlreadyInitialized)
	}

	k.log.Infof("Starting %s", k.Type().String())
	k.party.Run(ctx)
	k.wg.Add(1)
	go k.run(ctx)
}

func (k *KeygenController) WaitFor() {
	k.wg.Wait()
}

func (k *KeygenController) Next() IController {
	return k.factory.GetFinishController()
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
	k.party.WaitFor()

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
	k.data.New.LocalPubKey = k.storage.GetTssSecret().PubKeyStr()
	k.data.New.GlobalPubKey = k.storage.GetTssSecret().GlobalPubKeyStr()
	k.data.New.T = ((k.data.New.N + 2) / 3) * 2
	k.data.New.IsActive = true
	k.data.Processing = true

	for i := range result.Ks {
		partyId := k.data.New.SortedPartyIDs.FindByKey(result.Ks[i])
		for j := range k.data.New.Parties {
			if k.data.New.Parties[j].Account == partyId.Id {
				k.data.New.Parties[j].PubKey = hexutil.Encode(elliptic.Marshal(eth.S256(), result.BigXj[i].X(), result.BigXj[i].Y()))
				break
			}
		}
	}
}

func (k *KeygenController) updateSessionData() {
	session, err := k.pg.SessionQ().SessionByID(int64(k.data.SessionId), false)
	if err != nil {
		k.log.WithError(err).Error("error selecting session")
		return
	}

	if session == nil {
		k.log.Error("session entry is not initialized")
		return
	}

	session.SessionType = sql.NullInt64{
		Int64: int64(types.SessionType_KeygenSession),
		Valid: true,
	}

	session.DataID = sql.NullInt64{
		Int64: session.ID,
		Valid: true,
	}

	err = k.pg.KeygenSessionDatumQ().Insert(&data.KeygenSessionDatum{
		ID:      session.ID,
		Parties: partyAccounts(k.data.New.Parties),
		Key: sql.NullString{
			String: k.data.New.GlobalPubKey,
			Valid:  k.data.New.GlobalPubKey != "",
		},
	})

	if err != nil {
		k.log.WithError(err).Error("error creating session data entry")
		return
	}

	if err = k.pg.SessionQ().Update(session); err != nil {
		k.log.WithError(err).Error("error updating session entry")
	}
}
