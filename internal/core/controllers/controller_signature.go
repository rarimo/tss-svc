package controllers

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/crypto/pkg"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureController struct {
	mu          *sync.Mutex
	data        *LocalSessionData
	isKeySigner bool

	auth *core.RequestAuthorizer
	log  *logan.Entry

	party   *tss.SignParty
	factory *ControllerFactory
}

var _ IController = &SignatureController{}

func (s *SignatureController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := s.auth.Auth(request)
	if err != nil {
		return err
	}

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

	if sign.Data == s.party.Data() {
		s.party.Receive(sender, request.IsBroadcast, sign.Details.Value)
	}
	return nil
}

func (s *SignatureController) Run(ctx context.Context) {
	s.party.Run(ctx)

	<-ctx.Done()

	s.mu.Lock()
	defer s.mu.Unlock()

	result := s.party.Result()
	if result == nil {
		s.data.Processing = false
		return
	}

	signature := hexutil.Encode(append(result.Signature, result.SignatureRecovery...))
	if s.isKeySigner {
		s.data.KeySignature = signature
	}
	s.data.OperationSignature = signature
}

func (s *SignatureController) WaitFor() {
	s.party.WaitFor()
}

func (s *SignatureController) Next() IController {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data.SessionType == types.SessionType_ReshareSession && s.data.Processing && s.isKeySigner {
		op := &rarimo.ChangeParties{
			Parties:   s.data.New.Parties,
			Signature: s.data.KeySignature,
		}
		content, _ := pkg.GetChangePartiesContent(op)
		s.data.Root = hexutil.Encode(content.CalculateHash())
		s.data.Indexes = []string{s.data.Root}
		return s.factory.GetSignController(s.data.Root)
	}
	return s.factory.GetFinishController()
}

func (s *SignatureController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_SIGN
}
