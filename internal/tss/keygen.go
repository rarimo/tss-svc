package tss

import (
	"context"
	"crypto/elliptic"
	"math/big"
	"sync"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/crypto"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type KeygenParty struct {
	wg *sync.WaitGroup

	log *logan.Entry

	partyIds tss.SortedPartyIDs
	parties  map[string]*rarimo.Party
	secret   *secret.TssSecret

	party tss.Party
	con   *connectors.BroadcastConnector

	id     uint64
	result *keygen.LocalPartySaveData
}

func NewKeygenParty(id uint64, parties []*rarimo.Party, secret *secret.TssSecret, log *logan.Entry) *KeygenParty {
	return &KeygenParty{
		id:       id,
		wg:       &sync.WaitGroup{},
		log:      log,
		partyIds: core.PartyIds(parties),
		parties:  core.PartiesByAccountMapping(parties),
		secret:   secret,
		con:      connectors.NewBroadcastConnector(parties, secret, log),
	}
}

func (k *KeygenParty) Result() *keygen.LocalPartySaveData {
	return k.result
}

func (k *KeygenParty) Receive(sender *rarimo.Party, isBroadcast bool, details []byte) {
	k.log.Infof("Received keygen request from %s", sender.Account)
	_, data, _ := bech32.DecodeAndConvert(sender.Account)
	_, err := k.party.UpdateFromBytes(details, k.partyIds.FindByKey(new(big.Int).SetBytes(data)), isBroadcast)
	if err != nil {
		k.log.WithError(err).Debug("error updating party")
	}
}

func (k *KeygenParty) Run(ctx context.Context) {
	k.log.Infof("Running TSS key generation on set: %v", k.parties)
	self := k.partyIds.FindByKey(core.GetTssPartyKey(k.secret.AccountAddress()))
	out := make(chan tss.Message, 1000)
	end := make(chan keygen.LocalPartySaveData, 1)
	peerCtx := tss.NewPeerContext(k.partyIds)
	params := tss.NewParameters(s256k1.S256(), peerCtx, self, k.partyIds.Len(), crypto.GetThreshold(k.partyIds.Len()))

	k.party = k.secret.GetKeygenParty(params, out, end)
	go func() {
		err := k.party.Start()
		if err != nil {
			k.log.WithError(err).Error("error running tss party")
			close(end)
		}
	}()

	k.wg.Add(2)
	go k.run(ctx, end)
	go k.listenOutput(ctx, out)
}

func (k *KeygenParty) WaitFor() {
	k.wg.Wait()
}

func (k *KeygenParty) run(ctx context.Context, end <-chan keygen.LocalPartySaveData) {
	defer k.wg.Done()

	<-ctx.Done()

	select {
	case result, ok := <-end:
		if !ok {
			k.log.Error("TSS party chanel closed")
			return
		}

		k.log.Infof("Pub key: %s", hexutil.Encode(elliptic.Marshal(s256k1.S256(), result.ECDSAPub.X(), result.ECDSAPub.Y())))
		k.result = &result
	default:
		k.log.Info("Reshare process has not been finished yet or has some errors")
	}
}

func (k *KeygenParty) listenOutput(ctx context.Context, out <-chan tss.Message) {
	defer k.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-out:
			details, err := cosmostypes.NewAnyWithValue(msg.WireMsg().Message)
			if err != nil {
				k.log.WithError(err).Error("failed to parse details")
				continue
			}

			request := &types.MsgSubmitRequest{
				Id:          k.id,
				Type:        types.RequestType_Keygen,
				IsBroadcast: msg.IsBroadcast(),
				Details:     details,
			}

			to := msg.GetTo()
			if msg.IsBroadcast() {
				to = k.partyIds
			}

			k.log.Infof("Sending to %v", to)
			for _, to := range to {
				k.log.Infof("Sending message to %s", to.Id)
				party, _ := k.parties[to.Id]

				if party.Account == k.secret.AccountAddress() {
					k.log.Info("Sending to self")
					k.Receive(party, msg.IsBroadcast(), request.Details.Value)
					continue
				}

				k.con.MustSubmitTo(ctx, request, party)
			}
		}
	}
}
