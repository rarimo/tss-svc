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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureController struct {
	wg   *sync.WaitGroup
	id   uint64
	root string

	tssParams *tss.Parameters
	party     tss.Party
	out       chan tss.Message
	end       chan *signing.SignatureData

	result chan *session.Signature

	params    *local.Params
	secret    *local.Secret
	connector *connectors.BroadcastConnector

	log *logan.Entry
}

func NewSignatureController(
	id uint64,
	root string,
	params *local.Params,
	secret *local.Secret,
	result chan *session.Signature,
	log *logan.Entry,
	connector *connectors.BroadcastConnector,
) *SignatureController {
	parties := params.PartyIds()
	localId := parties.FindByKey(secret.PartyId().KeyInt())
	peerCtx := tss.NewPeerContext(parties)
	tssParams := tss.NewParameters(peerCtx, localId, len(parties), params.T())

	return &SignatureController{
		wg:        &sync.WaitGroup{},
		id:        id,
		root:      root,
		out:       make(chan tss.Message, 1000),
		end:       make(chan *signing.SignatureData, 1000),
		tssParams: tssParams,
		params:    params,
		secret:    secret,
		result:    result,
		connector: connector,
		log:       log,
	}
}

var _ IController = &SignatureController{}

func (s *SignatureController) ReceiveFromSender(sender rarimo.Party, request *types.MsgSubmitRequest) {
	if request.Type == types.RequestType_Sign {
		sign := new(types.SignRequest)

		if err := proto.Unmarshal(request.Details.Value, sign); err != nil {
			s.log.WithError(err).Error("error unmarshalling request")
		}

		if sign.Root == s.root {
			s.log.Infof("Received message from %s", sender.Account)
			_, data, _ := bech32.DecodeAndConvert(sender.Account)
			partyId := s.tssParams.Parties().IDs().FindByKey(new(big.Int).SetBytes(data))
			_, err := s.party.UpdateFromBytes(request.Details.Value, partyId, request.IsBroadcast)
			if err != nil {
				s.log.WithError(err).Error("error updating party")
			}
		}
	}

}

func (s *SignatureController) Run(ctx context.Context) {
	s.party = signing.NewLocalParty(nil, s.tssParams, *s.secret.GetLocalPartyData(), s.out, s.end)
	go func() {
		err := s.party.Start()
		if err != nil {
			s.log.WithError(err).Error("error starting tss party")
		}
	}()

	s.wg.Add(2)
	go s.run(ctx)
	go s.listenOutput(ctx, s.out)
}

// TODO mocked for one party
func (s *SignatureController) run(ctx context.Context) {
	signature, err := crypto.Sign(hexutil.MustDecode(s.root), s.secret.ECDSAPrvKey())
	if err != nil {
		s.log.WithError(err).Error("error signing root hash")
		return
	}

	s.log.Infof("[Signing %d] - Signed root %s signature %s", s.id, s.root, hexutil.Encode(signature))

	s.result <- &session.Signature{
		Signed:    []string{s.secret.ECDSAPubKeyStr()},
		Signature: hexutil.Encode(signature),
	}

	s.log.Infof("[Signing %d] - Controller finished", s.id)
	s.wg.Done()
}

func (s *SignatureController) WaitFinish() {
	s.wg.Wait()
}

func (s *SignatureController) listenOutput(ctx context.Context, out <-chan tss.Message) {
	for {
		select {
		case <-ctx.Done():
			s.wg.Done()
			return
		case msg := <-out:
			details, err := cosmostypes.NewAnyWithValue(msg.WireMsg().Message)
			if err != nil {
				s.log.WithError(err).Error("failed to parse details")
				continue
			}

			request := &types.MsgSubmitRequest{
				Type:        types.RequestType_Keygen,
				IsBroadcast: msg.IsBroadcast(),
				Details:     details,
			}

			toParties := msg.GetTo()
			if msg.IsBroadcast() {
				toParties = s.params.PartyIds()
			}

			s.log.Infof("Sending to %v", toParties)

			for _, to := range toParties {
				s.log.Infof("Sending message to %s", to.Id)
				party, _ := s.params.PartyByAccount(to.Id)

				if party.Account == s.secret.AccountAddressStr() {
					s.log.Info("Sending to self")
					s.ReceiveFromSender(party, request)
					continue
				}

				failed := s.connector.SubmitTo(ctx, request, &party)
				for _, f := range failed {
					s.log.Error("failed submitting to party %s", f.Account)
				}
			}
		}
	}
}
