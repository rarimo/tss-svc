package grpc

import (
	"context"
	"net"

	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"gitlab.com/rarimo/tss/tss-svc/internal/pool"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ServerImpl struct {
	types.UnimplementedServiceServer
	manager  *core.SessionManager
	log      *logan.Entry
	listener net.Listener
	pg       *pg.Storage
	storage  secret.Storage
	pool     *pool.Pool
}

func NewServer(manager *core.SessionManager, cfg config.Config) *ServerImpl {
	return &ServerImpl{
		manager:  manager,
		log:      cfg.Log(),
		listener: cfg.Listener(),
		pg:       cfg.Storage(),
		storage:  secret.NewLocalStorage(cfg),
		pool:     pool.NewPool(cfg),
	}
}

func (s *ServerImpl) Run() error {
	grpcServer := grpc.NewServer()
	types.RegisterServiceServer(grpcServer, s)
	return grpcServer.Serve(s.listener)
}

var _ types.ServiceServer = &ServerImpl{}

func (s *ServerImpl) Submit(_ context.Context, request *types.MsgSubmitRequest) (*types.MsgSubmitResponse, error) {
	return &types.MsgSubmitResponse{}, s.manager.Receive(request)
}

func (s *ServerImpl) AddOperation(_ context.Context, request *types.MsgAddOperationRequest) (*types.MsgAddOperationResponse, error) {
	err := s.pool.Add(request.Index)
	if err != nil {
		s.log.WithError(err).Error("[GRPC] Error adding to the pool")
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request index: maybe already signed")
	}
	return &types.MsgAddOperationResponse{}, nil
}

func (s *ServerImpl) Info(_ context.Context, _ *types.MsgInfoRequest) (*types.MsgInfoResponse, error) {
	id := s.manager.ID()
	session, err := s.getSessionResp(id)
	if err != nil {
		return nil, err
	}

	return &types.MsgInfoResponse{
		LocalAccount:     s.storage.GetTssSecret().AccountAddress(),
		LocalPublicKey:   s.storage.GetTssSecret().TssPubKey(),
		CurrentSessionId: id,
		SessionData:      session,
	}, nil
}

func (s *ServerImpl) Session(_ context.Context, request *types.MsgSessionRequest) (*types.MsgSessionResponse, error) {
	session, err := s.getSessionResp(request.Id)
	if err != nil {
		return nil, err
	}

	return &types.MsgSessionResponse{
		Data: session,
	}, nil
}

func (s *ServerImpl) getSessionResp(id uint64) (*types.Session, error) {
	session, err := s.pg.SessionQ().SessionByID(int64(id), false)
	if err != nil {
		s.log.WithError(err).Error("[GRPC] Error selecting current session by id")
		return nil, status.Error(codes.Internal, "Internal error")
	}

	if session == nil {
		return nil, status.Error(codes.NotFound, "Current session not found")
	}

	var details *cosmostypes.Any

	switch session.SessionType.Int64 {
	case int64(types.SessionType_DefaultSession):
		data, err := s.pg.DefaultSessionDatumQ().DefaultSessionDatumByID(session.DataID.Int64, false)
		if err != nil {
			s.log.WithError(err).Error("[GRPC] Error selecting current session data by id")
			return nil, status.Error(codes.Internal, "Internal error")
		}

		if data == nil {
			break
		}

		details, err = cosmostypes.NewAnyWithValue(&types.DefaultSessionData{
			Parties:   data.Parties,
			Proposer:  data.Proposer.String,
			Indexes:   data.Indexes,
			Root:      data.Root.String,
			Accepted:  data.Accepted,
			Signature: data.Signature.String,
		})

	case int64(types.SessionType_ReshareSession):
		data, err := s.pg.ReshareSessionDatumQ().ReshareSessionDatumByID(session.DataID.Int64, false)
		if err != nil {
			s.log.WithError(err).Error("[GRPC] Error selecting current session data by id")
			return nil, status.Error(codes.Internal, "Internal error")
		}

		if data == nil {
			break
		}

		details, err = cosmostypes.NewAnyWithValue(&types.ReshareSessionData{
			Parties:      data.Parties,
			Proposer:     data.Proposer.String,
			OldKey:       data.OldKey.String,
			NewKey:       data.NewKey.String,
			Root:         data.Root.String,
			KeySignature: data.KeySignature.String,
			Signature:    data.Signature.String,
		})

	case int64(types.SessionType_KeygenSession):
		data, err := s.pg.KeygenSessionDatumQ().KeygenSessionDatumByID(session.DataID.Int64, false)
		if err != nil {
			s.log.WithError(err).Error("[GRPC] Error selecting current session data by id")
			return nil, status.Error(codes.Internal, "Internal error")
		}

		if data == nil {
			break
		}

		details, err = cosmostypes.NewAnyWithValue(&types.KeygenSessionData{
			Parties: data.Parties,
			Key:     data.Key.String,
		})
	}

	if err != nil {
		s.log.WithError(err).Error("[GRPC] Failed to parse details data")
		return nil, status.Error(codes.Internal, "Internal error")
	}

	return &types.Session{
		Id:         uint64(session.ID),
		Status:     types.SessionStatus(session.Status),
		StartBlock: uint64(session.BeginBlock),
		EndBlock:   uint64(session.EndBlock),
		Type:       types.SessionType(session.SessionType.Int64),
		Data:       details,
	}, nil
}
