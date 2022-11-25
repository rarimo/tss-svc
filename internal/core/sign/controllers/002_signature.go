package controllers

import (
	"context"

	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureController struct {
	*defaultController

	bounds *core.Bounds
	data   LocalAcceptanceData
	result LocalSignatureData

	party   *tss.SignParty
	factory *ControllerFactory
}

var _ IController = &SignatureController{}

func (s *SignatureController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := s.auth.Auth(request)
	if err != nil {
		return err
	}

	if _, ok := s.data.Accepted[sender.Address]; !ok {
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

	<-ctx.Done()
	s.result.Signature = s.party.Result().Signature
}

func (s *SignatureController) WaitFor() {
	s.party.WaitFor()
}

func (s *SignatureController) Next() IController {
	return s.factory.GetFinishController(core.NewBounds(s.bounds.Finish+1, s.params.Step(FinishingIndex).Duration), s.result)
}

func (s *SignatureController) Bounds() *core.Bounds {
	return s.bounds
}
