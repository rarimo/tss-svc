package controllers

import (
	"context"
	"crypto/elliptic"
	"database/sql"
	"sync"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"gitlab.com/rarimo/tss/tss-svc/internal/tss"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// KeygenController is responsible for initial key generation. It can only be launched with empty secret storage and
// after finishing will update storage with generated secret.
type KeygenController struct {
	IKeygenController
	wg *sync.WaitGroup

	data *LocalSessionData

	auth  *core.RequestAuthorizer
	log   *logan.Entry
	party *tss.KeygenParty
}

// Implements IController interface
var _ IController = &KeygenController{}

func (k *KeygenController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := k.auth.Auth(request)
	if err != nil {
		return err
	}

	if request.Type != types.RequestType_Keygen {
		return ErrInvalidRequestType
	}

	go k.party.Receive(sender, request.IsBroadcast, request.Details.Value)

	return nil
}

func (k *KeygenController) Run(ctx context.Context) {
	k.log.Infof("Starting: %s", k.Type().String())
	k.party.Run(ctx)
	k.wg.Add(1)
	go k.run(ctx)
}

func (k *KeygenController) WaitFor() {
	k.wg.Wait()
}

func (k *KeygenController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_KEYGEN
}

func (k *KeygenController) run(ctx context.Context) {
	defer func() {
		k.log.Infof("Finishing: %s", k.Type().String())
		k.updateSessionData()
		k.wg.Done()
	}()

	<-ctx.Done()
	k.party.WaitFor()

	result := k.party.Result()
	if result == nil {
		k.data.Processing = false
		return
	}

	k.finish(result)
}

// IKeygenController defines custom logic for every acceptance controller.
type IKeygenController interface {
	Next() IController
	updateSessionData()
	finish(result *keygen.LocalPartySaveData)
}

// DefaultKeygenController represents custom logic for types.SessionType_KeygenSession
type DefaultKeygenController struct {
	data    *LocalSessionData
	pg      *pg.Storage
	log     *logan.Entry
	factory *ControllerFactory
}

// Implements IKeygenController interface
var _ IKeygenController = &DefaultKeygenController{}

func (d *DefaultKeygenController) Next() IController {
	return d.factory.GetFinishController()
}

func (d *DefaultKeygenController) updateSessionData() {
	session, err := d.pg.KeygenSessionDatumQ().KeygenSessionDatumByID(int64(d.data.SessionId), false)
	if err != nil {
		d.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		d.log.Error("Session entry is not initialized")
		return
	}

	session.Parties = partyAccounts(d.data.Set.Parties)
	session.Key = sql.NullString{
		String: d.data.NewSecret.GlobalPubKey(),
		Valid:  d.data.Processing,
	}

	if err = d.pg.KeygenSessionDatumQ().Update(session); err != nil {
		d.log.WithError(err).Error("Error updating session entry")
	}
}

func (d *DefaultKeygenController) finish(result *keygen.LocalPartySaveData) {
	if result == nil {
		d.data.Processing = false
		return
	}

	d.data.NewSecret = d.data.Secret.NewWithData(result)
	d.data.Processing = true
}

// ReshareKeygenController represents custom logic for types.SessionType_ReshareSession
type ReshareKeygenController struct {
	data    *LocalSessionData
	pg      *pg.Storage
	log     *logan.Entry
	factory *ControllerFactory
}

// Implements IKeygenController interface
var _ IKeygenController = &ReshareKeygenController{}

func (r *ReshareKeygenController) Next() IController {
	if r.data.Processing && !contains(r.data.Set.UnverifiedParties, r.data.Secret.AccountAddress()) {
		return r.factory.GetKeySignController(hexutil.Encode(eth.Keccak256(hexutil.MustDecode(r.data.NewSecret.GlobalPubKey()))))
	}

	return r.factory.GetFinishController()
}

func (r *ReshareKeygenController) updateSessionData() {
	session, err := r.pg.ReshareSessionDatumQ().ReshareSessionDatumByID(int64(r.data.SessionId), false)
	if err != nil {
		r.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		r.log.Error("Session entry is not initialized")
		return
	}

	session.NewKey = sql.NullString{
		String: r.data.NewSecret.GlobalPubKey(),
		Valid:  r.data.Processing,
	}

	if err = r.pg.ReshareSessionDatumQ().Update(session); err != nil {
		r.log.WithError(err).Error("Error updating session data entry")
	}
}

func (r *ReshareKeygenController) finish(result *keygen.LocalPartySaveData) {
	r.data.NewSecret = r.data.Secret.NewWithData(result)
	r.data.NewParties = make([]*rarimo.Party, len(r.data.Set.Parties))

	partyIDs := core.PartyIds(r.data.Set.Parties)
	for i := range result.Ks {
		partyId := partyIDs.FindByKey(result.Ks[i])
		for j, party := range r.data.Set.Parties {
			if party.Account == partyId.Id {
				r.data.NewParties[j] = &rarimo.Party{
					PubKey:   hexutil.Encode(elliptic.Marshal(eth.S256(), result.BigXj[i].X(), result.BigXj[i].Y())),
					Address:  party.Address,
					Account:  party.Account,
					Verified: true,
				}
				break
			}
		}
	}
}
