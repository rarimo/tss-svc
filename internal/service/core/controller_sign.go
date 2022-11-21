package core

import (
	"context"
	goerr "errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/bnb-chain/tss-lib/common"
	"github.com/bnb-chain/tss-lib/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3/errors"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

var (
	ErrSenderHasNotAccepted = goerr.New("sender has not accepted proposal")
)

type SignatureController struct {
	*defaultController
	*bounds

	mu *sync.Mutex
	wg *sync.WaitGroup

	data      AcceptanceData
	parties   tss.SortedPartyIDs
	tssParams *tss.Parameters
	end       chan common.SignatureData
	out       chan tss.Message

	index map[string]uint

	result  SignatureData
	party   tss.Party
	factory *ControllerFactory
}

func NewSignatureController(
	data AcceptanceData,
	bounds *bounds,
	defaultController *defaultController,
	factory *ControllerFactory,
) *SignatureController {
	parties := defaultController.params.PartyIds()
	localId := parties.FindByKey(defaultController.secret.PartyId().KeyInt())
	peerCtx := tss.NewPeerContext(parties)
	tssParams := tss.NewParameters(s256k1.S256(), peerCtx, localId, defaultController.params.N(), defaultController.params.T())

	return &SignatureController{
		defaultController: defaultController,
		bounds:            bounds,
		mu:                &sync.Mutex{},
		wg:                &sync.WaitGroup{},
		data:              data,
		parties:           parties,
		tssParams:         tssParams,
		out:               make(chan tss.Message, 1000),
		end:               make(chan common.SignatureData, 1000),
		index:             make(map[string]uint),
		result: SignatureData{
			Indexes: data.Indexes,
			Root:    data.Root,
			Reshare: data.Reshare,
		},
		factory: factory,
	}
}

var _ IController = &SignatureController{}

func (s *SignatureController) StepType() types.StepType {
	return types.StepType_Signing
}

func (s *SignatureController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := s.auth.Auth(request)
	if err != nil {
		return err
	}

	return s.receive(sender, request)
}

func (s *SignatureController) receive(sender rarimo.Party, request *types.MsgSubmitRequest) error {
	if _, ok := s.data.Acceptances[sender.Address]; !ok {
		return ErrSenderHasNotAccepted
	}

	if request.Type != types.RequestType_Sign {
		return ErrInvalidRequestType
	}

	sign := new(types.SignRequest)
	if err := proto.Unmarshal(request.Details.Value, sign); err != nil {
		return errors.Wrap(err, "error unmarshalling request")
	}

	if s.party != nil && sign.Root == s.data.Root {
		s.infof("Received signing request from %s for root %s", sender.Account, sign.Root)
		s.index[sender.Account]++
		_, data, _ := bech32.DecodeAndConvert(sender.Account)
		_, err := s.party.UpdateFromBytes(sign.Details.Value, s.parties.FindByKey(new(big.Int).SetBytes(data)), request.IsBroadcast)
		if err != nil {
			return errors.Wrap(err, "error updating party")
		}
	}

	return nil
}

func (s *SignatureController) Run(ctx context.Context) {
	s.party = signing.NewLocalParty(new(big.Int).SetBytes(hexutil.MustDecode(s.data.Root)), s.tssParams, *s.secret.MustGetLocalPartyData(), s.out, s.end)
	go func() {
		err := s.party.Start()
		if err != nil {
			s.errorf(err, "error starting tss party")
			close(s.end)
		}
	}()

	s.wg.Add(2)
	go s.run(ctx)
	go s.listenOutput(ctx, s.out)
}

func (s *SignatureController) WaitFor() {
	s.wg.Wait()
}

func (s *SignatureController) Next() IController {
	fBounds := NewBounds(s.End()+1, s.params.Step(FinishingIndex).Duration)
	return s.factory.GetFinishController(s.result, fBounds)
}

func (s *SignatureController) run(ctx context.Context) {
	defer func() {
		s.infof("Controller finished")
		s.wg.Done()
	}()

	<-ctx.Done()

	select {
	case result, ok := <-s.end:
		if !ok {
			s.errorf(nil, "TSS Party chanel closed")
			return
		}

		signature := append(result.Signature, result.SignatureRecovery...)
		s.result.Signature = hexutil.Encode(signature)
		s.infof("Signed root %s signature %", s.data.Root, s.result.Signature)
		s.UpdateSignature(s.result)
	default:
		s.infof("Signature process has not been finished yet or has some errors")
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
				s.errorf(err, "Failed to parse details")
				continue
			}

			sign := &types.SignRequest{
				Root:    s.data.Root,
				Details: details,
			}

			details, err = cosmostypes.NewAnyWithValue(sign)
			if err != nil {
				s.errorf(err, "Failed to parse sign request to details")
				continue
			}

			request := &types.MsgSubmitRequest{
				Type:        types.RequestType_Sign,
				IsBroadcast: msg.IsBroadcast(),
				Details:     details,
			}

			s.infof("Sending to %v", msg.GetTo())
			for _, to := range msg.GetTo() {
				s.infof("Sending message to %s", to.Id)
				party, _ := s.params.PartyByAccount(to.Id)

				if party.Account == s.secret.AccountAddressStr() {
					s.infof("Sending to self")
					if err := s.receive(party, request); err != nil {
						s.errorf(err, "Failed to update self")
					}
					continue
				}

				if failed := s.SubmitTo(ctx, request, &party); len(failed) != 0 {
					s.SubmitTo(ctx, request, &party)
				}
			}
		}
	}
}

func (s *SignatureController) infof(msg string, args ...interface{}) {
	s.Infof("[Proposal %d] - %s", s.SessionID(), fmt.Sprintf(msg, args))
}

func (s *SignatureController) errorf(err error, msg string, args ...interface{}) {
	s.WithError(err).Errorf("[Proposal %d] - %s", s.SessionID(), fmt.Sprintf(msg, args))
}
