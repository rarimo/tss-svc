package tss

import (
	"context"
	"math/big"
	"sync"

	"github.com/bnb-chain/tss-lib/common"
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

type SignParty struct {
	wg *sync.WaitGroup

	log *logan.Entry

	partyIds tss.SortedPartyIDs
	parties  map[string]*rarimo.Party
	secret   *secret.TssSecret

	self  *tss.PartyID
	party tss.Party
	con   *connectors.BroadcastConnector

	data   string
	id     uint64
	result *common.SignatureData
}

func NewSignParty(data string, id uint64, parties []*rarimo.Party, secret *secret.TssSecret, log *logan.Entry) *SignParty {
	return &SignParty{
		wg:       &sync.WaitGroup{},
		log:      log,
		parties:  core.PartiesByAccountMapping(parties),
		partyIds: core.PartyIds(parties),
		secret:   secret,
		con:      connectors.NewBroadcastConnector(parties, secret, log),
		data:     data,
		id:       id,
	}
}

func (p *SignParty) Run(ctx context.Context) {
	p.log.Infof("Running TSS signing on set: %v", p.parties)
	p.self = p.partyIds.FindByKey(core.GetTssPartyKey(p.secret.AccountAddress()))
	out := make(chan tss.Message, 1000)
	end := make(chan common.SignatureData, 1)
	peerCtx := tss.NewPeerContext(p.partyIds)
	params := tss.NewParameters(s256k1.S256(), peerCtx, p.self, p.partyIds.Len(), crypto.GetThreshold(p.partyIds.Len()))
	p.party = p.secret.GetSignParty(new(big.Int).SetBytes(hexutil.MustDecode(p.data)), params, out, end)
	go func() {
		err := p.party.Start()
		if err != nil {
			p.log.WithError(err).Error("error running tss party")
			close(end)
		}
	}()

	p.wg.Add(2)
	go p.run(ctx, end)
	go p.listenOutput(ctx, out)
}

func (p *SignParty) WaitFor() {
	p.wg.Wait()
}

func (p *SignParty) Result() *common.SignatureData {
	return p.result
}

func (p *SignParty) Data() string {
	return p.data
}

func (p *SignParty) Receive(sender *rarimo.Party, isBroadcast bool, details []byte) {
	if p.party != nil {
		p.log.Infof("Received signing request from %s id: %d", sender.Account, p.id)
		_, data, _ := bech32.DecodeAndConvert(sender.Account)
		_, err := p.party.UpdateFromBytes(details, p.partyIds.FindByKey(new(big.Int).SetBytes(data)), isBroadcast)
		if err != nil {
			p.log.WithError(err).Debug("error updating party")
		}
	}
}

func (p *SignParty) run(ctx context.Context, end <-chan common.SignatureData) {
	defer p.wg.Done()

	<-ctx.Done()

	select {
	case result, ok := <-end:
		if !ok {
			p.log.Error("TSS party chanel closed")
			return
		}

		p.result = &result
		p.log.Infof("Signed data %s signature %s", p.data, hexutil.Encode(append(p.result.Signature, p.result.SignatureRecovery...)))
	default:
		p.log.Info("Signature process has not been finished yet or has some errors")
	}
}

func (p *SignParty) listenOutput(ctx context.Context, out <-chan tss.Message) {
	defer p.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-out:
			details, err := cosmostypes.NewAnyWithValue(msg.WireMsg().Message)
			if err != nil {
				p.log.WithError(err).Error("Failed to parse details")
				continue
			}

			sign, err := cosmostypes.NewAnyWithValue(&types.SignRequest{
				Data:    p.data,
				Details: details,
			})

			if err != nil {
				p.log.WithError(err).Error("Failed to parse sign")
				continue
			}

			request := &types.MsgSubmitRequest{
				Id:          p.id,
				Type:        types.RequestType_Sign,
				IsBroadcast: msg.IsBroadcast(),
				Details:     sign,
			}

			to := msg.GetTo()
			if msg.IsBroadcast() {
				to = p.partyIds
			}

			p.log.Infof("Sending to %v", to)
			for _, to := range to {
				p.log.Infof("Sending message to %s", to.Id)
				party, _ := p.parties[to.Id]

				if party.Account == p.secret.AccountAddress() {
					p.log.Info("Sending to self")
					p.Receive(party, msg.IsBroadcast(), details.Value)
					continue
				}

				if failed := p.con.SubmitTo(ctx, request, party); len(failed) != 0 {
					p.con.SubmitTo(ctx, request, party)
				}
			}
		}
	}
}
