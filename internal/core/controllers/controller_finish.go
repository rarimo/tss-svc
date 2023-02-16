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

// FinishController is responsible for finishing sessions. For example: submit transactions, update session entry, etc.
type FinishController struct {
	IFinishController
	wg *sync.WaitGroup

	data *LocalSessionData
	log  *logan.Entry
}

// Implements IController interface
var _ IController = &FinishController{}

func (f *FinishController) Receive(*types.MsgSubmitRequest) error {
	return nil
}

func (f *FinishController) Run(context.Context) {
	f.log.Infof("Starting: %s", f.Type().String())
	f.wg.Add(1)
	defer func() {
		f.log.Infof("Finishing: %s", f.Type().String())
		f.updateSessionEntry()
		f.wg.Done()
	}()

	f.finish()
}

func (f *FinishController) WaitFor() {
	f.wg.Wait()
}

func (f *FinishController) Next() IController {
	return nil
}

func (f *FinishController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_FINISH
}

type IFinishController interface {
	finish()
	updateSessionEntry()
}

type KeygenFinishController struct {
	data    *LocalSessionData
	storage secret.Storage
	core    *connectors.CoreConnector
	pg      *pg.Storage
	log     *logan.Entry
}

var _ IFinishController = &KeygenFinishController{}

func (k *KeygenFinishController) finish() {
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

func (k *KeygenFinishController) updateSessionEntry() {
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

type DefaultFinishController struct {
	data *LocalSessionData
	core *connectors.CoreConnector
	pg   *pg.Storage
	log  *logan.Entry
}

var _ IFinishController = &DefaultFinishController{}

func (d *DefaultFinishController) finish() {
	if d.data.Processing {
		d.log.Infof("Session %s #%d finished successfully", d.data.SessionType.String(), d.data.SessionId)
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

func (d *DefaultFinishController) returnToPool() {
	// try to return indexes back to the pool
	for _, index := range d.data.Indexes {
		pool.GetPool().Add(index)
	}
}

func (d *DefaultFinishController) updateSessionEntry() {
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

type ReshareFinishController struct {
	data    *LocalSessionData
	storage secret.Storage
	core    *connectors.CoreConnector
	pg      *pg.Storage
	log     *logan.Entry
}

var _ IFinishController = &ReshareFinishController{}

func (r *ReshareFinishController) finish() {
	if r.data.Processing {
		r.log.Infof("Session %s #%d finished successfully", r.data.SessionType.String(), r.data.SessionId)
		if err := r.storage.SetTssSecret(r.data.NewSecret); err != nil {
			panic(err)
		}

		if contains(r.data.Set.UnverifiedParties, r.data.Secret.AccountAddress()) {
			// That party is new one, no additional operations required
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

func (r *ReshareFinishController) updateSessionEntry() {
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
