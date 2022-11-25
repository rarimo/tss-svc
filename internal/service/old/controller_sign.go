package old

import (
	"context"
	goerr "errors"
	"fmt"

	"gitlab.com/rarify-protocol/tss-svc/internal/core/sign"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/sign/controllers"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

var (
	ErrSenderHasNotAccepted = goerr.New("sender has not accepted proposal")
)

type SignatureController struct {
	*sign.defaultController
	*bounds

	party   *tss.SignParty
	data    AcceptanceData
	result  SignatureData
	factory *controllers.ControllerFactory
}

func NewSignatureController(
	data AcceptanceData,
	bounds *bounds,
	defaultController *sign.defaultController,
	factory *controllers.ControllerFactory,
) *SignatureController {
	return &SignatureController{
		defaultController: defaultController,
		bounds:            bounds,
		data:              data,
		party:             tss.NewSignParty(data.Root, data.Root, defaultController.secret, defaultController.params, defaultController.BroadcastConnector, defaultController.Entry),
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

	if _, ok := s.data.Acceptances[sender.Address]; !ok {
		return ErrSenderHasNotAccepted
	}

	if request.Type != types.RequestType_Sign {
		return ErrInvalidRequestType
	}

	s.party.Receive(sender, request.IsBroadcast, request.Details.Value)
	return nil
}

func (s *SignatureController) Run(ctx context.Context) {
	s.party.Run(ctx)
}

func (s *SignatureController) WaitFor() {
	s.party.WaitFor()
}

func (s *SignatureController) Next() IController {
	fBounds := NewBounds(s.End()+1, s.params.Step(FinishingIndex).Duration)
	return s.factory.GetFinishController(s.result, fBounds)
}

func (s *SignatureController) infof(msg string, args ...interface{}) {
	s.Infof("[Proposal %d] - %s", s.SessionID(), fmt.Sprintf(msg, args))
}

func (s *SignatureController) errorf(err error, msg string, args ...interface{}) {
	s.WithError(err).Errorf("[Proposal %d] - %s", s.SessionID(), fmt.Sprintf(msg, args))
}
