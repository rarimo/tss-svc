package grpc

import (
	"context"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/ignite/cli/ignite/pkg/openapiconsole"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/docs"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"gitlab.com/rarimo/tss/tss-svc/internal/pool"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

type ServerImpl struct {
	types.UnimplementedServiceServer
	manager  *core.SessionManager
	log      *logan.Entry
	listener net.Listener
	pg       *pg.Storage
	storage  secret.Storage
	pool     *pool.Pool
	swagger  *config.SwaggerInfo
}

func NewServer(ctx core.Context, manager *core.SessionManager) *ServerImpl {
	return &ServerImpl{
		manager:  manager,
		log:      ctx.Log(),
		listener: ctx.Listener(),
		pg:       ctx.PG(),
		storage:  ctx.SecretStorage(),
		pool:     ctx.Pool(),
		swagger:  ctx.Swagger(),
	}
}

func (s *ServerImpl) RunGRPC() error {
	grpcServer := grpc.NewServer()
	types.RegisterServiceServer(grpcServer, s)
	return grpcServer.Serve(s.listener)
}

func (s *ServerImpl) RunGateway() error {
	if !s.swagger.Enabled {
		return nil
	}

	grpcGatewayRouter := runtime.NewServeMux()
	httpRouter := http.NewServeMux()

	err := types.RegisterServiceHandlerServer(context.Background(), grpcGatewayRouter, s)
	if err != nil {
		panic(err)
	}

	httpRouter.Handle("/static/service.swagger.json", http.FileServer(http.FS(docs.Docs)))
	httpRouter.HandleFunc("/api", openapiconsole.Handler("TSS service", "/static/service.swagger.json"))
	httpRouter.Handle("/", grpcGatewayRouter)
	return http.ListenAndServe(s.swagger.Addr, httpRouter)
}

var _ types.ServiceServer = &ServerImpl{}

func (s *ServerImpl) Submit(ctx context.Context, request *types.MsgSubmitRequest) (*types.MsgSubmitResponse, error) {
	if err := s.manager.Receive(ctx, request); err != nil {
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

		details, err := anypb.New(&types.DefaultSessionData{
			Parties:   session.Parties,
			Proposer:  session.Proposer.String,
			Indexes:   session.Indexes,
			Root:      session.Root.String,
			Accepted:  session.Accepted,
			Signature: session.Signature.String,
		})

		if err != nil {
			return nil, status.Error(codes.Internal, "Failed to marshal data")
		}

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

		details, err := anypb.New(&types.ReshareSessionData{
			Parties:      session.Parties,
			Proposer:     session.Proposer.String,
			OldKey:       session.OldKey.String,
			NewKey:       session.NewKey.String,
			Root:         session.Root.String,
			KeySignature: session.KeySignature.String,
			Signature:    session.Signature.String,
		})

		if err != nil {
			return nil, status.Error(codes.Internal, "Failed to marshal data")
		}

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

		details, err := anypb.New(&types.KeygenSessionData{
			Parties: session.Parties,
			Key:     session.Key.String,
		})

		if err != nil {
			return nil, status.Error(codes.Internal, "Failed to marshal data")
		}

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
