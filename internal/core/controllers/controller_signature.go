package controllers

import (
	"context"
	"database/sql"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/rarimo-core/x/rarimocore/crypto/pkg"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/tss"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// iSignatureController defines custom logic for every signature controller.
type iSignatureController interface {
	Next() IController
	finish(signature string)
	updateSessionData(ctx core.Context)
}

// SignatureController is responsible for signing data by signature producers.
type SignatureController struct {
	iSignatureController
	wg    *sync.WaitGroup
	data  *LocalSessionData
	auth  *core.RequestAuthorizer
	party *tss.SignParty
}

// Implements IController interface
var _ IController = &SignatureController{}

// Receive accepts the signature requests  from other parties and delivers it to the `tss.SignParty.
// If sender is not present in current signers set request will not be accepted.
func (s *SignatureController) Receive(c context.Context, request *types.MsgSubmitRequest) error {
	sender, err := s.auth.Auth(request)
	if err != nil {
		return err
	}

	if _, ok := s.data.Signers[sender.Account]; !ok {
		s.data.Offenders[sender.Account] = struct{}{}
		return ErrSenderIsNotSigner
	}

	if request.Type != types.RequestType_Sign {
		return ErrInvalidRequestType
	}

	sign := new(types.SignRequest)
	if err := request.Details.UnmarshalTo(sign); err != nil {
		return errors.Wrap(err, "error unmarshalling request")
	}

	if sign.Data != s.party.Data() {
		s.data.Offenders[sender.Account] = struct{}{}
		return nil
	}

	if err := s.party.Receive(sender, request.IsBroadcast, sign.Details.Value); err != nil {
		ctx := core.WrapCtx(c)
		ctx.Log().WithError(err).Error("failed to receive request on party")
		// can be done without lock: no remove or change operation exist, only add
		s.data.Offenders[sender.Account] = struct{}{}
	}

	return nil
}

// Run launches the `tss.SignParty` logic. After context canceling it will check the tss party result
// and execute `iSignatureController.finish` logic.
func (s *SignatureController) Run(c context.Context) {
	ctx := core.WrapCtx(c)
	ctx.Log().Infof("Starting: %s", s.Type().String())
	s.party.Run(c)
	s.wg.Add(1)
	go s.run(ctx)
}

// WaitFor waits until controller finishes its logic. Context cancel should be called before.
func (s *SignatureController) WaitFor() {
	s.wg.Wait()
}

func (s *SignatureController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_SIGN
}

func (s *SignatureController) run(ctx core.Context) {
	defer func() {
		ctx.Log().Infof("Finishing: %s", s.Type().String())
		s.updateSessionData(ctx)
		s.wg.Done()
	}()

	<-ctx.Context().Done()

	s.party.WaitFor()

	result := s.party.Result()
	if result == nil {
		s.data.Processing = false
		return
	}

	signature := hexutil.Encode(append(result.Signature, result.SignatureRecovery...))
	s.finish(signature)
}

// keySignatureController represents custom logic for types.SessionType_ReshareSession for signing the new key with old signature.
type keySignatureController struct {
	data *LocalSessionData
}

// Implements iSignatureController interface
var _ iSignatureController = &keySignatureController{}

// Next will return the root signature controller if current signing was successful.
// Otherwise, it will return finish controller.
// WaitFor should be called before.
func (s *keySignatureController) Next() IController {
	if s.data.Processing {
		return s.data.GetRootSignController()
	}
	return s.data.GetFinishController()
}

// finish will store the result signature and generates the ChangeParties operation to be signed in the next controller.
func (s *keySignatureController) finish(signature string) {
	s.data.KeySignature = signature
	op := &rarimo.ChangeParties{
		Parties:      s.data.NewParties,
		NewPublicKey: s.data.NewSecret.GlobalPubKey(),
		Signature:    s.data.KeySignature,
	}
	content, _ := pkg.GetChangePartiesContent(op)
	s.data.Root = hexutil.Encode(content.CalculateHash())
	s.data.Indexes = []string{s.data.Root}
}

// updateSessionData updates the database entry according to the controller result.
func (s *keySignatureController) updateSessionData(ctx core.Context) {
	session, err := ctx.PG().ReshareSessionDatumQ().ReshareSessionDatumByID(int64(s.data.SessionId), false)
	if err != nil {
		ctx.Log().WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		ctx.Log().Error("Session entry is not initialized")
		return
	}

	session.Signature = sql.NullString{
		String: s.data.OperationSignature,
		Valid:  s.data.OperationSignature != "",
	}

	if err = ctx.PG().ReshareSessionDatumQ().Update(session); err != nil {
		ctx.Log().WithError(err).Error("Error updating session entry")
	}
}

// rootSignatureController represents custom logic for both types.SessionType_ReshareSession
// and types.SessionType_DefaultSession for signing the root of indexes set.
type rootSignatureController struct {
	data *LocalSessionData
}

// Implements iSignatureController interface
var _ iSignatureController = &rootSignatureController{}

// Next returns the finish controller instance.
// WaitFor should be called before.
func (s *rootSignatureController) Next() IController {
	return s.data.GetFinishController()
}

// finish saves the generated signature
func (s *rootSignatureController) finish(signature string) {
	s.data.OperationSignature = signature
}

// updateSessionData updates the database entry according to the controller result.
func (s *rootSignatureController) updateSessionData(ctx core.Context) {
	switch s.data.SessionType {
	case types.SessionType_DefaultSession:
		data, err := ctx.PG().DefaultSessionDatumQ().DefaultSessionDatumByID(int64(s.data.SessionId), false)
		if err != nil {
			ctx.Log().WithError(err).Error("Error selecting session data")
			return
		}

		if data == nil {
			ctx.Log().Error("Session data is not initialized")
			return
		}

		data.Signature = sql.NullString{
			String: s.data.OperationSignature,
			Valid:  s.data.OperationSignature != "",
		}

		err = ctx.PG().DefaultSessionDatumQ().Update(data)
	case types.SessionType_ReshareSession:
		data, err := ctx.PG().ReshareSessionDatumQ().ReshareSessionDatumByID(int64(s.data.SessionId), false)
		if err != nil {
			ctx.Log().WithError(err).Error("Error selecting session data")
			return
		}

		if data == nil {
			ctx.Log().Error("Session data is not initialized")
			return
		}

		data.KeySignature = sql.NullString{
			String: s.data.KeySignature,
			Valid:  s.data.KeySignature != "",
		}
		data.Root = sql.NullString{
			String: s.data.Root,
			Valid:  true,
		}

		err = ctx.PG().ReshareSessionDatumQ().Update(data)
	}
}
