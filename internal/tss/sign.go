package tss

import (
	"context"
	"math/big"
	"sync"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureResult struct {
	Signature string
	Data      string
}

type SignParty struct {
	wg *sync.WaitGroup

	con *connectors.BroadcastConnector
	log *logan.Entry

	set     *PartiesSetData
	party   tss.Party
	storage secret.Storage
	secret  *secret.TssSecret

	data   string
	id     uint64
	result SignatureResult
}

func NewSignParty(data string, id uint64, storage secret.Storage, secret *secret.TssSecret, set *PartiesSetData, con *connectors.BroadcastConnector, log *logan.Entry) *SignParty {
	return &SignParty{
		log:     log,
		wg:      &sync.WaitGroup{},
		set:     set,
		secret:  secret,
		con:     con,
		storage: storage,
		data:    data,
		id:      id,
	}
}

func (p *SignParty) Run(ctx context.Context) {
	out := make(chan tss.Message, 1000)
	end := make(chan common.SignatureData, 1)
	peerCtx := tss.NewPeerContext(p.set.Parties)
	partyId := p.set.Parties.FindByKey(p.storage.PartyKey())
	tssParams := tss.NewParameters(s256k1.S256(), peerCtx, partyId, p.set.N, p.set.N)

	p.party = signing.NewLocalParty(new(big.Int).SetBytes(hexutil.MustDecode(p.data)), tssParams, *p.secret.Data, out, end)
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

func (p *SignParty) Result() SignatureResult {
	return p.result
}

func (p *SignParty) Receive(sender rarimo.Party, isBroadcast bool, details []byte) {
	if p.party != nil {
		p.log.Infof("Received signing request from %s id: %s", sender.Account, p.id)
		_, data, _ := bech32.DecodeAndConvert(sender.Account)
		_, err := p.party.UpdateFromBytes(details, p.set.Parties.FindByKey(new(big.Int).SetBytes(data)), isBroadcast)
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

		p.result = SignatureResult{
			Signature: hexutil.Encode(append(result.Signature, result.SignatureRecovery...)),
			Data:      p.data,
		}

		p.log.Infof("Signed data %s signature %", p.data, p.result.Signature)
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

			request := &types.MsgSubmitRequest{
				Id:          p.id,
				Type:        types.RequestType_Sign,
				IsBroadcast: msg.IsBroadcast(),
				Details:     details,
			}

			p.log.Infof("Sending to %v", msg.GetTo())
			for _, to := range msg.GetTo() {
				p.log.Infof("Sending message to %s", to.Id)
				party, _ := Find(p.set.Set, to.Id)

				if party.Account == p.storage.AccountAddressStr() {
					p.log.Info("Sending to self")
					p.Receive(party, msg.IsBroadcast(), details.Value)
					continue
				}

				if failed := p.con.SubmitTo(ctx, request, &party); len(failed) != 0 {
					p.con.SubmitTo(ctx, request, &party)
				}
			}
		}
	}
}
