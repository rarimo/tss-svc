package step

import (
	"context"
	"math/big"
	"sync"

	"github.com/binance-chain/tss-lib/ecdsa/signing"
	"github.com/binance-chain/tss-lib/tss"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureController struct {
	wg          *sync.WaitGroup
	id          uint64
	root        string
	acceptances map[string]struct{}
	parties     tss.SortedPartyIDs

	tssParams *tss.Parameters
	party     tss.Party
	end       chan *signing.SignatureData
	out       chan tss.Message
	result    chan *session.Signature

	params    *local.Params
	secret    *local.Secret
	connector *connectors.BroadcastConnector

	log *logan.Entry
}

func NewSignatureController(
	id uint64,
	root string,
	acceptances []string,
	params *local.Params,
	secret *local.Secret,
	result chan *session.Signature,
	log *logan.Entry,
	connector *connectors.BroadcastConnector,
) *SignatureController {
	parties := params.PartyIds()
	localId := parties.FindByKey(secret.PartyId().KeyInt())
	peerCtx := tss.NewPeerContext(parties)
	tssParams := tss.NewParameters(peerCtx, localId, params.N(), params.T())

	amap := make(map[string]struct{})
	for _, acc := range acceptances {
		amap[acc] = struct{}{}
	}

	return &SignatureController{
		wg:          &sync.WaitGroup{},
		id:          id,
		root:        root,
		acceptances: amap,
		parties:     parties,
		out:         make(chan tss.Message, 1000),
		end:         make(chan *signing.SignatureData, 1000),
		tssParams:   tssParams,
		params:      params,
		secret:      secret,
		result:      result,
		connector:   connector,
		log:         log,
	}
}

var _ IController = &SignatureController{}

func (s *SignatureController) ReceiveFromSender(sender rarimo.Party, request *types.MsgSubmitRequest) {
	if _, ok := s.acceptances[sender.Account]; ok && request.Type == types.RequestType_Sign {
		sign := new(types.SignRequest)
		if err := proto.Unmarshal(request.Details.Value, sign); err != nil {
			s.log.WithError(err).Error("error unmarshalling request")
		}

		if sign.Root == s.root {
			s.log.Infof("[Signing %d] - Received signing request from %s for root %s ---", s.id, sender.Account, s.root)
			_, data, _ := bech32.DecodeAndConvert(sender.Account)
			_, err := s.party.UpdateFromBytes(sign.Details.Value, s.parties.FindByKey(new(big.Int).SetBytes(data)), request.IsBroadcast)
			if err != nil {
				s.log.WithError(err).Error("error updating party")
			}
		}
	}
}

func (s *SignatureController) Run(ctx context.Context) {
	s.party = signing.NewLocalParty(new(big.Int).SetBytes(hexutil.MustDecode(s.root)), s.tssParams, *s.secret.GetLocalPartyData(), s.out, s.end)
	go func() {
		err := s.party.Start()
		if err != nil {
			s.log.WithError(err).Error("error starting tss party")
			close(s.end)
		}
	}()

	s.wg.Add(2)
	go s.run(ctx)
	go s.listenOutput(ctx, s.out)
}

func (s *SignatureController) run(ctx context.Context) {
	defer func() {
		s.log.Infof("[Signing %d] - Controller finished", s.id)
		s.wg.Done()
	}()

	<-ctx.Done()

	select {
	case result, ok := <-s.end:
		if !ok {
			s.log.Error("TSS Party chanel closed")
			return
		}

		signature := append(result.Signature.Signature, result.Signature.SignatureRecovery...)
		s.log.Infof("[Signing %d] - Signed root %s signature %s", s.id, s.root, hexutil.Encode(signature))

		s.party.WaitingFor()
		s.result <- &session.Signature{
			// TODO fixme
			Signed:    []string{s.secret.ECDSAPubKeyStr()},
			Signature: hexutil.Encode(signature),
		}
	default:
		s.log.Infof("[Signing %d] Signature process has not been finished yet or has some errors", s.id)
	}
}

func (s *SignatureController) listenOutput(ctx context.Context, out <-chan tss.Message) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-out:
			details, err := cosmostypes.NewAnyWithValue(msg.WireMsg().Message)
			if err != nil {
				s.log.WithError(err).Error("failed to parse details")
				continue
			}

			sign := &types.SignRequest{
				Root:    s.root,
				Details: details,
			}

			details, err = cosmostypes.NewAnyWithValue(sign)
			if err != nil {
				s.log.WithError(err).Error("failed to parse sign request to details")
				continue
			}

			request := &types.MsgSubmitRequest{
				Type:        types.RequestType_Sign,
				IsBroadcast: msg.IsBroadcast(),
				Details:     details,
			}

			toParties := msg.GetTo()
			if msg.IsBroadcast() {
				toParties = s.params.PartyIds()
			}

			s.log.Infof("[Signing %d] Sending to %v", s.id, toParties)
			for _, to := range toParties {
				s.log.Infof("[Signing %d] Sending message to %s", s.id, to.Id)
				party, _ := s.params.PartyByAccount(to.Id)

				if party.Account == s.secret.AccountAddressStr() {
					s.log.Infof("[Signing %d] Sending to self", s.id)
					s.ReceiveFromSender(party, request)
					continue
				}

				if failed := s.connector.SubmitTo(ctx, request, &party); len(failed) != 0 {
					// Retry
					s.connector.SubmitTo(ctx, request, &party)
				}
			}
		}
	}
}

func (s *SignatureController) WaitFinish() {
	s.wg.Wait()
}
