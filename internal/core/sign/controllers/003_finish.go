package controllers

import (
	"context"
	"sync"

	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/connectors"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type FinishController struct {
	*defaultController
	wg *sync.WaitGroup

	bounds *core.Bounds
	data   LocalSignatureData

	core     *connectors.CoreConnector
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

	if f.data.Signature != "" {
		if err := f.core.SubmitConfirmation(f.data.Indexes, f.data.Root, f.data.Signature); err != nil {
			f.log.WithError(err).Error("Failed to submit confirmation. Maybe already submitted.")
		}
		f.proposer.WithSignature(f.data.Signature)
	}
}

func (f *FinishController) WaitFor() {
	f.wg.Wait()
}

func (f *FinishController) Next() IController {
	return nil
}

func (f *FinishController) Bounds() *core.Bounds {
	return f.bounds
}
