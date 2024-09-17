// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.27.3
// source: redis_cache-service.proto

package redis_cache_service

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	CardService_RedisGetCard_FullMethodName = "/redis_cache_service.proto.CardService/RedisGetCard"
)

// CardServiceClient is the client API for CardService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type CardServiceClient interface {
	RedisGetCard(ctx context.Context, in *RedisGetCardRequest, opts ...grpc.CallOption) (*RedisGetCardResponse, error)
}

type cardServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewCardServiceClient(cc grpc.ClientConnInterface) CardServiceClient {
	return &cardServiceClient{cc}
}

func (c *cardServiceClient) RedisGetCard(ctx context.Context, in *RedisGetCardRequest, opts ...grpc.CallOption) (*RedisGetCardResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(RedisGetCardResponse)
	err := c.cc.Invoke(ctx, CardService_RedisGetCard_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// CardServiceServer is the server API for CardService service.
// All implementations must embed UnimplementedCardServiceServer
// for forward compatibility.
type CardServiceServer interface {
	RedisGetCard(context.Context, *RedisGetCardRequest) (*RedisGetCardResponse, error)
	mustEmbedUnimplementedCardServiceServer()
}

// UnimplementedCardServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedCardServiceServer struct{}

func (UnimplementedCardServiceServer) RedisGetCard(context.Context, *RedisGetCardRequest) (*RedisGetCardResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RedisGetCard not implemented")
}
func (UnimplementedCardServiceServer) mustEmbedUnimplementedCardServiceServer() {}
func (UnimplementedCardServiceServer) testEmbeddedByValue()                     {}

// UnsafeCardServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to CardServiceServer will
// result in compilation errors.
type UnsafeCardServiceServer interface {
	mustEmbedUnimplementedCardServiceServer()
}

func RegisterCardServiceServer(s grpc.ServiceRegistrar, srv CardServiceServer) {
	// If the following call pancis, it indicates UnimplementedCardServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&CardService_ServiceDesc, srv)
}

func _CardService_RedisGetCard_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RedisGetCardRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CardServiceServer).RedisGetCard(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: CardService_RedisGetCard_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CardServiceServer).RedisGetCard(ctx, req.(*RedisGetCardRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// CardService_ServiceDesc is the grpc.ServiceDesc for CardService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var CardService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "redis_cache_service.proto.CardService",
	HandlerType: (*CardServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "RedisGetCard",
			Handler:    _CardService_RedisGetCard_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "redis_cache-service.proto",
}
