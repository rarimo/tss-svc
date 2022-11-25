package tss

import (
	"context"
	"crypto/elliptic"
	"math/big"
	"sync"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/ecdsa/resharing"
	"github.com/bnb-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/params"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type PartiesSetData struct {
	N       int
	T       int
	Set     []*rarimo.Party
	Parties tss.SortedPartyIDs
}

func NewPartiesSetData(set []*rarimo.Party) *PartiesSetData {
	return &PartiesSetData{
		N:       len(set),
		T:       T(len(set)),
		Parties: PartyIds(set),
	}
}

type ReshareParty struct {
	wg *sync.WaitGroup

	con    *connectors.BroadcastConnector
	params *params.Params
	log    *logan.Entry

	old *PartiesSetData
	new *PartiesSetData

	storage secret.Storage
	secret  *secret.TssSecret
	party   tss.Party
	result  bool
}

func NewReshareParty(old, new *PartiesSetData, storage secret.Storage, secret *secret.TssSecret, params *params.Params, con *connectors.BroadcastConnector, log *logan.Entry) *ReshareParty {
	return &ReshareParty{
		wg:      &sync.WaitGroup{},
		con:     con,
		params:  params,
		log:     log,
		old:     old,
		new:     new,
		storage: storage,
		secret:  secret,
	}
}

func (r *ReshareParty) Result() bool {
	return r.result
}

func (r *ReshareParty) Receive(sender rarimo.Party, isBroadcast bool, details []byte) {
	r.log.Infof("Received reshare request from %s ", sender.Account)
	_, data, _ := bech32.DecodeAndConvert(sender.Account)
	_, err := r.party.UpdateFromBytes(details, r.new.Parties.FindByKey(new(big.Int).SetBytes(data)), isBroadcast)
	if err != nil {
		r.log.WithError(err).Debug("error updating party")
	}
}

func (r *ReshareParty) Run(ctx context.Context) {
	out := make(chan tss.Message, 1000)
	end := make(chan keygen.LocalPartySaveData, 1)
	localId := r.new.Parties.FindByKey(r.storage.PartyKey())

	data := r.secret.Data
	if data == nil {
		empty := keygen.NewLocalPartySaveData(r.old.N)
		data = &empty
		data.LocalPreParams = *r.secret.Params
	}

	params := tss.NewReSharingParameters(s256k1.S256(), tss.NewPeerContext(r.old.Parties), tss.NewPeerContext(r.new.Parties), localId, r.old.N, r.old.T, r.new.N, r.new.T)
	r.party = resharing.NewLocalParty(params, *data, out, end)

	go func() {
		err := r.party.Start()
		if err != nil {
			panic(err)
		}
	}()

	r.wg.Add(2)
	go r.run(ctx, end)
	go r.listenOutput(ctx, out)
}

func (r *ReshareParty) WaitFor() {
	defer r.wg.Wait()
}

func (r *ReshareParty) run(ctx context.Context, end <-chan keygen.LocalPartySaveData) {
	defer r.wg.Done()

	<-ctx.Done()

	select {
	case result, ok := <-end:
		if !ok {
			r.log.Error("TSS party chanel closed")
			return
		}

		r.log.Infof("Pub key: %s", hexutil.Encode(elliptic.Marshal(s256k1.S256(), result.ECDSAPub.X(), result.ECDSAPub.Y())))
		if err := r.storage.SetTssSecret(secret.NewTssSecret(&result, r.secret.Params, r.secret)); err != nil {
			r.log.WithError(err).Error("failed to set tss params")
		}
		r.result = true
	default:
		r.log.Info("Reshare process has not been finished yet or has some errors")
	}
}

func (r *ReshareParty) listenOutput(ctx context.Context, out <-chan tss.Message) {
	defer r.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-out:
			details, err := cosmostypes.NewAnyWithValue(msg.WireMsg().Message)
			if err != nil {
				r.log.WithError(err).Error("Failed to parse details")
				continue
			}

			request := &types.MsgSubmitRequest{
				Type:        types.RequestType_Reshare,
				IsBroadcast: msg.IsBroadcast(),
				Details:     details,
			}

			r.log.Infof("Sending to %v", msg.GetTo())
			for _, to := range msg.GetTo() {
				r.log.Infof("Sending message to %s", to.Id)
				party, _ := Find(r.new.Set, to.Id)

				if party.Account == r.storage.AccountAddressStr() {
					r.log.Info("Sending to self")
					r.Receive(party, msg.IsBroadcast(), request.Details.Value)
					continue
				}

				r.con.MustSubmitTo(ctx, request, &party)
			}
		}
	}
}
