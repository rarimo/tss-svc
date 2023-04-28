package controllers

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/internal/connectors"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
	"google.golang.org/protobuf/types/known/anypb"
)

// iAcceptanceController defines custom logic for every acceptance controller.
type iAcceptanceController interface {
	Next() IController
	validate(ctx core.Context, details *anypb.Any, st types.SessionType) bool
	shareAcceptance(ctx core.Context)
	updateSessionData(ctx core.Context)
	finish(ctx core.Context)
}

// AcceptanceController is responsible for sharing and collecting acceptances for different types of session.
type AcceptanceController struct {
	iAcceptanceController
	mu sync.Mutex
	wg *sync.WaitGroup

	data *LocalSessionData

	auth *core.RequestAuthorizer
	log  *logan.Entry
}

// Implements IController interface
var _ IController = &AcceptanceController{}

// Run initiates sharing of the acceptances with other parties.
// Should be launched only in case of valid proposal (session processing should be `true`)
func (a *AcceptanceController) Run(c context.Context) {
	ctx := core.WrapCtx(c)
	a.log.Infof("Starting: %s", a.Type().String())
	a.wg.Add(1)
	go a.run(ctx)
}

// Receive accepts acceptances from other parties and executes `iAcceptanceController.validate` function.
// If function returns positive result sender address will be added to the accepted list.
// Self acceptance will be added to the list after controller base logic execution.
// After context finishing it calls the `iAcceptanceController.finish` method to calculate the acceptance results.
func (a *AcceptanceController) Receive(c context.Context, request *types.MsgSubmitRequest) error {
	ctx := core.WrapCtx(c)
	sender, err := a.auth.Auth(request)
	if err != nil {
		return err
	}

	if request.Type != types.RequestType_Acceptance {
		return ErrInvalidRequestType
	}

	if !a.validate(ctx, request.Details, request.SessionType) {
		a.data.Offenders[sender.Account] = struct{}{}
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.log.Infof("Received acceptance request from %s for session type=%s", sender.Account, request.SessionType.String())
	a.data.Acceptances[sender.Account] = struct{}{}
	return nil
}

// WaitFor waits until controller finishes its logic. Context cancel should be called before.
// WaitFor should be called before.
func (a *AcceptanceController) WaitFor() {
	a.wg.Wait()
}

func (a *AcceptanceController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_ACCEPTANCE
}

func (a *AcceptanceController) run(ctx core.Context) {
	defer func() {
		a.log.Infof("Finishing: %s", a.Type().String())
		a.updateSessionData(ctx)
		a.wg.Done()
	}()

	a.shareAcceptance(ctx)
	<-ctx.Context().Done()

	a.mu.Lock()
	defer a.mu.Unlock()

	// adding self
	a.data.Acceptances[ctx.SecretStorage().GetTssSecret().AccountAddress()] = struct{}{}

	a.log.Infof("Received acceptances list: %v", a.data.Acceptances)

	// report for parties that has not voted for accepted proposal
	for _, party := range a.data.Set.Parties {
		if _, ok := a.data.Acceptances[party.Account]; !ok {
			a.data.Offenders[party.Account] = struct{}{}
		}
	}

	a.finish(ctx)
}

// defaultAcceptanceController represents custom logic for types.SessionType_DefaultSession
type defaultAcceptanceController struct {
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
	factory   *ControllerFactory
}

// Implements iAcceptanceController interface
var _ iAcceptanceController = &defaultAcceptanceController{}

// Next method returns the next controller instance to be launched.
// If self party is the session signer the next controller should be a root signature controller.
// Otherwise, it will be a finish controller.
func (a *defaultAcceptanceController) Next() IController {
	if a.data.Processing && a.data.IsSigner {
		return a.factory.GetRootSignController(a.data.Root)
	}
	return a.factory.GetFinishController()
}

func (a *defaultAcceptanceController) validate(ctx core.Context, any *anypb.Any, st types.SessionType) bool {
	if st != types.SessionType_DefaultSession {
		return false
	}

	details := new(types.DefaultSessionAcceptanceData)
	if err := any.UnmarshalTo(details); err != nil {
		ctx.Log().WithError(err).Error("Error unmarshalling request")
	}

	return details.Root == a.data.Root
}

func (a *defaultAcceptanceController) shareAcceptance(ctx core.Context) {
	details, err := anypb.New(&types.DefaultSessionAcceptanceData{Root: a.data.Root})
	if err != nil {
		ctx.Log().WithError(err).Error("Error parsing details")
		return
	}

	go a.broadcast.SubmitAllWithReport(ctx.Context(), ctx.Core(), &types.MsgSubmitRequest{
		Id:          a.data.SessionId,
		Type:        types.RequestType_Acceptance,
		IsBroadcast: true,
		Details:     details,
	})
}

// updateSessionData updates the database entry according to the controller result.
func (a *defaultAcceptanceController) updateSessionData(ctx core.Context) {
	session, err := ctx.PG().DefaultSessionDatumQ().DefaultSessionDatumByID(int64(a.data.SessionId), false)
	if err != nil {
		ctx.Log().WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		ctx.Log().Error("Session entry is not initialized")
		return
	}

	session.Accepted = acceptancesToArr(a.data.Acceptances)
	if err = ctx.PG().DefaultSessionDatumQ().Update(session); err != nil {
		ctx.Log().WithError(err).Error("Error updating session entry")
	}
}

// finish verifies that results satisfies the requirements (t + 1 acceptances) and calculates the signature producers set.
func (a *defaultAcceptanceController) finish(ctx core.Context) {
	// T+1 required for signing
	if len(a.data.Acceptances) <= a.data.Set.T {
		a.data.Processing = false
		return
	}

	defer func() {
		ctx.Log().Infof("Session signers: %v", acceptancesToArr(a.data.Signers))
		_, a.data.IsSigner = a.data.Signers[ctx.SecretStorage().GetTssSecret().AccountAddress()]
	}()

	if len(a.data.Acceptances) == a.data.Set.T+1 {
		a.data.Signers = a.data.Acceptances
		return
	}

	a.data.Signers = GetSignersSet(a.data.Acceptances, a.data.Set.T, a.data.Set.LastSignature, a.data.SessionId)
}

// reshareAcceptanceController represents custom logic for types.SessionType_ReshareSession
type reshareAcceptanceController struct {
	data      *LocalSessionData
	broadcast *connectors.BroadcastConnector
	factory   *ControllerFactory
}

// Implements iAcceptanceController interface
var _ iAcceptanceController = &reshareAcceptanceController{}

// Next method returns the next controller instance to be launched. If controller finished successfully
// the next controller will be a keygen controller. Otherwise, it will be a finish controller.
func (a *reshareAcceptanceController) Next() IController {
	if a.data.Processing {
		return a.factory.GetKeygenController()
	}

	return a.factory.GetFinishController()
}

func (a *reshareAcceptanceController) validate(ctx core.Context, any *anypb.Any, st types.SessionType) bool {
	if st != types.SessionType_ReshareSession {
		return false
	}

	details := new(types.ReshareSessionAcceptanceData)
	if err := any.UnmarshalTo(details); err != nil {
		ctx.Log().WithError(err).Error("Error unmarshalling request")
	}

	return checkSet(details.New, a.data.Set)
}

func (a *reshareAcceptanceController) shareAcceptance(ctx core.Context) {
	details, err := anypb.New(&types.ReshareSessionAcceptanceData{New: getSet(a.data.Set)})
	if err != nil {
		ctx.Log().WithError(err).Error("Error parsing details")
		return
	}

	go a.broadcast.SubmitAllWithReport(ctx.Context(), ctx.Core(), &types.MsgSubmitRequest{
		Id:          a.data.SessionId,
		Type:        types.RequestType_Acceptance,
		IsBroadcast: true,
		Details:     details,
	})
}

func (a *reshareAcceptanceController) updateSessionData(ctx core.Context) {
	// Nothing to do for reshare session
}

// finish verifies that results satisfies the requirements (all accepted) and calculates the
// signature producers set (based on old parties).
func (a *reshareAcceptanceController) finish(ctx core.Context) {
	if len(a.data.Acceptances) < a.data.Set.N {
		a.data.Processing = false
		return
	}

	defer func() {
		ctx.Log().Infof("Session signers: %v", acceptancesToArr(a.data.Signers))
	}()

	signAcceptances := filterAcceptances(a.data.Acceptances, a.data.Set.VerifiedParties)

	if len(signAcceptances) == a.data.Set.T+1 {
		a.data.Signers = signAcceptances
		_, a.data.IsSigner = a.data.Signers[ctx.SecretStorage().GetTssSecret().AccountAddress()]
		return
	}

	a.data.Signers = GetSignersSet(signAcceptances, a.data.Set.T, a.data.Set.LastSignature, a.data.SessionId)
}
