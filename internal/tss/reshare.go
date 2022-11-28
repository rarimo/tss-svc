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
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type ReshareParty struct {
	wg *sync.WaitGroup

	log *logan.Entry

	old   *core.InputSet
	new   *core.InputSet
	party tss.Party
	con   *connectors.BroadcastConnector

	id     uint64
	result *keygen.LocalPartySaveData
}

func NewReshareParty(id uint64, old, new *core.InputSet, log *logan.Entry) *ReshareParty {
	return &ReshareParty{
		id:  id,
		wg:  &sync.WaitGroup{},
		log: log,
		old: old,
		new: new,
		con: connectors.NewBroadcastConnector(new, log),
	}
}

func (r *ReshareParty) Result() *keygen.LocalPartySaveData {
	return r.result
}

func (r *ReshareParty) Receive(sender rarimo.Party, isBroadcast bool, details []byte) {
	r.log.Infof("Received reshare request from %s ", sender.Account)
	_, data, _ := bech32.DecodeAndConvert(sender.Account)
	_, err := r.party.UpdateFromBytes(details, r.new.SortedPartyIDs.FindByKey(new(big.Int).SetBytes(data)), isBroadcast)
	if err != nil {
		r.log.WithError(err).Debug("error updating party")
	}
}

func (r *ReshareParty) Run(ctx context.Context) {
	r.log.Infof("Running TSS key re-generation on new set: %v", r.new.Parties)
	out := make(chan tss.Message, 1000)
	end := make(chan keygen.LocalPartySaveData, 1)

	if r.new.LocalTss.LocalData == nil {
		empty := keygen.NewLocalPartySaveData(r.new.N)
		r.new.LocalTss.LocalData = &empty
	}

	params := tss.NewReSharingParameters(s256k1.S256(), tss.NewPeerContext(r.old.SortedPartyIDs), tss.NewPeerContext(r.new.SortedPartyIDs), r.new.LocalParty(), r.old.N, r.old.T, r.new.N, r.new.T)
	r.party = resharing.NewLocalParty(params, *r.new.LocalTss.LocalData, out, end)

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
		r.result = &result
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
				Id:          r.id,
				Type:        types.RequestType_Reshare,
				IsBroadcast: msg.IsBroadcast(),
				Details:     details,
			}

			r.log.Infof("Sending to %v", msg.GetTo())
			for _, to := range msg.GetTo() {
				r.log.Infof("Sending message to %s", to.Id)
				party, _ := r.new.PartyByAccount(to.Id)
				if party.Account == r.new.LocalAccountAddress {
					r.log.Info("Sending to self")
					r.Receive(party, msg.IsBroadcast(), request.Details.Value)
					continue
				}

				r.con.MustSubmitTo(ctx, request, &party)
			}
		}
	}
}
