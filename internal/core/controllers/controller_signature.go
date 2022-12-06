package controllers

import (
	"context"
	"database/sql"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/crypto/pkg"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureController struct {
	ISignatureController
	mu   sync.Mutex
	wg   *sync.WaitGroup
	data *LocalSessionData

	auth  *core.RequestAuthorizer
	log   *logan.Entry
	party *tss.SignParty
}

var _ IController = &SignatureController{}

func (s *SignatureController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := s.auth.Auth(request)
	if err != nil {
		return err
	}

	if _, ok := s.data.Acceptances[sender.Account]; !ok {
		return ErrSenderHasNotAccepted
	}

	if request.Type != types.RequestType_Sign {
		return ErrInvalidRequestType
	}

	sign := new(types.SignRequest)
	if err := proto.Unmarshal(request.Details.Value, sign); err != nil {
		return errors.Wrap(err, "error unmarshalling request")
	}

	if sign.Data == s.party.Data() {
		s.party.Receive(sender, request.IsBroadcast, sign.Details.Value)
	}
	return nil
}

func (s *SignatureController) Run(ctx context.Context) {
	s.log.Infof("Starting %s controller", s.Type().String())
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
		s.log.Infof("%s finished", s.Type().String())
		s.updateSessionData()
		s.wg.Done()
	}()

	<-ctx.Done()

	s.party.WaitFor()

	s.mu.Lock()
	defer s.mu.Unlock()

	result := s.party.Result()
	if result == nil {
		s.data.Processing = false
		return
	}

	signature := hexutil.Encode(append(result.Signature, result.SignatureRecovery...))
	s.finish(signature)
}

type ISignatureController interface {
	Next() IController
	finish(signature string)
	updateSessionData()
}

type KeySignatureController struct {
	data    *LocalSessionData
	factory *ControllerFactory
	pg      *pg.Storage
	log     *logan.Entry
}

var _ ISignatureController = &KeySignatureController{}

func (s *KeySignatureController) Next() IController {
	if s.data.Processing {
		op := &rarimo.ChangeParties{
			Parties:   s.data.New.Parties,
			Signature: s.data.KeySignature,
		}
		content, _ := pkg.GetChangePartiesContent(op)
		s.data.Root = hexutil.Encode(content.CalculateHash())
		s.data.Indexes = []string{s.data.Root}
		return s.factory.GetRootSignController(s.data.Root)
	}
	return s.factory.GetFinishController()
}

func (s *KeySignatureController) finish(signature string) {
	s.data.KeySignature = signature
}

func (s *KeySignatureController) updateSessionData() {
	session, err := s.pg.SessionQ().SessionByID(int64(s.data.SessionId), false)
	if err != nil {
		s.log.WithError(err).Error("error selecting session")
		return
	}

	if session == nil {
		s.log.Error("session entry is not initialized")
		return
	}

	data, err := s.pg.ReshareSessionDatumQ().ReshareSessionDatumByID(session.DataID.Int64, false)
	if err != nil {
		s.log.WithError(err).Error("error selecting session data")
		return
	}

	if data == nil {
		s.log.Error("session data is not initialized")
		return
	}

	data.Signature = sql.NullString{
		String: s.data.OperationSignature,
		Valid:  s.data.OperationSignature != "",
	}

	if err = s.pg.ReshareSessionDatumQ().Update(data); err != nil {
		s.log.WithError(err).Error("error updating session data entry")
	}
}

type RootSignatureController struct {
	data    *LocalSessionData
	factory *ControllerFactory
	pg      *pg.Storage
	log     *logan.Entry
}

var _ ISignatureController = &RootSignatureController{}

func (s *RootSignatureController) Next() IController {
	return s.factory.GetFinishController()
}

func (s *RootSignatureController) finish(signature string) {
	s.data.OperationSignature = signature
}

func (s *RootSignatureController) updateSessionData() {
	session, err := s.pg.SessionQ().SessionByID(int64(s.data.SessionId), false)
	if err != nil {
		s.log.WithError(err).Error("error selecting session")
		return
	}

	if session == nil {
		s.log.Error("session entry is not initialized")
		return
	}

	switch s.data.SessionType {
	case types.SessionType_DefaultSession:
		data, err := s.pg.DefaultSessionDatumQ().DefaultSessionDatumByID(session.DataID.Int64, false)
		if err != nil {
			s.log.WithError(err).Error("error selecting session data")
			return
		}

		if data == nil {
			s.log.Error("session data is not initialized")
			return
		}

		data.Signature = sql.NullString{
			String: s.data.OperationSignature,
			Valid:  s.data.OperationSignature != "",
		}

		err = s.pg.DefaultSessionDatumQ().Update(data)
	case types.SessionType_ReshareSession:
		data, err := s.pg.ReshareSessionDatumQ().ReshareSessionDatumByID(session.DataID.Int64, false)
		if err != nil {
			s.log.WithError(err).Error("error selecting session data")
			return
		}

		if data == nil {
			s.log.Error("session data is not initialized")
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

	if err != nil {
		s.log.WithError(err).Error("error updating session data entry")
	}
}
