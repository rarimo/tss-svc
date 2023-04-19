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
		storage:  secret.NewVaultStorage(cfg),
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
	if err := s.manager.Receive(request); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	return &types.MsgSubmitResponse{}, nil
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
	sessions := make(map[string]*types.Session)
	sessionTypes := []types.SessionType{types.SessionType_DefaultSession, types.SessionType_ReshareSession, types.SessionType_KeygenSession}

	for _, sessionType := range sessionTypes {
		id, ok := s.manager.ID(sessionType)
		if ok {
			session, err := s.getSessionResp(sessionType, int64(id))
			if err != nil {
				return nil, err
			}

			sessions[sessionType.String()] = session
		}
	}

	return &types.MsgInfoResponse{
		LocalAccount:   s.storage.GetTssSecret().AccountAddress(),
		LocalPublicKey: s.storage.GetTssSecret().TssPubKey(),
		Sessions:       sessions,
	}, nil
}

func (s *ServerImpl) Session(_ context.Context, request *types.MsgSessionRequest) (*types.MsgSessionResponse, error) {
	session, err := s.getSessionResp(request.SessionType, int64(request.Id))
	if err != nil {
		return nil, err
	}

	if session == nil {
		return nil, status.Errorf(codes.NotFound, "Session not found")
	}

	return &types.MsgSessionResponse{
		Data: session,
	}, nil
}

func (s *ServerImpl) getSessionResp(sessionType types.SessionType, id int64) (*types.Session, error) {
	switch sessionType {
	case types.SessionType_DefaultSession:
		session, err := s.pg.DefaultSessionDatumQ().DefaultSessionDatumByID(id, false)
		if err != nil {
			s.log.WithError(err).Error("[GRPC] Error selecting session by id")
			return nil, status.Error(codes.Internal, "Internal error")
		}

		if session == nil {
			return nil, nil
		}

		details, _ := cosmostypes.NewAnyWithValue(&types.DefaultSessionData{
			Parties:   session.Parties,
			Proposer:  session.Proposer.String,
			Indexes:   session.Indexes,
			Root:      session.Root.String,
			Accepted:  session.Accepted,
			Signature: session.Signature.String,
		})

		return &types.Session{
			Id:         uint64(session.ID),
			Status:     types.SessionStatus(session.Status),
			StartBlock: uint64(session.BeginBlock),
			EndBlock:   uint64(session.EndBlock),
			Type:       types.SessionType_DefaultSession,
			Data:       details,
		}, nil

	case types.SessionType_ReshareSession:
		session, err := s.pg.ReshareSessionDatumQ().ReshareSessionDatumByID(id, false)
		if err != nil {
			s.log.WithError(err).Error("[GRPC] Error selecting session by id")
			return nil, status.Error(codes.Internal, "Internal error")
		}

		if session == nil {
			return nil, nil
		}

		details, _ := cosmostypes.NewAnyWithValue(&types.ReshareSessionData{
			Parties:      session.Parties,
			Proposer:     session.Proposer.String,
			OldKey:       session.OldKey.String,
			NewKey:       session.NewKey.String,
			Root:         session.Root.String,
			KeySignature: session.KeySignature.String,
			Signature:    session.Signature.String,
		})

		return &types.Session{
			Id:         uint64(session.ID),
			Status:     types.SessionStatus(session.Status),
			StartBlock: uint64(session.BeginBlock),
			EndBlock:   uint64(session.EndBlock),
			Type:       types.SessionType_ReshareSession,
			Data:       details,
		}, nil

	case types.SessionType_KeygenSession:
		session, err := s.pg.KeygenSessionDatumQ().KeygenSessionDatumByID(id, false)
		if err != nil {
			s.log.WithError(err).Error("[GRPC] Error selecting session data by id")
			return nil, status.Error(codes.Internal, "Internal error")
		}

		if session == nil {
			return nil, nil
		}

		details, err := cosmostypes.NewAnyWithValue(&types.KeygenSessionData{
			Parties: session.Parties,
			Key:     session.Key.String,
		})

		return &types.Session{
			Id:         uint64(session.ID),
			Status:     types.SessionStatus(session.Status),
			StartBlock: uint64(session.BeginBlock),
			EndBlock:   uint64(session.EndBlock),
			Type:       types.SessionType_ReshareSession,
			Data:       details,
		}, nil
	}

	return nil, nil
}
