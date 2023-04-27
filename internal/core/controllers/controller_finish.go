package controllers

import (
	"context"
	"sync"

	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// iFinishController defines custom logic for every finish controller.
type iFinishController interface {
	finish(ctx core.Context)
	updateSessionEntry(ctx core.Context)
}

// FinishController is responsible for finishing sessions. For example: submit transactions, update session entry, etc.
type FinishController struct {
	iFinishController
	wg   *sync.WaitGroup
	data *LocalSessionData
}

// Implements IController interface
var _ IController = &FinishController{}

func (f *FinishController) Receive(context.Context, *types.MsgSubmitRequest) error {
	return nil
}

// Run initiates the report submitting for all parties that was included into Offenders set. After it executes the
// `iFinishController.finish` logic.
func (f *FinishController) Run(c context.Context) {
	ctx := core.WrapCtx(c)
	ctx.Log().Infof("Starting: %s", f.Type().String())
	f.wg.Add(1)
	defer func() {
		ctx.Log().Infof("Finishing: %s", f.Type().String())
		f.updateSessionEntry(ctx)
		f.wg.Done()
	}()

	for offender := range f.data.Offenders {
		if err := ctx.Core().SubmitReport(
			f.data.SessionId,
			rarimo.ViolationType_Spam,
			offender,
			"Party shared invalid data or have not accepted valid proposal",
		); err != nil {
			ctx.Log().WithError(err).Errorf("Error submitting violation report for party: %s", offender)
		}
	}

	f.finish(ctx)
}

// WaitFor waits until controller finishes its logic. Context cancel should be called before.
func (f *FinishController) WaitFor() {
	f.wg.Wait()
}

func (f *FinishController) Next() IController {
	return nil
}

func (f *FinishController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_FINISH
}

// keygenFinishController represents custom logic for types.SessionType_KeygenSession
type keygenFinishController struct {
	data *LocalSessionData
}

var _ iFinishController = &keygenFinishController{}

// finish in case of successful session updates the stores TSS secret and submits the `rarimo.MsgSetupInitial` message.
func (k *keygenFinishController) finish(ctx core.Context) {
	if k.data.Processing {
		ctx.Log().Infof("Session %s #%d finished successfully", k.data.SessionType.String(), k.data.SessionId)
		if err := ctx.SecretStorage().SetTssSecret(k.data.NewSecret); err != nil {
			panic(err)
		}

		ctx.Log().Info("Submitting setup initial message to finish keygen session.")
		msg := &rarimo.MsgSetupInitial{
			Creator:        k.data.NewSecret.AccountAddress(),
			NewPublicKey:   k.data.NewSecret.GlobalPubKey(),
			PartyPublicKey: k.data.NewSecret.TssPubKey(),
		}

		if err := ctx.Core().Submit(msg); err != nil {
			panic(err)
		}
		return
	}

	ctx.Log().Infof("Session %s #%d finished unsuccessfully", k.data.SessionType.String(), k.data.SessionId)
}

// updateSessionData updates the database entry according to the controller result.
func (k *keygenFinishController) updateSessionEntry(ctx core.Context) {
	session, err := ctx.PG().KeygenSessionDatumQ().KeygenSessionDatumByID(int64(k.data.SessionId), false)
	if err != nil {
		ctx.Log().WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		ctx.Log().Error("Session entry is not initialized")
		return
	}

	session.Status = int(types.SessionStatus_SessionSucceeded)
	if !k.data.Processing {
		session.Status = int(types.SessionStatus_SessionFailed)
	}

	if err := ctx.PG().KeygenSessionDatumQ().Update(session); err != nil {
		ctx.Log().Error("Error updating session entry")
	}
}

// keygenFinishController represents custom logic for types.SessionType_DefaultSession
type defaultFinishController struct {
	data *LocalSessionData
}

var _ iFinishController = &defaultFinishController{}

// finish in case of successful session checks that self party was a signer.
// If true, it will share the generated signature via submitting confirmation message to the core.
// In case of unsuccessful session the selected indexes will be returned to the pool.
func (d *defaultFinishController) finish(ctx core.Context) {
	if d.data.Processing {
		ctx.Log().Infof("Session %s #%d finished successfully", d.data.SessionType.String(), d.data.SessionId)
		if !d.data.IsSigner {
			ctx.Log().Info("Self party was not a part of signing round")
			return
		}

		ctx.Log().Info("Submitting confirmation message to finish default session.")
		if err := ctx.Core().SubmitConfirmation(d.data.Indexes, d.data.Root, d.data.OperationSignature); err != nil {
			ctx.Log().WithError(err).Error("Failed to submit confirmation. Maybe already submitted.")
			d.returnToPool(ctx)
		}
		return
	}

	ctx.Log().Infof("Session %s #%d finished unsuccessfully", d.data.SessionType.String(), d.data.SessionId)
	d.returnToPool(ctx)
}

func (d *defaultFinishController) returnToPool(ctx core.Context) {
	// try to return indexes back to the pool
	for _, index := range d.data.Indexes {
		ctx.Pool().Add(index)
	}
}

// updateSessionData updates the database entry according to the controller result.
func (d *defaultFinishController) updateSessionEntry(ctx core.Context) {
	session, err := ctx.PG().DefaultSessionDatumQ().DefaultSessionDatumByID(int64(d.data.SessionId), false)
	if err != nil {
		ctx.Log().WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		ctx.Log().Error("Session entry is not initialized")
		return
	}

	session.Status = int(types.SessionStatus_SessionSucceeded)
	if !d.data.Processing {
		session.Status = int(types.SessionStatus_SessionFailed)
	}

	if err := ctx.PG().DefaultSessionDatumQ().Update(session); err != nil {
		ctx.Log().Error("Error updating session entry")
	}
}

// keygenFinishController represents custom logic for types.SessionType_ReshareSession
type reshareFinishController struct {
	data *LocalSessionData
}

var _ iFinishController = &reshareFinishController{}

// finish in case of successful session updates the TSS secret and submits two messages to the core:
// `rarimo.MsgCreateChangePartiesOp` and `rarimo.MsgCreateConfirmation` - change parties operation and it's confirmation.
func (r *reshareFinishController) finish(ctx core.Context) {
	if r.data.Processing {
		ctx.Log().Infof("Session %s #%d finished successfully", r.data.SessionType.String(), r.data.SessionId)
		if err := ctx.SecretStorage().SetTssSecret(r.data.NewSecret); err != nil {
			panic(err)
		}

		if contains(r.data.Set.UnverifiedParties, ctx.SecretStorage().GetTssSecret().AccountAddress()) {
			ctx.Log().Info("Self party was a new.")
			return
		}

		if !r.data.IsSigner {
			ctx.Log().Info("Self party was not a part of signing round.")
			return
		}

		ctx.Log().Info("Submitting change parties and confirmation messages to finish reshare session.")
		msg1 := &rarimo.MsgCreateChangePartiesOp{
			Creator:      ctx.SecretStorage().GetTssSecret().AccountAddress(),
			NewSet:       r.data.NewParties,
			Signature:    r.data.KeySignature,
			NewPublicKey: r.data.NewSecret.GlobalPubKey(),
		}

		msg2 := &rarimo.MsgCreateConfirmation{
			Creator:        ctx.SecretStorage().GetTssSecret().AccountAddress(),
			Root:           r.data.Root,
			Indexes:        r.data.Indexes,
			SignatureECDSA: r.data.OperationSignature,
		}

		if err := ctx.Core().Submit(msg1, msg2); err != nil {
			ctx.Log().WithError(err).Error("Failed to submit confirmation. Maybe already submitted.")
		}
		return
	}

	ctx.Log().Infof("Session %s #%d finished unsuccessfully", r.data.SessionType.String(), r.data.SessionId)
}

// updateSessionData updates the database entry according to the controller result.
func (r *reshareFinishController) updateSessionEntry(ctx core.Context) {
	session, err := ctx.PG().ReshareSessionDatumQ().ReshareSessionDatumByID(int64(r.data.SessionId), false)
	if err != nil {
		ctx.Log().WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		ctx.Log().Error("Session entry is not initialized")
		return
	}

	session.Status = int(types.SessionStatus_SessionSucceeded)
	if !r.data.Processing {
		session.Status = int(types.SessionStatus_SessionFailed)
	}

	if err := ctx.PG().ReshareSessionDatumQ().Update(session); err != nil {
		ctx.Log().Error("Error updating session entry")
	}
}
