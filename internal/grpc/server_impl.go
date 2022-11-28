package grpc

import (
	"context"
	"net"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ServerImpl struct {
	types.UnimplementedServiceServer
	manager  *core.SessionManager
	log      *logan.Entry
	listener net.Listener
	storage  *pg.Storage
	pool     *pool.Pool
}

func NewServer(manager *core.SessionManager, cfg config.Config) *ServerImpl {
	return &ServerImpl{
		manager:  manager,
		log:      cfg.Log(),
		listener: cfg.Listener(),
		storage:  cfg.Storage(),
		pool:     pool.NewPool(cfg),
	}
}

func (s *ServerImpl) Run() error {
	grpcServer := grpc.NewServer()
	types.RegisterServiceServer(grpcServer, s)
	return grpcServer.Serve(s.listener)
}

var _ types.ServiceServer = &ServerImpl{}

func (s *ServerImpl) Submit(ctx context.Context, request *types.MsgSubmitRequest) (*types.MsgSubmitResponse, error) {
	return &types.MsgSubmitResponse{}, s.manager.Receive(request)
}

func (s *ServerImpl) AddOperation(ctx context.Context, request *types.MsgAddOperationRequest) (*types.MsgAddOperationResponse, error) {
	err := s.pool.Add(request.Index)
	if err != nil {
		s.log.WithError(err).Error("error adding to the pool")
		return nil, status.Errorf(codes.InvalidArgument, "Invalid request index: maybe already signed")
	}
	return &types.MsgAddOperationResponse{}, nil
}
