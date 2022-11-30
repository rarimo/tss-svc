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
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type ReshareController struct {
	mu   sync.Mutex
	wg   *sync.WaitGroup
	data *LocalSessionData

	auth *core.RequestAuthorizer
	log  *logan.Entry

	party   *tss.ReshareParty
	storage secret.Storage
	pg      *pg.Storage
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
	r.log.Infof("Starting %s", r.Type().String())
	r.party.Run(ctx)
	r.wg.Add(1)
	go r.run(ctx)
}

func (r *ReshareController) WaitFor() {
	r.wg.Wait()
}

func (r *ReshareController) Next() IController {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.data.Processing {
		return r.factory.GetSignController(hexutil.Encode(eth.Keccak256(hexutil.MustDecode(r.data.NewGlobalPublicKey))), true)
	}

	return r.factory.GetFinishController()
}

func (r *ReshareController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_RESHARE
}

func (r *ReshareController) run(ctx context.Context) {
	defer func() {
		r.log.Infof("%s finished", r.Type().String())
		r.updateSessionData()
		r.wg.Done()
	}()

	<-ctx.Done()
	r.party.WaitFor()

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
	r.data.New.T = ((r.data.New.N + 2) / 3) * 2
	r.data.New.IsActive = true
	r.data.NewGlobalPublicKey = r.data.New.GlobalPubKey

	for i := range result.Ks {
		partyId := r.data.New.SortedPartyIDs.FindByKey(result.Ks[i])
		for j := range r.data.New.Parties {
			if r.data.New.Parties[j].Account == partyId.Id {
				r.data.New.Parties[j].PubKey = hexutil.Encode(elliptic.Marshal(eth.S256(), result.BigXj[i].X(), result.BigXj[i].Y()))
				break
			}
		}
	}
}

func (r *ReshareController) updateSessionData() {
	session, err := r.pg.SessionQ().SessionByID(int64(r.data.SessionId), false)
	if err != nil {
		r.log.WithError(err).Error("error selecting session")
		return
	}

	if session == nil {
		r.log.Error("session entry is not initialized")
		return
	}

	if r.data.SessionType == types.SessionType_ReshareSession {
		data, err := r.pg.ReshareSessionDatumQ().ReshareSessionDatumByID(session.DataID.Int64, false)
		if err != nil {
			r.log.WithError(err).Error("error selecting session data")
			return
		}

		if data == nil {
			r.log.Error("session data is not initialized")
			return
		}

		data.NewKey = sql.NullString{
			String: r.data.NewGlobalPublicKey,
			Valid:  r.data.NewGlobalPublicKey != "",
		}

		if err = r.pg.ReshareSessionDatumQ().Update(data); err != nil {
			r.log.WithError(err).Error("error updating session data entry")
		}
	}
}
