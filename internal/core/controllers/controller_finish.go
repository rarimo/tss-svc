package controllers

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type FinishController struct {
	wg *sync.WaitGroup

	core *connectors.CoreConnector
	log  *logan.Entry

	data *LocalSessionData

	proposer *core.Proposer
	factory  *ControllerFactory
}

var _ IController = &FinishController{}

func (f *FinishController) Receive(request *types.MsgSubmitRequest) error {
	return nil
}

func (f *FinishController) Run(ctx context.Context) {
	f.wg.Add(1)
	func() {
		f.log.Info("Finish controller finished")
		f.wg.Done()
	}()

	if !f.data.Processing {
		return
	}

	switch f.data.SessionType {
	case types.SessionType_DefaultSession:
		f.finishDefaultSession()
	case types.SessionType_ReshareSession:
		f.finishReshareSession()
	}
}

func (f *FinishController) WaitFor() {
	f.wg.Wait()
}

func (f *FinishController) Next() IController {
	return nil
}

func (f *FinishController) Type() types.ControllerType {
	if f.data.SessionType == types.SessionType_DefaultSession {
		return types.ControllerType_CONTROLLER_FINISH_DEFAULT
	}
	return types.ControllerType_CONTROLLER_FINISH_RESHARE
}

func (f *FinishController) finishReshareSession() {
	msg1 := &rarimo.MsgCreateChangePartiesOp{
		Creator:   f.data.New.LocalAccountAddress,
		NewSet:    f.data.New.Parties,
		Signature: f.data.KeySignature,
	}

	msg2 := &rarimo.MsgCreateConfirmation{
		Creator:        f.data.New.LocalAccountAddress,
		Root:           f.data.Root,
		Indexes:        f.data.Indexes,
		SignatureECDSA: f.data.OperationSignature,
	}

	if err := f.core.Submit(msg1, msg2); err != nil {
		f.log.WithError(err).Error("Failed to submit confirmation. Maybe already submitted.")
	}
	f.proposer.WithSignature(f.data.OperationSignature)
}

func (f *FinishController) finishDefaultSession() {
	if err := f.core.SubmitConfirmation(f.data.Indexes, f.data.Root, f.data.OperationSignature); err != nil {
		f.log.WithError(err).Error("Failed to submit confirmation. Maybe already submitted.")
	}
	f.proposer.WithSignature(f.data.OperationSignature)
}
