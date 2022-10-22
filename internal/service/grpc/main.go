package grpc

import (
	"context"

	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type ServerImpl struct {
	types.UnimplementedServiceServer
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
	//TODO implement me
	panic("implement me")
}
