package controllers

import (
	"context"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureController struct {
	data *LocalSessionData

	broadcast *connectors.BroadcastConnector
	auth      *core.RequestAuthorizer
	log       *logan.Entry

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

	if result := s.party.Result(); result != nil {
		s.data.OperationSignature = hexutil.Encode(append(result.Signature, result.SignatureRecovery...))
	}
}

func (s *SignatureController) WaitFor() {
	s.party.WaitFor()
}

func (s *SignatureController) Next() IController {
	return nil
}

func (s *SignatureController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_SIGN
}
