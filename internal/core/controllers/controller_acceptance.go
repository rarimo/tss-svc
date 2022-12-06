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
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type AcceptanceController struct {
	IAcceptanceController
	mu sync.Mutex
	wg *sync.WaitGroup

	data *LocalSessionData

	auth *core.RequestAuthorizer
	log  *logan.Entry
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

	if a.validate(acceptance.Details, acceptance.Type) {
		a.mu.Lock()
		defer a.mu.Unlock()
		a.log.Infof("Received acceptance request from %s for session type=%s", sender.Account, acceptance.Type.String())
		a.data.Acceptances[sender.Account] = struct{}{}
	}

	return nil
}

func (a *AcceptanceController) WaitFor() {
	a.wg.Wait()
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

	a.shareAcceptance(ctx)
	<-ctx.Done()

	a.mu.Lock()
	defer a.mu.Unlock()

	// adding self
	a.data.Acceptances[a.data.New.LocalAccountAddress] = struct{}{}
	a.data.AcceptedSigningPartyIds = getPartyIDsFromAcceptances(a.data.Acceptances, a.data.Old)

	a.log.Infof("Acceptances: %v", a.data.Acceptances)
	a.finish()
}

type IAcceptanceController interface {
	Next() IController
	validate(details *cosmostypes.Any, st types.SessionType) bool
	shareAcceptance(ctx context.Context)
	updateSessionData()
	finish()
}

type DefaultAcceptanceController struct {
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
	log       *logan.Entry
	pg        *pg.Storage
	factory   *ControllerFactory
}

var _ IAcceptanceController = &DefaultAcceptanceController{}

func (a *DefaultAcceptanceController) Next() IController {
	if a.data.Processing {
		return a.factory.GetRootSignController(a.data.Root)
	}
	return a.factory.GetFinishController()
}

func (a *DefaultAcceptanceController) validate(any *cosmostypes.Any, st types.SessionType) bool {
	if st != types.SessionType_DefaultSession {
		return false
	}

	details := new(types.DefaultSessionAcceptanceData)
	if err := proto.Unmarshal(any.Value, details); err != nil {
		a.log.WithError(err).Error("error unmarshalling request")
	}

	return details.Root == a.data.Root
}

func (a *DefaultAcceptanceController) shareAcceptance(ctx context.Context) {
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

func (a *DefaultAcceptanceController) updateSessionData() {
	session, err := a.pg.SessionQ().SessionByID(int64(a.data.SessionId), false)
	if err != nil {
		a.log.WithError(err).Error("error selecting session")
		return
	}

	if session == nil {
		a.log.Error("session entry is not initialized")
		return
	}

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

func (a *DefaultAcceptanceController) finish() {
	if len(a.data.Acceptances) <= a.data.New.T {
		a.data.Processing = false
	}
}

type ReshareAcceptanceController struct {
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
	log       *logan.Entry
	pg        *pg.Storage
	factory   *ControllerFactory
}

var _ IAcceptanceController = &ReshareAcceptanceController{}

func (a *ReshareAcceptanceController) Next() IController {
	if a.data.Processing {
		return a.factory.GetReshareController()
	}

	return a.factory.GetFinishController()
}

func (a *ReshareAcceptanceController) validate(any *cosmostypes.Any, st types.SessionType) bool {
	if st != types.SessionType_ReshareSession {
		return false
	}

	details := new(types.ReshareSessionAcceptanceData)
	if err := proto.Unmarshal(any.Value, details); err != nil {
		a.log.WithError(err).Error("error unmarshalling request")
	}

	return checkSet(details.New, a.data.Old)
}

func (a *ReshareAcceptanceController) shareAcceptance(ctx context.Context) {
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

func (a *ReshareAcceptanceController) updateSessionData() {
	// Nothing to do for reshare session
}

func (a *ReshareAcceptanceController) finish() {
	if len(a.data.Acceptances) < a.data.New.N {
		a.data.Processing = false
	}
}
