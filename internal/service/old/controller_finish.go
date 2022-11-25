package old

import (
	"context"
	"fmt"
	"sync"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/sign"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/sign/controllers"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type FinishController struct {
	*bounds
	*sign.defaultController

	wg *sync.WaitGroup

	data     SignatureData
	proposer *ProposerProvider
	factory  *controllers.ControllerFactory
}

func NewFinishController(
	data SignatureData,
	proposer *ProposerProvider,
	defaultController *sign.defaultController,
	bounds *bounds,
	factory *controllers.ControllerFactory,
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
	f.checkRats()
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

func (f *FinishController) checkRats() {
	rats := f.rats.GetRats()
	parties := f.params.Parties()
	newSet := make([]*rarimo.Party, 0, len(parties))

	if len(rats) == 0 {
		return
	}

	for _, p := range parties {
		if _, ok := rats[p.Account]; !ok {
			newSet = append(newSet, p)
		}
	}

	if len(newSet) < len(parties) {
		if err := f.SubmitChangeSet(newSet); err != nil {
			f.errorf(err, "error submitting change parties op")
		}
	}
}

func (f *FinishController) infof(msg string, args ...interface{}) {
	f.Infof("[Acceptance %d] - %s", f.SessionID(), fmt.Sprintf(msg, args))
}

func (f *FinishController) errorf(err error, msg string, args ...interface{}) {
	f.WithError(err).Errorf("[Acceptance %d] - %s", f.SessionID(), fmt.Sprintf(msg, args))
}
