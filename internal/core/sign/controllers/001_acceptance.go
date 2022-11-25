package controllers

import (
	"context"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type AcceptanceController struct {
	*defaultController
	mu *sync.Mutex
	wg *sync.WaitGroup

	bounds *core.Bounds
	data   LocalProposalData
	result LocalAcceptanceData

	storage secret.Storage
	factory *ControllerFactory
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

	if acceptance.DataToSign == a.data.Root {
		a.mu.Lock()
		defer a.mu.Unlock()

		a.log.Infof("Received acceptance from %s for root %s", sender.Account, acceptance.DataToSign)
		a.result.Accepted[sender.Account] = struct{}{}
	}

	return nil
}

func (a *AcceptanceController) WaitFor() {
	a.wg.Wait()
}

func (a *AcceptanceController) Next() IController {
	if len(a.result.Accepted) <= a.params.T() {
		bounds := core.NewBounds(a.bounds.Finish+1,
			a.params.Step(SigningIndex).Duration+
				1+a.params.Step(FinishingIndex).Duration,
		)

		data := LocalSignatureData{}
		data.LocalProposalData = a.data

		return a.factory.GetFinishController(bounds, data)
	}

	return a.factory.GetSignController(core.NewBounds(a.bounds.Finish+1, a.params.Step(SigningIndex).Duration), a.result)
}

func (a *AcceptanceController) Bounds() *core.Bounds {
	return a.bounds
}

func (a *AcceptanceController) run(ctx context.Context) {
	defer func() {
		a.log.Info("Acceptance controller finished")
		a.wg.Done()
	}()

	details, err := cosmostypes.NewAnyWithValue(&types.AcceptanceRequest{DataToSign: a.data.Root})
	if err != nil {
		a.log.WithError(err).Error("error parsing details")
		return
	}

	a.broadcast.SubmitAll(ctx, &types.MsgSubmitRequest{
		Id:          a.data.SessionId,
		Type:        types.RequestType_Acceptance,
		IsBroadcast: true,
		Details:     details,
	})

	<-ctx.Done()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.result.Accepted[a.storage.AccountAddressStr()] = struct{}{}
	a.log.Infof("Acceptances: %v", a.result.Accepted)
}
