package tss

import (
	"context"
	"crypto/elliptic"
	"math/big"
	"sync"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
	s256k1 "github.com/btcsuite/btcd/btcec/v2"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rarimo/rarimo-core/x/rarimocore/crypto"
	rarimo "github.com/rarimo/rarimo-core/x/rarimocore/types"
	"github.com/rarimo/tss-svc/internal/connectors"
	"github.com/rarimo/tss-svc/internal/core"
	"github.com/rarimo/tss-svc/internal/secret"
	"github.com/rarimo/tss-svc/pkg/types"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/protobuf/types/known/anypb"
)

type KeygenParty struct {
	wg *sync.WaitGroup

	log *logan.Entry

	partyIds tss.SortedPartyIDs
	parties  map[string]*rarimo.Party
	secret   *secret.TssSecret

	party tss.Party
	con   *connectors.BroadcastConnector
	core  *connectors.CoreConnector

	id     uint64
	result *keygen.LocalPartySaveData

	waiting chan waitingMessage
}

func NewKeygenParty(id uint64, sessionType types.SessionType, parties []*rarimo.Party, secret *secret.TssSecret, coreCon *connectors.CoreConnector, log *logan.Entry) *KeygenParty {
	return &KeygenParty{
		id:       id,
		wg:       &sync.WaitGroup{},
		log:      log,
		partyIds: core.PartyIds(parties),
		parties:  partiesByAccountMapping(parties),
		secret:   secret,
		con:      connectors.NewBroadcastConnector(sessionType, parties, secret, log),
		core:     coreCon,
		waiting:  make(chan waitingMessage, WaitingCap),
	}
}

func (k *KeygenParty) Result() *keygen.LocalPartySaveData {
	return k.result
}

func (k *KeygenParty) Receive(sender *rarimo.Party, isBroadcast bool, details []byte) error {
	if k.party != nil {
		k.receiveWaiting()
		return k.receive(sender, isBroadcast, details)
	}

	k.pushToWaiting(sender, isBroadcast, details)

	return nil
}

func (k *KeygenParty) receive(sender *rarimo.Party, isBroadcast bool, details []byte) error {
	k.log.Debugf("Received keygen request from %s", sender.Account)
	_, data, _ := bech32.DecodeAndConvert(sender.Account)
	_, err := k.party.UpdateFromBytes(details, k.partyIds.FindByKey(new(big.Int).SetBytes(data)), isBroadcast)
	if err != nil {
		return err
	}
	logPartyStatus(k.log, k.party, k.secret.AccountAddress())
	k.log.Debugf("Finished processing keygen request from %s", sender.Account)

	return nil
}

func (k *KeygenParty) Run(ctx context.Context) {
	k.log.Infof("Running TSS key generation on set: %v", k.parties)
	self := k.partyIds.FindByKey(core.GetTssPartyKey(k.secret.AccountAddress()))
	out := make(chan tss.Message, OutChannelSize)
	end := make(chan *keygen.LocalPartySaveData, EndChannelSize)
	peerCtx := tss.NewPeerContext(k.partyIds)
	params := tss.NewParameters(tss.S256(), peerCtx, self, k.partyIds.Len(), crypto.GetThreshold(k.partyIds.Len()))

	tss.S256()
	k.party = k.secret.GetKeygenParty(params, out, end)
	go func() {
		err := k.party.Start()
		if err != nil {
			k.log.WithError(err).Error("Error running tss party")
			close(end)
		}
	}()

	k.wg.Add(2)
	go k.run(ctx, end)
	go k.listenOutput(ctx, out)
}

func (k *KeygenParty) WaitFor() {
	k.log.Debug("Waiting for finishing keygen party group")
	k.wg.Wait()
	k.log.Debug("Keygen party group finished")
}

func (k *KeygenParty) receiveWaiting() {
	if len(k.waiting) == 0 {
		return
	}

	k.log.Debug("Processing waiting messages")

	for {
		select {
		case msg := <-k.waiting:
			if err := k.receive(msg.sender, msg.isBroadcast, msg.details); err != nil {
				k.log.WithError(err).Error("failed to receive waiting message")
			}
		default:
			return
		}
	}
}

func (k *KeygenParty) pushToWaiting(sender *rarimo.Party, isBroadcast bool, details []byte) {
	k.log.Debug("Message will be pushed to the waiting queue")
	k.waiting <- waitingMessage{
		sender:      sender,
		isBroadcast: isBroadcast,
		details:     details,
	}
}

func (k *KeygenParty) run(ctx context.Context, end <-chan *keygen.LocalPartySaveData) {
	defer func() {
		k.log.Debug("Listening to keygen party result finished")
		k.wg.Done()
	}()

	<-ctx.Done()

	select {
	case result, ok := <-end:
		if !ok {
			k.log.Error("TSS party chanel closed")
			return
		}

		k.log.Infof("New generated public key: %s", hexutil.Encode(elliptic.Marshal(s256k1.S256(), result.ECDSAPub.X(), result.ECDSAPub.Y())))
		k.result = result
	default:
		k.log.Error("Keygen process has not been finished yet or has some errors")
	}
}

func (k *KeygenParty) listenOutput(ctx context.Context, out <-chan tss.Message) {
	defer func() {
		k.log.Debug("Listening to keygen party output finished")
		k.wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-out:
			details, err := anypb.New(msg.WireMsg().Message)
			if err != nil {
				k.log.WithError(err).Error("Failed to parse details")
				continue
			}

			request := &types.MsgSubmitRequest{
				Data: &types.RequestData{
					Type:        types.RequestType_Keygen,
					Id:          k.id,
					IsBroadcast: msg.IsBroadcast(),
					Details:     details,
				},
			}

			to := msg.GetTo()
			if msg.IsBroadcast() {
				to = k.partyIds
			}

			receivers := make([]*rarimo.Party, 0, len(to))

			for _, receiver := range to {
				party, _ := k.parties[receiver.Id]

				if party.Account == k.secret.AccountAddress() {
					k.log.Debugf("Sending to self (%s)", party.Account)
					if err := k.Receive(party, msg.IsBroadcast(), details.Value); err != nil {
						k.log.WithError(err).Error("error submitting request to self")
					}
					continue
				}

				receivers = append(receivers, party)
			}

			if failed := k.con.SubmitToWithReport(ctx, k.core, request, receivers...); len(failed) != 0 {
				k.con.SubmitToWithReport(ctx, k.core, request, failed...)
			}
		}
	}
}
