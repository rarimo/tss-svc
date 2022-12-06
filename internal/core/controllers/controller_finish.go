package controllers

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

// FinishController is responsible for finishing sessions. For example: submit transactions, update session entry, etc.
type FinishController struct {
	wg *sync.WaitGroup

	core *connectors.CoreConnector
	log  *logan.Entry

	data *LocalSessionData

	proposer *core.Proposer
	pool     *pool.Pool
	pg       *pg.Storage
	factory  *ControllerFactory
}

// Implements IController interface
var _ IController = &FinishController{}

func (f *FinishController) Receive(*types.MsgSubmitRequest) error {
	return nil
}

func (f *FinishController) Run(context.Context) {
	f.log.Infof("Starting %s", f.Type().String())
	f.wg.Add(1)
	defer func() {
		f.log.Infof("%s finished", f.Type().String())
		f.updateSessionEntry()
		f.wg.Done()
	}()

	if !f.data.Processing {
		f.log.Info("Unsuccessful session")
		// try to return indexes back to the pool
		for _, index := range f.data.Indexes {
			f.pool.Add(index)
		}

		return
	}

	switch f.data.SessionType {
	case types.SessionType_DefaultSession:
		f.finishDefaultSession()
	case types.SessionType_ReshareSession:
		f.finishReshareSession()
	case types.SessionType_KeygenSession:
		f.finishKeygenSession()
	}
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

func (f *FinishController) finishKeygenSession() {
	f.log.Info("Submitting setup initial message to finish keygen session.")
	msg := &rarimo.MsgSetupInitial{
		Creator:        f.data.New.LocalAccountAddress,
		NewPublicKey:   f.data.New.GlobalPubKey,
		PartyPublicKey: f.data.New.LocalPubKey,
	}

	if err := f.core.Submit(msg); err != nil {
		panic(err)
	}
}

func (f *FinishController) finishReshareSession() {
	f.log.Info("Submitting change parties and confirmation messages to finish reshare session.")
	msg1 := &rarimo.MsgCreateChangePartiesOp{
		Creator:      f.data.New.LocalAccountAddress,
		NewSet:       f.data.New.Parties,
		Signature:    f.data.KeySignature,
		NewPublicKey: f.data.NewGlobalPublicKey,
	}

	msg2 := &rarimo.MsgCreateConfirmation{
		Creator:        f.data.New.LocalAccountAddress,
		Root:           f.data.Root,
		Indexes:        f.data.Indexes,
		SignatureECDSA: f.data.OperationSignature,
	}

	if err := f.core.Submit(msg1, msg2); err != nil {
		f.log.WithError(err).Error("Failed to submit confirmation. Maybe already submitted.")
	}
}

func (f *FinishController) finishDefaultSession() {
	f.log.Info("Submitting confirmation message to finish reshare session.")
	if err := f.core.SubmitConfirmation(f.data.Indexes, f.data.Root, f.data.OperationSignature); err != nil {
		f.log.WithError(err).Error("Failed to submit confirmation. Maybe already submitted.")
	}
}

func (f *FinishController) updateSessionEntry() {
	session, err := f.pg.SessionQ().SessionByID(int64(f.data.SessionId), false)
	if err != nil {
		f.log.WithError(err).Error("error selecting session")
		return
	}

	if session == nil {
		f.log.Error("session entry is not initialized")
		return
	}

	session.Status = int(types.SessionStatus_SessionSucceeded)
	if !f.data.Processing {
		session.Status = int(types.SessionStatus_SessionFailed)
	}

	if err := f.pg.SessionQ().Update(session); err != nil {
		f.log.Error("error updating session entry")
	}
}
