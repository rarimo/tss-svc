package controllers

import (
	"context"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type AcceptanceController struct {
	mu sync.Mutex
	wg *sync.WaitGroup

	data *LocalSessionData

	broadcast *connectors.BroadcastConnector
	auth      *core.RequestAuthorizer
	log       *logan.Entry
	pg        *pg.Storage
	factory   *ControllerFactory
}

var _ IController = &AcceptanceController{}

func (a *AcceptanceController) Run(ctx context.Context) {
	a.log.Infof("Starting %s", a.Type().String())
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
			return a.factory.GetSignController(a.data.Root, false)
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
		a.log.Infof("%s finished", a.Type().String())
		a.updateSessionData()
		a.wg.Done()
	}()

	switch a.data.SessionType {
	case types.SessionType_DefaultSession:
		a.shareDefaultAcceptance(ctx)
	case types.SessionType_ReshareSession:
		a.shareReshareAcceptance(ctx)
	}

	<-ctx.Done()

	a.mu.Lock()
	defer a.mu.Unlock()

	// adding self
	a.data.Acceptances[a.data.New.LocalAccountAddress] = struct{}{}

	accepted := make([]*rarimo.Party, 0, a.data.Old.N)
	for _, p := range a.data.Old.Parties {
		if _, ok := a.data.Acceptances[p.Account]; ok {
			accepted = append(accepted, p)
		}
	}
	a.data.AcceptedSigningPartyIds = core.PartyIds(accepted)

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

func (a *AcceptanceController) shareDefaultAcceptance(ctx context.Context) {
	details, err := cosmostypes.NewAnyWithValue(&types.DefaultSessionAcceptanceData{Root: a.data.Root})
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
		Id:          a.data.SessionId,
		Type:        types.RequestType_Acceptance,
		IsBroadcast: true,
		Details:     details,
	})
}

func (a *AcceptanceController) shareReshareAcceptance(ctx context.Context) {
	details, err := cosmostypes.NewAnyWithValue(&types.ReshareSessionAcceptanceData{New: getSet(a.data.New)})
	if err != nil {
		a.log.WithError(err).Error("error parsing details")
		return
	}

	details, err = cosmostypes.NewAnyWithValue(&types.AcceptanceRequest{Type: types.SessionType_ReshareSession, Details: details})
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
}

func (a *AcceptanceController) updateSessionData() {
	session, err := a.pg.SessionQ().SessionByID(int64(a.data.SessionId), false)
	if err != nil {
		a.log.WithError(err).Error("error selecting session")
		return
	}

	if session == nil {
		a.log.Error("session entry is not initialized")
		return
	}

	if a.data.SessionType == types.SessionType_DefaultSession {
		data, err := a.pg.DefaultSessionDatumQ().DefaultSessionDatumByID(session.DataID.Int64, false)
		if err != nil {
			a.log.WithError(err).Error("error selecting session data")
			return
		}

		if data == nil {
			a.log.Error("session data is not initialized")
			return
		}

		data.Accepted = acceptancesToArr(a.data.Acceptances)
		if err = a.pg.DefaultSessionDatumQ().Update(data); err != nil {
			a.log.WithError(err).Error("error updating session data entry")
		}
	}
}
