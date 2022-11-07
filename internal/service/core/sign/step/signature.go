package step

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"sync"

	"github.com/binance-chain/tss-lib/ecdsa/signing"
	"github.com/binance-chain/tss-lib/tss"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
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
	end       chan *signing.SignatureData

	index map[string]struct{}
	si    chan *big.Int

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
		end:       make(chan *signing.SignatureData, 1000),
		si:        make(chan *big.Int, params.N()),
		index:     make(map[string]struct{}),
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

		if _, ok := s.index[sender.Account]; !ok && sign.Root == s.root {
			s.log.Infof("Received message from %s", sender.Account)
			s.index[sender.Account] = struct{}{}
			s.si <- new(big.Int).SetBytes(sign.Si)
		}
	}

}

func (s *SignatureController) Run(ctx context.Context) {
	s.party = signing.NewLocalParty(nil, s.tssParams, *s.secret.GetLocalPartyData(), make(chan tss.Message, 1000), s.end)
	go func() {
		err := s.party.Start()
		if err != nil {
			s.log.WithError(err).Error("error starting tss party")
		}
	}()

	s.wg.Add(1)
	go s.run(ctx)
}

func (s *SignatureController) run(ctx context.Context) {
	defer s.wg.Done()
	preResult := <-s.end

	msg := new(big.Int).SetBytes(hexutil.MustDecode(s.root))
	si := signing.FinalizeGetOurSigShare(preResult, msg)
	details, err := cosmostypes.NewAnyWithValue(
		&types.SignRequest{
			Root: s.root,
			Si:   si.Bytes(),
		},
	)

	if err != nil {
		s.log.WithError(err).Error("error parsing details")
		return
	}

	s.connector.SubmitAll(ctx, &types.MsgSubmitRequest{
		Type:        types.RequestType_Sign,
		IsBroadcast: true,
		Details:     details,
	})

	<-ctx.Done()

	shared := make(map[*tss.PartyID]*big.Int)
	shared[s.secret.PartyId()] = si

	point := s.secret.GetLocalPartyData().ECDSAPub
	key := &ecdsa.PublicKey{
		Curve: secp256k1.S256(),
		X:     point.X(),
		Y:     point.Y(),
	}

	data, _, err := signing.FinalizeGetAndVerifyFinalSig(preResult, key, msg, s.secret.PartyId(), si, shared)
	if err != nil {
		s.log.WithError(err).Error("error finalizing signature")
		return
	}

	signature := append(data.Signature.Signature, data.Signature.SignatureRecovery...)
	s.log.Infof("[Signing %d] - Signed root %s signature %s", s.id, s.root, hexutil.Encode(signature))

	s.result <- &session.Signature{
		Signed:    []string{s.secret.ECDSAPubKeyStr()},
		Signature: hexutil.Encode(signature),
	}

	s.log.Infof("[Signing %d] - Controller finished", s.id)
}

func (s *SignatureController) WaitFinish() {
	s.wg.Wait()
}
