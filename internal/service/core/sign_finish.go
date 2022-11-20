package core

import (
	"context"
	"fmt"
	"sync"

	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type FinishController struct {
	*bounds
	*defaultController

	wg *sync.WaitGroup

	sessionId uint64
	data      types.SignatureData
	proposer  *ProposerProvider
	factory   *ControllerFactory
}

func NewFinishController(
	sessionId uint64,
	data types.SignatureData,
	proposer *ProposerProvider,
	defaultController *defaultController,
	bounds *bounds,
	factory *ControllerFactory,
) IController {
	return &FinishController{
		bounds:            bounds,
		defaultController: defaultController,
		wg:                &sync.WaitGroup{},
		sessionId:         sessionId,
		data:              data,
		proposer:          proposer,
		factory:           factory,
	}
}

var _ IController = &FinishController{}

func (f *FinishController) Receive(request *types.MsgSubmitRequest) error {
	return nil
}

func (f *FinishController) Run(ctx context.Context) {
	f.wg.Add(1)
	func() {
		f.infof("Controller finished")
		f.wg.Done()
	}()

	if f.data.Signature != "" {
		if err := f.SubmitConfirmation(f.data.Indexes, f.data.Root, f.data.Signature); err != nil {
			f.errorf(err, "Failed to submit confirmation. Maybe already submitted.")
			return
		}

		f.proposer.Update(f.data.Signature)
	}
}

func (f *FinishController) WaitFor() {
	f.wg.Wait()
}

func (f *FinishController) Next() IController {
	f.params.UpdateParams()
	pBounds := NewBounds(f.End()+1, f.params.Step(ProposingIndex).Duration)
	return f.factory.GetProposalController(f.sessionId+1, f.proposer.GetProposer(f.sessionId+1), pBounds)
}

func (f *FinishController) infof(msg string, args ...interface{}) {
	f.Infof("[Acceptance %d] - %s", f.sessionId, fmt.Sprintf(msg, args))
}

func (f *FinishController) errorf(err error, msg string, args ...interface{}) {
	f.WithError(err).Errorf("[Acceptance %d] - %s", f.sessionId, fmt.Sprintf(msg, args))
}
