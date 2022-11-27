package controllers

import (
	"context"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type AcceptanceController struct {
	mu *sync.Mutex
	wg *sync.WaitGroup

	data *LocalSessionData

	broadcast *connectors.BroadcastConnector
	auth      *core.RequestAuthorizer
	log       *logan.Entry

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

	if a.data.SessionType != acceptance.Type {
		return ErrInvalidRequestType
	}

	switch acceptance.Type {
	case types.SessionType_DefaultSession:
		details := new(types.DefaultSessionAcceptanceData)
		if err := proto.Unmarshal(request.Details.Value, acceptance); err != nil {
			return errors.Wrap(err, "error unmarshalling request")
		}

		if details.Root == a.data.Root {
			a.mu.Lock()
			defer a.mu.Unlock()

			a.log.Infof("Received acceptance from %s for root %s", sender.Account, details.Root)
			a.data.Acceptances[sender.Account] = struct{}{}
		}

	case types.SessionType_ReshareSession:
		details := new(types.ReshareSessionAcceptanceData)
		if err := proto.Unmarshal(request.Details.Value, acceptance); err != nil {
			return errors.Wrap(err, "error unmarshalling request")
		}

		if checkSet(details.New, a.data.New) {
			a.mu.Lock()
			defer a.mu.Unlock()

			a.log.Infof("Received acceptance from %s for reshare", sender.Account)
			a.data.Acceptances[sender.Account] = struct{}{}
		}
	}

	return nil
}

func (a *AcceptanceController) WaitFor() {
	a.wg.Wait()
}

func (a *AcceptanceController) Next() IController {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.data.Processing {
		switch a.data.SessionType {
		case types.SessionType_DefaultSession:
			return a.factory.GetSignRootController()
		case types.SessionType_ReshareSession:
			return a.factory.GetReshareController()
		}
	}

	return a.factory.GetFinishController()
}

func (a *AcceptanceController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_ACCEPTANCE
}

func (a *AcceptanceController) run(ctx context.Context) {
	defer func() {
		a.log.Info("Acceptance controller finished")
		a.wg.Done()
	}()

	// TODO
	details, err := cosmostypes.NewAnyWithValue(&types.DefaultSessionAcceptanceData{Root: ""})
	if err != nil {
		a.log.WithError(err).Error("error parsing details")
		return
	}

	details, err = cosmostypes.NewAnyWithValue(&types.AcceptanceRequest{Type: types.SessionType_DefaultSession, Details: details})
	if err != nil {
		a.log.WithError(err).Error("error parsing details")
		return
	}

	a.broadcast.SubmitAll(ctx, &types.MsgSubmitRequest{
		Id:          0,
		Type:        types.RequestType_Acceptance,
		IsBroadcast: true,
		Details:     details,
	})

	<-ctx.Done()

	a.mu.Lock()
	defer a.mu.Unlock()

	a.data.Acceptances[a.data.New.LocalAccountAddress] = struct{}{}
	a.log.Infof("Acceptances: %v", a.data.Acceptances)

	switch a.data.SessionType {
	case types.SessionType_DefaultSession:
		if len(a.data.Acceptances) <= a.data.New.T {
			a.data.Processing = false
		}
	case types.SessionType_ReshareSession:
		if len(a.data.Acceptances) < a.data.New.N {
			a.data.Processing = false
		}
	}
}
