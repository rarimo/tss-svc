package core

import (
	"context"
	"fmt"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type AcceptanceController struct {
	*defaultController
	*bounds

	mu *sync.Mutex
	wg *sync.WaitGroup

	sessionId uint64
	data      types.ProposalData
	result    types.AcceptanceData

	factory *ControllerFactory
}

func NewAcceptanceController(
	defaultController *defaultController,
	sessionId uint64,
	data types.ProposalData,
	bounds *bounds,
	factory *ControllerFactory,
) *AcceptanceController {
	return &AcceptanceController{
		bounds:            bounds,
		defaultController: defaultController,
		mu:                &sync.Mutex{},
		wg:                &sync.WaitGroup{},
		sessionId:         sessionId,
		data:              data,
		result: types.AcceptanceData{
			Indexes:     data.Indexes,
			Root:        data.Root,
			Acceptances: make(map[string]bool),
			Reshare:     data.Reshare,
		},
		factory: factory,
	}
}

var _ IController = &AcceptanceController{}

func (a *AcceptanceController) Run(ctx context.Context) {
	a.wg.Add(1)
	go a.run(ctx)
}

func (a *AcceptanceController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := a.auth.Auth(request)
	if err != nil {
		return err
	}

	if request.Type != types.RequestType_Acceptance {
		return ErrInvalidRequestType
	}

	acceptance := new(types.AcceptanceRequest)
	if err := proto.Unmarshal(request.Details.Value, acceptance); err != nil {
		return errors.Wrap(err, "error unmarshalling request")
	}

	if acceptance.Root == a.data.Root {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.infof("Received acceptance from %s for root %s", sender.Account, acceptance.Root)
		a.result.Acceptances[sender.Account] = true
	}

	return nil
}

func (a *AcceptanceController) WaitFor() {
	a.wg.Wait()
}

func (a *AcceptanceController) Next() IController {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.result.Acceptances) > a.params.T() {
		a.infof("Reached required amount of acceptances")
		if a.data.Reshare {
			return a.factory.GetReshareController(a.sessionId, a.result, NewBounds(a.End()+1, a.params.Step(ReshareIndex).Duration))
		}
		sBounds := NewBounds(a.End()+1, a.params.Step(ReshareIndex).Duration+1+a.params.Step(SigningIndex).Duration)
		return a.factory.GetSignatureController(a.sessionId, a.result, sBounds)
	}

	bounds := NewBounds(
		a.End()+1,
		a.params.Step(ReshareIndex).Duration+
			1+a.params.Step(SigningIndex).Duration+
			1+a.params.Step(FinishingIndex).Duration,
	)

	return a.factory.GetFinishController(a.sessionId, types.SignatureData{}, bounds)
}

func (a *AcceptanceController) run(ctx context.Context) {
	defer func() {
		a.infof("Controller finished")
		a.wg.Done()
	}()

	details, err := cosmostypes.NewAnyWithValue(&types.AcceptanceRequest{Root: a.data.Root})
	if err != nil {
		a.errorf(err, "error parsing details")
		return
	}

	a.SubmitAll(ctx, &types.MsgSubmitRequest{
		Type:        types.RequestType_Acceptance,
		IsBroadcast: true,
		Details:     details,
	})

	<-ctx.Done()

	a.mu.Lock()
	defer a.mu.Unlock()
	a.result.Acceptances[a.secret.AccountAddressStr()] = true
	a.infof("Acceptances: %v", a.result.Acceptances)
}

func (a *AcceptanceController) infof(msg string, args ...interface{}) {
	a.Infof("[Acceptance %d] - %s", a.sessionId, fmt.Sprintf(msg, args))
}

func (a *AcceptanceController) errorf(err error, msg string, args ...interface{}) {
	a.WithError(err).Errorf("[Acceptance %d] - %s", a.sessionId, fmt.Sprintf(msg, args))
}
