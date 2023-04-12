package controllers

import (
	"context"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/internal/connectors"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// AcceptanceController is responsible for sharing and collecting acceptances for different types of session.
type AcceptanceController struct {
	IAcceptanceController
	mu sync.Mutex
	wg *sync.WaitGroup

	data *LocalSessionData

	auth *core.RequestAuthorizer
	log  *logan.Entry
}

// Implements IController interface
var _ IController = &AcceptanceController{}

func (a *AcceptanceController) Run(ctx context.Context) {
	a.log.Infof("Starting: %s", a.Type().String())
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

	if !a.validate(request.Details, request.SessionType) {
		a.data.Offenders[sender.Account] = struct{}{}
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.log.Infof("Received acceptance request from %s for session type=%s", sender.Account, request.SessionType.String())
	a.data.Acceptances[sender.Account] = struct{}{}
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
		a.log.Infof("Finishing: %s", a.Type().String())
		a.updateSessionData()
		a.wg.Done()
	}()

	a.shareAcceptance(ctx)
	<-ctx.Done()

	a.mu.Lock()
	defer a.mu.Unlock()

	// adding self
	a.data.Acceptances[a.data.Secret.AccountAddress()] = struct{}{}

	a.log.Infof("Received acceptances list: %v", a.data.Acceptances)

	if proposalAccepted := a.finish(); proposalAccepted {
		// report for parties that has not voted for accepted proposal
		for _, party := range a.data.Set.Parties {
			if _, ok := a.data.Acceptances[party.Account]; !ok {
				a.data.Offenders[party.Account] = struct{}{}
			}
		}
	}
}

// IAcceptanceController defines custom logic for every acceptance controller.
type IAcceptanceController interface {
	Next() IController
	validate(details *cosmostypes.Any, st types.SessionType) bool
	shareAcceptance(ctx context.Context)
	updateSessionData()
	finish() bool
}

// DefaultAcceptanceController represents custom logic for types.SessionType_DefaultSession
type DefaultAcceptanceController struct {
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
	core      *connectors.CoreConnector
	log       *logan.Entry
	pg        *pg.Storage
	factory   *ControllerFactory
}

// Implements IAcceptanceController interface
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
		a.log.WithError(err).Error("Error unmarshalling request")
	}

	return details.Root == a.data.Root
}

func (a *DefaultAcceptanceController) shareAcceptance(ctx context.Context) {
	details, err := cosmostypes.NewAnyWithValue(&types.DefaultSessionAcceptanceData{Root: a.data.Root})
	if err != nil {
		a.log.WithError(err).Error("Error parsing details")
		return
	}

	go a.broadcast.SubmitAllWithReport(ctx, a.core, &types.MsgSubmitRequest{
		Id:          a.data.SessionId,
		Type:        types.RequestType_Acceptance,
		IsBroadcast: true,
		Details:     details,
	})
}

func (a *DefaultAcceptanceController) updateSessionData() {
	session, err := a.pg.DefaultSessionDatumQ().DefaultSessionDatumByID(int64(a.data.SessionId), false)
	if err != nil {
		a.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		a.log.Error("Session entry is not initialized")
		return
	}

	session.Accepted = acceptancesToArr(a.data.Acceptances)
	if err = a.pg.DefaultSessionDatumQ().Update(session); err != nil {
		a.log.WithError(err).Error("Error updating session entry")
	}
}

func (a *DefaultAcceptanceController) finish() bool {
	// T+1 required for signing
	if len(a.data.Acceptances) <= a.data.Set.T {
		a.data.Processing = false
		return false
	}

	defer func() {
		a.log.Infof("Session signers: %v", acceptancesToArr(a.data.Signers))
	}()

	if len(a.data.Acceptances) == a.data.Set.T+1 {
		a.data.Signers = a.data.Acceptances
		return true
	}

	a.data.Signers = GetSignersSet(a.data.Acceptances, a.data.Set.T, a.data.Set.LastSignature, a.data.SessionId)
	return true
}

// ReshareAcceptanceController represents custom logic for types.SessionType_ReshareSession
type ReshareAcceptanceController struct {
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
	core      *connectors.CoreConnector
	log       *logan.Entry
	pg        *pg.Storage
	factory   *ControllerFactory
}

// Implements IAcceptanceController interface
var _ IAcceptanceController = &ReshareAcceptanceController{}

func (a *ReshareAcceptanceController) Next() IController {
	if a.data.Processing {
		return a.factory.GetKeygenController()
	}

	return a.factory.GetFinishController()
}

func (a *ReshareAcceptanceController) validate(any *cosmostypes.Any, st types.SessionType) bool {
	if st != types.SessionType_ReshareSession {
		return false
	}

	details := new(types.ReshareSessionAcceptanceData)
	if err := proto.Unmarshal(any.Value, details); err != nil {
		a.log.WithError(err).Error("Error unmarshalling request")
	}

	return checkSet(details.New, a.data.Set)
}

func (a *ReshareAcceptanceController) shareAcceptance(ctx context.Context) {
	details, err := cosmostypes.NewAnyWithValue(&types.ReshareSessionAcceptanceData{New: getSet(a.data.Set)})
	if err != nil {
		a.log.WithError(err).Error("Error parsing details")
		return
	}

	go a.broadcast.SubmitAllWithReport(ctx, a.core, &types.MsgSubmitRequest{
		Id:          a.data.SessionId,
		Type:        types.RequestType_Acceptance,
		IsBroadcast: true,
		Details:     details,
	})
}

func (a *ReshareAcceptanceController) updateSessionData() {
	// Nothing to do for reshare session
}

func (a *ReshareAcceptanceController) finish() bool {
	if len(a.data.Acceptances) < a.data.Set.N {
		a.data.Processing = false
		return false
	}

	defer func() {
		a.log.Infof("Session signers: %v", acceptancesToArr(a.data.Signers))
	}()

	signAcceptances := filterAcceptances(a.data.Acceptances, a.data.Set.VerifiedParties)

	if len(signAcceptances) == a.data.Set.T+1 {
		a.data.Signers = signAcceptances
		return true
	}

	a.data.Signers = GetSignersSet(signAcceptances, a.data.Set.T, a.data.Set.LastSignature, a.data.SessionId)
	return true
}
