package controllers

import (
	"context"
	"crypto/elliptic"
	"database/sql"
	"sync"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/tss"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// iKeygenController defines custom logic for every acceptance controller.
type iKeygenController interface {
	Next() IController
	updateSessionData(ctx core.Context)
	finish(ctx core.Context, result *keygen.LocalPartySaveData)
}

// KeygenController is responsible for initial key generation. It can only be launched with empty secret storage and
// after finishing will update storage with generated secret.
type KeygenController struct {
	iKeygenController
	wg    *sync.WaitGroup
	data  *LocalSessionData
	auth  *core.RequestAuthorizer
	party *tss.KeygenParty
}

// Implements IController interface
var _ IController = &KeygenController{}

// Receive accepts the keygen requests from other parties and delivers them to the `tss.KeygenParty`
func (k *KeygenController) Receive(c context.Context, request *types.MsgSubmitRequest) error {
	sender, err := k.auth.Auth(request)
	if err != nil {
		return err
	}

	if request.Data.Type != types.RequestType_Keygen {
		return ErrInvalidRequestType
	}

	if err := k.party.Receive(sender, request.Data.IsBroadcast, request.Data.Details.Value); err != nil {
		ctx := core.WrapCtx(c)
		ctx.Log().WithError(err).Error("failed to receive request on party")
		// can be done without lock: no remove or change operation exist, only add
		k.data.Offenders[sender.Account] = struct{}{}
	}

	return nil
}

// Run launches the `tss.KeygenParty` logic. After context canceling it will check the tss party result
// and execute `iKeygenController.finish` logic.
func (k *KeygenController) Run(c context.Context) {
	ctx := core.WrapCtx(c)
	ctx.Log().Infof("Starting: %s", k.Type().String())
	k.party.Run(c)
	k.wg.Add(1)
	go k.run(ctx)
}

// WaitFor waits until controller finishes its logic. Context cancel should be called before.
func (k *KeygenController) WaitFor() {
	k.wg.Wait()
}

func (k *KeygenController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_KEYGEN
}

func (k *KeygenController) run(ctx core.Context) {
	defer func() {
		ctx.Log().Infof("Finishing: %s", k.Type().String())
		k.updateSessionData(ctx)
		k.wg.Done()
	}()

	<-ctx.Context().Done()
	k.party.WaitFor()

	result := k.party.Result()
	if result == nil {
		k.data.Processing = false
		return
	}

	k.finish(ctx, result)
}

// defaultKeygenController represents custom logic for types.SessionType_KeygenSession
type defaultKeygenController struct {
	data *LocalSessionData
}

// Implements iKeygenController interface
var _ iKeygenController = &defaultKeygenController{}

// Next returns the finish controller instance.
// WaitFor should be called before.
func (d *defaultKeygenController) Next() IController {
	return d.data.GetFinishController()
}

// updateSessionData updates the database entry according to the controller result.
func (d *defaultKeygenController) updateSessionData(ctx core.Context) {
	if !d.data.Processing {
		return
	}

	session, err := ctx.PG().KeygenSessionDatumQ().KeygenSessionDatumByID(int64(d.data.SessionId), false)
	if err != nil {
		ctx.Log().WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		ctx.Log().Error("Session entry is not initialized")
		return
	}

	session.Parties = partyAccounts(d.data.Set.Parties)
	session.Key = sql.NullString{
		String: d.data.NewSecret.GlobalPubKey(),
		Valid:  d.data.Processing,
	}

	if err = ctx.PG().KeygenSessionDatumQ().Update(session); err != nil {
		ctx.Log().WithError(err).Error("Error updating session entry")
	}
}

// finish sets up new secret in data (without storing it in secret store).
func (d *defaultKeygenController) finish(ctx core.Context, result *keygen.LocalPartySaveData) {
	d.data.NewSecret = ctx.SecretStorage().GetTssSecret().NewWithData(result)
	d.data.Processing = true
}

// reshareKeygenController represents custom logic for types.SessionType_ReshareSession
type reshareKeygenController struct {
	data *LocalSessionData
}

// Implements iKeygenController interface
var _ iKeygenController = &reshareKeygenController{}

// Next returns the key signature controller if self party is selected signer for current session.
// Otherwise, it will return finish controller instance.
// WaitFor should be called before.
func (r *reshareKeygenController) Next() IController {
	if r.data.Processing && r.data.IsSigner {
		return r.data.GetKeySignController()
	}

	return r.data.GetFinishController()
}

// updateSessionData updates the database entry according to the controller result.
func (r *reshareKeygenController) updateSessionData(ctx core.Context) {
	if !r.data.Processing {
		return
	}

	session, err := ctx.PG().ReshareSessionDatumQ().ReshareSessionDatumByID(int64(r.data.SessionId), false)
	if err != nil {
		ctx.Log().WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		ctx.Log().Error("Session entry is not initialized")
		return
	}

	session.NewKey = sql.NullString{
		String: r.data.NewSecret.GlobalPubKey(),
		Valid:  r.data.Processing,
	}

	if err = ctx.PG().ReshareSessionDatumQ().Update(session); err != nil {
		ctx.Log().WithError(err).Error("Error updating session data entry")
	}
}

// finish sets up new secret in data (without storing it in secret store) and calculates new parties ECDSA public keys.
func (r *reshareKeygenController) finish(ctx core.Context, result *keygen.LocalPartySaveData) {
	r.data.NewSecret = ctx.SecretStorage().GetTssSecret().NewWithData(result)
	r.data.NewParties = make([]*rarimo.Party, len(r.data.Set.Parties))

	partyIDs := core.PartyIds(r.data.Set.Parties)
	for i := range result.Ks {
		partyId := partyIDs.FindByKey(result.Ks[i])
		for j, party := range r.data.Set.Parties {
			if party.Account == partyId.Id {
				// Marshalled point contains constant 0x04 first byte, we have to remove it
				marshalled := elliptic.Marshal(eth.S256(), result.BigXj[i].X(), result.BigXj[i].Y())

				r.data.NewParties[j] = &rarimo.Party{
					PubKey:  hexutil.Encode(marshalled[1:]),
					Address: party.Address,
					Account: party.Account,
					Status:  rarimo.PartyStatus_Active,
				}
				break
			}
		}
	}
}
