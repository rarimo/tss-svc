package grpc

import (
	"context"
	"net"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

type ServerImpl struct {
	types.UnimplementedServiceServer
	core     core.IGlobalReceiver
	log      *logan.Entry
	listener net.Listener
}

func NewServer(receiver core.IGlobalReceiver, cfg config.Config) *ServerImpl {
	return &ServerImpl{
		core:     receiver,
		log:      cfg.Log(),
		listener: cfg.Listener(),
	}
}

func (s *ServerImpl) Run() error {
	grpcServer := grpc.NewServer()
	types.RegisterServiceServer(grpcServer, s)
	return grpcServer.Serve(s.listener)
}

var _ types.ServiceServer = &ServerImpl{}

func (s *ServerImpl) Info(ctx context.Context, request *types.MsgInfoRequest) (*types.MsgInfoResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s *ServerImpl) AllSessionsInfo(ctx context.Context, request *types.MsgAllSessionInfoRequest) (*types.MsgAllSessionInfoResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s *ServerImpl) SessionInfo(ctx context.Context, request *types.MsgSessionInfoRequest) (*types.MsgSessionInfoResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s *ServerImpl) Submit(ctx context.Context, request *types.MsgSubmitRequest) (*types.MsgSubmitResponse, error) {
	return &types.MsgSubmitResponse{}, s.core.Receive(request)
}
