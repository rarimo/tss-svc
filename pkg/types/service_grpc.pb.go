// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             (unknown)
// source: service.proto

package types

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ServiceClient is the client API for Service service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ServiceClient interface {
	Submit(ctx context.Context, in *MsgSubmitRequest, opts ...grpc.CallOption) (*MsgSubmitResponse, error)
	AddOperation(ctx context.Context, in *MsgAddOperationRequest, opts ...grpc.CallOption) (*MsgAddOperationResponse, error)
	Info(ctx context.Context, in *MsgInfoRequest, opts ...grpc.CallOption) (*MsgInfoResponse, error)
	Session(ctx context.Context, in *MsgSessionRequest, opts ...grpc.CallOption) (*MsgSessionResponse, error)
}

type serviceClient struct {
	cc grpc.ClientConnInterface
}

func NewServiceClient(cc grpc.ClientConnInterface) ServiceClient {
	return &serviceClient{cc}
}

func (c *serviceClient) Submit(ctx context.Context, in *MsgSubmitRequest, opts ...grpc.CallOption) (*MsgSubmitResponse, error) {
	out := new(MsgSubmitResponse)
	err := c.cc.Invoke(ctx, "/Service/Submit", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) AddOperation(ctx context.Context, in *MsgAddOperationRequest, opts ...grpc.CallOption) (*MsgAddOperationResponse, error) {
	out := new(MsgAddOperationResponse)
	err := c.cc.Invoke(ctx, "/Service/AddOperation", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) Info(ctx context.Context, in *MsgInfoRequest, opts ...grpc.CallOption) (*MsgInfoResponse, error) {
	out := new(MsgInfoResponse)
	err := c.cc.Invoke(ctx, "/Service/Info", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceClient) Session(ctx context.Context, in *MsgSessionRequest, opts ...grpc.CallOption) (*MsgSessionResponse, error) {
	out := new(MsgSessionResponse)
	err := c.cc.Invoke(ctx, "/Service/Session", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ServiceServer is the server API for Service service.
// All implementations should embed UnimplementedServiceServer
// for forward compatibility
type ServiceServer interface {
	Submit(context.Context, *MsgSubmitRequest) (*MsgSubmitResponse, error)
	AddOperation(context.Context, *MsgAddOperationRequest) (*MsgAddOperationResponse, error)
	Info(context.Context, *MsgInfoRequest) (*MsgInfoResponse, error)
	Session(context.Context, *MsgSessionRequest) (*MsgSessionResponse, error)
}

// UnimplementedServiceServer should be embedded to have forward compatible implementations.
type UnimplementedServiceServer struct {
}

func (UnimplementedServiceServer) Submit(context.Context, *MsgSubmitRequest) (*MsgSubmitResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Submit not implemented")
}
func (UnimplementedServiceServer) AddOperation(context.Context, *MsgAddOperationRequest) (*MsgAddOperationResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddOperation not implemented")
}
func (UnimplementedServiceServer) Info(context.Context, *MsgInfoRequest) (*MsgInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Info not implemented")
}
func (UnimplementedServiceServer) Session(context.Context, *MsgSessionRequest) (*MsgSessionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Session not implemented")
}

// UnsafeServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ServiceServer will
// result in compilation errors.
type UnsafeServiceServer interface {
	mustEmbedUnimplementedServiceServer()
}

func RegisterServiceServer(s grpc.ServiceRegistrar, srv ServiceServer) {
	s.RegisterService(&Service_ServiceDesc, srv)
}

func _Service_Submit_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSubmitRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).Submit(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/Submit",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).Submit(ctx, req.(*MsgSubmitRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_AddOperation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgAddOperationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).AddOperation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/AddOperation",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).AddOperation(ctx, req.(*MsgAddOperationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_Info_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).Info(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/Info",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).Info(ctx, req.(*MsgInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Service_Session_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgSessionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceServer).Session(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Service/Session",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceServer).Session(ctx, req.(*MsgSessionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Service_ServiceDesc is the grpc.ServiceDesc for Service service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Service_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "Service",
	HandlerType: (*ServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Submit",
			Handler:    _Service_Submit_Handler,
		},
		{
			MethodName: "AddOperation",
			Handler:    _Service_AddOperation_Handler,
		},
		{
			MethodName: "Info",
			Handler:    _Service_Info_Handler,
		},
		{
			MethodName: "Session",
			Handler:    _Service_Session_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "service.proto",
}