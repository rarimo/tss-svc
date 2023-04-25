package controllers

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/connectors"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"gitlab.com/rarimo/tss/tss-svc/internal/pool"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// iFinishController defines custom logic for every finish controller.
type iFinishController interface {
	finish()
	updateSessionEntry()
}

// FinishController is responsible for finishing sessions. For example: submit transactions, update session entry, etc.
type FinishController struct {
	iFinishController
	wg *sync.WaitGroup

	core *connectors.CoreConnector
	data *LocalSessionData
	log  *logan.Entry
}

// Implements IController interface
var _ IController = &FinishController{}

func (f *FinishController) Receive(*types.MsgSubmitRequest) error {
	return nil
}

// Run initiates the report submitting for all parties that was included into Offenders set. After it executes the
// `iFinishController.finish` logic.
func (f *FinishController) Run(context.Context) {
	f.log.Infof("Starting: %s", f.Type().String())
	f.wg.Add(1)
	defer func() {
		f.log.Infof("Finishing: %s", f.Type().String())
		f.updateSessionEntry()
		f.wg.Done()
	}()

	for offender := range f.data.Offenders {
		if err := f.core.SubmitReport(
			f.data.SessionId,
			rarimo.ViolationType_Spam,
			offender,
			"Party shared invalid data or have not accepted valid proposal",
		); err != nil {
			f.log.WithError(err).Errorf("Error submitting violation report for party: %s", offender)
		}
	}

	f.finish()
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
	data    *LocalSessionData
	storage secret.Storage
	core    *connectors.CoreConnector
	pg      *pg.Storage
	log     *logan.Entry
}

var _ iFinishController = &keygenFinishController{}

// finish in case of successful session updates the stores TSS secret and submits the `rarimo.MsgSetupInitial` message.
func (k *keygenFinishController) finish() {
	if k.data.Processing {
		k.log.Infof("Session %s #%d finished successfully", k.data.SessionType.String(), k.data.SessionId)
		if err := k.storage.SetTssSecret(k.data.NewSecret); err != nil {
			panic(err)
		}

		k.log.Info("Submitting setup initial message to finish keygen session.")
		msg := &rarimo.MsgSetupInitial{
			Creator:        k.data.NewSecret.AccountAddress(),
			NewPublicKey:   k.data.NewSecret.GlobalPubKey(),
			PartyPublicKey: k.data.NewSecret.TssPubKey(),
		}

		if err := k.core.Submit(msg); err != nil {
			panic(err)
		}
		return
	}

	k.log.Infof("Session %s #%d finished unsuccessfully", k.data.SessionType.String(), k.data.SessionId)
}

// updateSessionData updates the database entry according to the controller result.
func (k *keygenFinishController) updateSessionEntry() {
	session, err := k.pg.KeygenSessionDatumQ().KeygenSessionDatumByID(int64(k.data.SessionId), false)
	if err != nil {
		k.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		k.log.Error("Session entry is not initialized")
		return
	}

	session.Status = int(types.SessionStatus_SessionSucceeded)
	if !k.data.Processing {
		session.Status = int(types.SessionStatus_SessionFailed)
	}

	if err := k.pg.KeygenSessionDatumQ().Update(session); err != nil {
		k.log.Error("Error updating session entry")
	}
}

// keygenFinishController represents custom logic for types.SessionType_DefaultSession
type defaultFinishController struct {
	data *LocalSessionData
	core *connectors.CoreConnector
	pg   *pg.Storage
	log  *logan.Entry
}

var _ iFinishController = &defaultFinishController{}

