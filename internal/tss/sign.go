package tss

import (
	"context"
	"math/big"
	"sync"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/rarimo-core/x/rarimocore/crypto"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/connectors"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
	"google.golang.org/protobuf/types/known/anypb"
)

type SignParty struct {
	wg *sync.WaitGroup

	log *logan.Entry

	partyIds tss.SortedPartyIDs
	parties  map[string]*rarimo.Party
	secret   *secret.TssSecret

	party tss.Party
	con   *connectors.BroadcastConnector
	core  *connectors.CoreConnector

	data   string
	id     uint64
	result *common.SignatureData
}

func NewSignParty(data string, id uint64, sessionType types.SessionType, parties []*rarimo.Party, secret *secret.TssSecret, coreCon *connectors.CoreConnector, log *logan.Entry) *SignParty {
	return &SignParty{
		wg:       &sync.WaitGroup{},
		log:      log,
		parties:  partiesByAccountMapping(parties),
		partyIds: core.PartyIds(parties),
		secret:   secret,
		con:      connectors.NewBroadcastConnector(sessionType, parties, secret, log),
		core:     coreCon,
		data:     data,
		id:       id,
	}
}

func (p *SignParty) Run(ctx context.Context) {
	p.log.Infof("Running TSS signing on set: %v", p.parties)
	self := p.partyIds.FindByKey(core.GetTssPartyKey(p.secret.AccountAddress()))
	out := make(chan tss.Message, OutChannelSize)
	end := make(chan common.SignatureData, EndChannelSize)
	peerCtx := tss.NewPeerContext(p.partyIds)
	params := tss.NewParameters(s256k1.S256(), peerCtx, self, p.partyIds.Len(), crypto.GetThreshold(p.partyIds.Len()))
	p.party = p.secret.GetSignParty(new(big.Int).SetBytes(hexutil.MustDecode(p.data)), params, out, end)
	go func() {
		err := p.party.Start()
		if err != nil {
			p.log.WithError(err).Error("Error running tss party")
			close(end)
		}
	}()

	p.wg.Add(2)
	go p.run(ctx, end)
	go p.listenOutput(ctx, out)
}

func (p *SignParty) WaitFor() {
	p.log.Debug("Waiting for finishing sign party group")
	p.wg.Wait()
	p.log.Debug("Sign party group finished")
}

func (p *SignParty) Result() *common.SignatureData {
	return p.result
}

func (p *SignParty) Data() string {
	return p.data
}

func (p *SignParty) Receive(sender *rarimo.Party, isBroadcast bool, details []byte) error {
	if p.party != nil {
		p.log.Debugf("Processing signing request from %s", sender.Account)
		_, data, _ := bech32.DecodeAndConvert(sender.Account)
		_, err := p.party.UpdateFromBytes(details, p.partyIds.FindByKey(new(big.Int).SetBytes(data)), isBroadcast)
		if err != nil {
			return err
		}
		logPartyStatus(p.log, p.party, p.secret.AccountAddress())
		p.log.Debugf("Finished processing signing request from %s", sender.Account)
	}

	return nil
}

func (p *SignParty) run(ctx context.Context, end <-chan common.SignatureData) {
	defer func() {
		p.log.Debug("Listening to sign party result finished")
		p.wg.Done()
	}()

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
		p.log.Error("Signature process has not been finished yet or has some errors")
	}
}

func (p *SignParty) listenOutput(ctx context.Context, out <-chan tss.Message) {
	defer func() {
		p.log.Debug("Listening to sign party output finished")
		p.wg.Done()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-out:
			details, err := anypb.New(msg.WireMsg().Message)
			if err != nil {
				p.log.WithError(err).Error("Failed to parse details")
				continue
			}

			sign, err := anypb.New(&types.SignRequest{
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

			receivers := make([]*rarimo.Party, 0, len(to))

			for _, receiver := range to {
				party, _ := p.parties[receiver.Id]

				if party.Account == p.secret.AccountAddress() {
					p.log.Debugf("Sending to self (%s)", party.Account)
					go func() {
						if err := p.Receive(party, msg.IsBroadcast(), request.Details.Value); err != nil {
							p.log.WithError(err).Error("error submitting request to self")
						}
					}()

					continue
				}

				receivers = append(receivers, party)
			}

			go func() {
				if failed := p.con.SubmitToWithReport(ctx, p.core, request, receivers...); len(failed) != 0 {
					p.con.SubmitToWithReport(ctx, p.core, request, failed...)
				}
			}()
		}
	}
}
