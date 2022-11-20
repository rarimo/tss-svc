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
	receiver core.IGlobalReceiver
	log      *logan.Entry
	listener net.Listener
}

func NewServer(cfg config.Config) *ServerImpl {
	return &ServerImpl{
		receiver: core.NewManager(nil),
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

func (s *ServerImpl) Submit(ctx context.Context, request *types.MsgSubmitRequest) (*types.MsgSubmitResponse, error) {
	return &types.MsgSubmitResponse{}, s.receiver.Receive(request)
}
