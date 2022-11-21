package core

import (
	"context"
	"fmt"
	"sync"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type FinishController struct {
	*bounds
	*defaultController

	wg *sync.WaitGroup

	data     SignatureData
	proposer *ProposerProvider
	factory  *ControllerFactory
}

func NewFinishController(
	data SignatureData,
	proposer *ProposerProvider,
	defaultController *defaultController,
	bounds *bounds,
	factory *ControllerFactory,
) IController {
	return &FinishController{
		bounds:            bounds,
		defaultController: defaultController,
		wg:                &sync.WaitGroup{},
		data:              data,
		proposer:          proposer,
		factory:           factory,
	}
}

var _ IController = &FinishController{}

func (f *FinishController) StepType() types.StepType {
	return types.StepType_Finishing
}

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
		var meta *rarimo.ConfirmationMeta

		if f.data.Reshare {
			meta = &rarimo.ConfirmationMeta{
				NewKeyECDSA: f.reshare.Key,
				PartyKey:    f.reshare.NewKeys,
			}
		}

		if err := f.SubmitConfirmation(f.data.Indexes, f.data.Root, f.data.Signature, meta); err != nil {
			f.errorf(err, "Failed to submit confirmation. Maybe already submitted.")
		}

		f.proposer.Update(f.data.Signature)
		f.Success()
	} else {
		f.Failed()
	}

	f.secret.UpdateSecret()
	f.params.UpdateParams()

}

func (f *FinishController) WaitFor() {
	f.wg.Wait()
}

func (f *FinishController) Next() IController {
	id := f.SessionID() + 1
	proposer := f.proposer.GetProposer(id)
	sessionBounds := NewBounds(
		f.End()+1,
		f.params.Step(ProposingIndex).Duration+
			1+f.params.Step(AcceptingIndex).Duration+
			1+f.params.Step(ReshareIndex).Duration+
			1+f.params.Step(SigningIndex).Duration+
			1+f.params.Step(FinishingIndex).Duration,
	)

	f.NextSession(id, proposer.Account, sessionBounds)
	return f.factory.GetProposalController(proposer, NewBounds(f.End()+1, f.params.Step(ProposingIndex).Duration))
}

func (f *FinishController) infof(msg string, args ...interface{}) {
	f.Infof("[Acceptance %d] - %s", f.SessionID(), fmt.Sprintf(msg, args))
}

func (f *FinishController) errorf(err error, msg string, args ...interface{}) {
	f.WithError(err).Errorf("[Acceptance %d] - %s", f.SessionID(), fmt.Sprintf(msg, args))
}
