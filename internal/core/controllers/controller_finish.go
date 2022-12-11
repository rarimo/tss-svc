package controllers

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

// FinishController is responsible for finishing sessions. For example: submit transactions, update session entry, etc.
type FinishController struct {
	IFinishController
	wg *sync.WaitGroup

	data *LocalSessionData

	pg  *pg.Storage
	log *logan.Entry
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

func (f *FinishController) updateSessionEntry() {
	session, err := f.pg.SessionQ().SessionByID(int64(f.data.SessionId), false)
	if err != nil {
		f.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		f.log.Error("Session entry is not initialized")
		return
	}

	session.Status = int(types.SessionStatus_SessionSucceeded)
	if !f.data.Processing {
		session.Status = int(types.SessionStatus_SessionFailed)
	}

	if err := f.pg.SessionQ().Update(session); err != nil {
		f.log.Error("Error updating session entry")
	}
}

type IFinishController interface {
	finish()
}

type KeygenFinishController struct {
	data    *LocalSessionData
	storage secret.Storage
	core    *connectors.CoreConnector
	log     *logan.Entry
}

var _ IFinishController = &KeygenFinishController{}

func (k *KeygenFinishController) finish() {
	if k.data.Processing {
		k.log.Infof("Session %d finished successfully", k.data.SessionId)
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

	k.log.Infof("Session %d finished unsuccessfully", k.data.SessionId)
}

type DefaultFinishController struct {
	data *LocalSessionData
	pool *pool.Pool
	core *connectors.CoreConnector
	log  *logan.Entry
}

var _ IFinishController = &DefaultFinishController{}

func (d *DefaultFinishController) finish() {
	if d.data.Processing {
		d.log.Infof("Session %d finished successfully", d.data.SessionId)
		d.log.Info("Submitting confirmation message to finish default session.")
		if err := d.core.SubmitConfirmation(d.data.Indexes, d.data.Root, d.data.OperationSignature); err != nil {
			d.log.WithError(err).Error("Failed to submit confirmation. Maybe already submitted.")
		}
		return
	}

	d.log.Infof("Session %d finished unsuccessfully", d.data.SessionId)

	// try to return indexes back to the pool
	for _, index := range d.data.Indexes {
		d.pool.Add(index)
	}
}

type ReshareFinishController struct {
	data    *LocalSessionData
	storage secret.Storage
	core    *connectors.CoreConnector
	log     *logan.Entry
}

var _ IFinishController = &ReshareFinishController{}

func (r *ReshareFinishController) finish() {
	if r.data.Processing {
		r.log.Infof("Session %d finished successfully", r.data.SessionId)
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

	r.log.Infof("Session %d finished unsuccessfully", r.data.SessionId)
}
