package controllers

import (
	"context"
	"database/sql"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/rarimo-core/x/rarimocore/crypto/pkg"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"gitlab.com/rarimo/tss/tss-svc/internal/tss"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// SignatureController is responsible for signing data by signature producers.
type SignatureController struct {
	ISignatureController
	wg   *sync.WaitGroup
	data *LocalSessionData

	auth  *core.RequestAuthorizer
	log   *logan.Entry
	party *tss.SignParty
}

// Implements IController interface
var _ IController = &SignatureController{}

func (s *SignatureController) Receive(request *types.MsgSubmitRequest) error {
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
	if err := proto.Unmarshal(request.Details.Value, sign); err != nil {
		return errors.Wrap(err, "error unmarshalling request")
	}

	if sign.Data != s.party.Data() {
		s.data.Offenders[sender.Account] = struct{}{}
		return nil
	}

	go func() {
		if err := s.party.Receive(sender, request.IsBroadcast, sign.Details.Value); err != nil {
			// can be done without lock: no remove or change operation exist, only add
			s.data.Offenders[sender.Account] = struct{}{}
		}
	}()

	return nil
}

func (s *SignatureController) Run(ctx context.Context) {
	s.log.Infof("Starting: %s", s.Type().String())
	s.party.Run(ctx)
	s.wg.Add(1)
	go s.run(ctx)
}

func (s *SignatureController) WaitFor() {
	s.wg.Wait()
}

func (s *SignatureController) Type() types.ControllerType {
	return types.ControllerType_CONTROLLER_SIGN
}

func (s *SignatureController) run(ctx context.Context) {
	defer func() {
		s.log.Infof("Finishing: %s", s.Type().String())
		s.updateSessionData()
		s.wg.Done()
	}()

	<-ctx.Done()

	s.party.WaitFor()

	result := s.party.Result()
	if result == nil {
		s.data.Processing = false
		return
	}

	signature := hexutil.Encode(append(result.Signature, result.SignatureRecovery...))
	s.finish(signature)
}

// ISignatureController defines custom logic for every signature controller.
type ISignatureController interface {
	Next() IController
	finish(signature string)
	updateSessionData()
}

// KeySignatureController represents custom logic for types.SessionType_ReshareSession for signing the new key with old signature.
type KeySignatureController struct {
	data    *LocalSessionData
	factory *ControllerFactory
	pg      *pg.Storage
	log     *logan.Entry
}

// Implements ISignatureController interface
var _ ISignatureController = &KeySignatureController{}

func (s *KeySignatureController) Next() IController {
	if s.data.Processing {
		return s.factory.GetRootSignController(s.data.Root)
	}
	return s.factory.GetFinishController()
}

func (s *KeySignatureController) finish(signature string) {
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

func (s *KeySignatureController) updateSessionData() {
	session, err := s.pg.ReshareSessionDatumQ().ReshareSessionDatumByID(int64(s.data.SessionId), false)
	if err != nil {
		s.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		s.log.Error("Session entry is not initialized")
		return
	}

	session.Signature = sql.NullString{
		String: s.data.OperationSignature,
		Valid:  s.data.OperationSignature != "",
	}

	if err = s.pg.ReshareSessionDatumQ().Update(session); err != nil {
		s.log.WithError(err).Error("Error updating session entry")
	}
}

// RootSignatureController represents custom logic for both types.SessionType_ReshareSession
// and types.SessionType_DefaultSession for signing the root of indexes set.
type RootSignatureController struct {
	data    *LocalSessionData
	factory *ControllerFactory
	pg      *pg.Storage
	log     *logan.Entry
}

// Implements ISignatureController interface
var _ ISignatureController = &RootSignatureController{}

func (s *RootSignatureController) Next() IController {
	return s.factory.GetFinishController()
}

func (s *RootSignatureController) finish(signature string) {
	s.data.OperationSignature = signature
}

func (s *RootSignatureController) updateSessionData() {
	switch s.data.SessionType {
	case types.SessionType_DefaultSession:
		data, err := s.pg.DefaultSessionDatumQ().DefaultSessionDatumByID(int64(s.data.SessionId), false)
		if err != nil {
			s.log.WithError(err).Error("Error selecting session data")
			return
		}

		if data == nil {
			s.log.Error("Session data is not initialized")
			return
		}

		data.Signature = sql.NullString{
			String: s.data.OperationSignature,
			Valid:  s.data.OperationSignature != "",
		}

		err = s.pg.DefaultSessionDatumQ().Update(data)
	case types.SessionType_ReshareSession:
		data, err := s.pg.ReshareSessionDatumQ().ReshareSessionDatumByID(int64(s.data.SessionId), false)
		if err != nil {
			s.log.WithError(err).Error("Error selecting session data")
			return
		}

		if data == nil {
			s.log.Error("Session data is not initialized")
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

		err = s.pg.ReshareSessionDatumQ().Update(data)
	}
}