// finish in case of successful session checks that self party was a signer.
// If true, it will share the generated signature via submitting confirmation message to the core.
// In case of unsuccessful session the selected indexes will be returned to the pool.
func (d *defaultFinishController) finish() {
	if d.data.Processing {
		d.log.Infof("Session %s #%d finished successfully", d.data.SessionType.String(), d.data.SessionId)
		if _, ok := d.data.Signers[d.data.Secret.AccountAddress()]; !ok {
			d.log.Info("Self party was not a part of signing round")
			return
		}

		d.log.Info("Submitting confirmation message to finish default session.")
		if err := d.core.SubmitConfirmation(d.data.Indexes, d.data.Root, d.data.OperationSignature); err != nil {
			d.log.WithError(err).Error("Failed to submit confirmation. Maybe already submitted.")
			d.returnToPool()
		}
		return
	}

	d.log.Infof("Session %s #%d finished unsuccessfully", d.data.SessionType.String(), d.data.SessionId)
	d.returnToPool()
}

func (d *defaultFinishController) returnToPool() {
	// try to return indexes back to the pool
	for _, index := range d.data.Indexes {
		pool.GetPool().Add(index)
	}
}

// updateSessionData updates the database entry according to the controller result.
func (d *defaultFinishController) updateSessionEntry() {
	session, err := d.pg.DefaultSessionDatumQ().DefaultSessionDatumByID(int64(d.data.SessionId), false)
	if err != nil {
		d.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		d.log.Error("Session entry is not initialized")
		return
	}

	session.Status = int(types.SessionStatus_SessionSucceeded)
	if !d.data.Processing {
		session.Status = int(types.SessionStatus_SessionFailed)
	}

	if err := d.pg.DefaultSessionDatumQ().Update(session); err != nil {
		d.log.Error("Error updating session entry")
	}
}

// keygenFinishController represents custom logic for types.SessionType_ReshareSession
type reshareFinishController struct {
	data    *LocalSessionData
	storage secret.Storage
	core    *connectors.CoreConnector
	pg      *pg.Storage
	log     *logan.Entry
}

var _ iFinishController = &reshareFinishController{}

// finish in case of successful session updates the TSS secret and submits two messages to the core:
// `rarimo.MsgCreateChangePartiesOp` and `rarimo.MsgCreateConfirmation` - change parties operation and it's confirmation.
func (r *reshareFinishController) finish() {
	if r.data.Processing {
		r.log.Infof("Session %s #%d finished successfully", r.data.SessionType.String(), r.data.SessionId)
		if err := r.storage.SetTssSecret(r.data.NewSecret); err != nil {
			panic(err)
		}

		if contains(r.data.Set.UnverifiedParties, r.data.Secret.AccountAddress()) {
			r.log.Info("Self party was a new.")
			return
		}

		if _, ok := r.data.Signers[r.data.Secret.AccountAddress()]; !ok {
			r.log.Info("Self party was not a part of signing round.")
			return
		}

		r.log.Info("Submitting change parties and confirmation messages to finish reshare session.")
		msg1 := &rarimo.MsgCreateChangePartiesOp{
			Creator:      r.data.Secret.AccountAddress(),
			NewSet:       r.data.NewParties,
			Signature:    r.data.KeySignature,
			NewPublicKey: r.data.NewSecret.GlobalPubKey(),
		}

		msg2 := &rarimo.MsgCreateConfirmation{
			Creator:        r.data.Secret.AccountAddress(),
			Root:           r.data.Root,
			Indexes:        r.data.Indexes,
			SignatureECDSA: r.data.OperationSignature,
		}

		if err := r.core.Submit(msg1, msg2); err != nil {
			r.log.WithError(err).Error("Failed to submit confirmation. Maybe already submitted.")
		}
		return
	}

	r.log.Infof("Session %s #%d finished unsuccessfully", r.data.SessionType.String(), r.data.SessionId)
}

// updateSessionData updates the database entry according to the controller result.
func (r *reshareFinishController) updateSessionEntry() {
	session, err := r.pg.ReshareSessionDatumQ().ReshareSessionDatumByID(int64(r.data.SessionId), false)
	if err != nil {
		r.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		r.log.Error("Session entry is not initialized")
		return
	}

	session.Status = int(types.SessionStatus_SessionSucceeded)
	if !r.data.Processing {
		session.Status = int(types.SessionStatus_SessionFailed)
	}

	if err := r.pg.ReshareSessionDatumQ().Update(session); err != nil {
		r.log.Error("Error updating session entry")
	}
}
