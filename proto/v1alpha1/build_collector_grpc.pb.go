// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package v1alpha1

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

// BuildCollectorClient is the client API for BuildCollector service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type BuildCollectorClient interface {
	CreateBuild(ctx context.Context, in *CreateBuildRequest, opts ...grpc.CallOption) (*CreateBuildResponse, error)
	UpdateBuildArtifacts(ctx context.Context, in *UpdateBuildArtifactsRequest, opts ...grpc.CallOption) (*UpdateBuildArtifactsResponse, error)
}

type buildCollectorClient struct {
	cc grpc.ClientConnInterface
}

func NewBuildCollectorClient(cc grpc.ClientConnInterface) BuildCollectorClient {
	return &buildCollectorClient{cc}
}

func (c *buildCollectorClient) CreateBuild(ctx context.Context, in *CreateBuildRequest, opts ...grpc.CallOption) (*CreateBuildResponse, error) {
	out := new(CreateBuildResponse)
	err := c.cc.Invoke(ctx, "/build_collector.v1alpha1.BuildCollector/CreateBuild", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *buildCollectorClient) UpdateBuildArtifacts(ctx context.Context, in *UpdateBuildArtifactsRequest, opts ...grpc.CallOption) (*UpdateBuildArtifactsResponse, error) {
	out := new(UpdateBuildArtifactsResponse)
	err := c.cc.Invoke(ctx, "/build_collector.v1alpha1.BuildCollector/UpdateBuildArtifacts", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BuildCollectorServer is the server API for BuildCollector service.
// All implementations should embed UnimplementedBuildCollectorServer
// for forward compatibility
type BuildCollectorServer interface {
	CreateBuild(context.Context, *CreateBuildRequest) (*CreateBuildResponse, error)
	UpdateBuildArtifacts(context.Context, *UpdateBuildArtifactsRequest) (*UpdateBuildArtifactsResponse, error)
}

// UnimplementedBuildCollectorServer should be embedded to have forward compatible implementations.
type UnimplementedBuildCollectorServer struct {
}

func (UnimplementedBuildCollectorServer) CreateBuild(context.Context, *CreateBuildRequest) (*CreateBuildResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateBuild not implemented")
}
func (UnimplementedBuildCollectorServer) UpdateBuildArtifacts(context.Context, *UpdateBuildArtifactsRequest) (*UpdateBuildArtifactsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateBuildArtifacts not implemented")
}

// UnsafeBuildCollectorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to BuildCollectorServer will
// result in compilation errors.
type UnsafeBuildCollectorServer interface {
	mustEmbedUnimplementedBuildCollectorServer()
}

func RegisterBuildCollectorServer(s grpc.ServiceRegistrar, srv BuildCollectorServer) {
	s.RegisterService(&BuildCollector_ServiceDesc, srv)
}

func _BuildCollector_CreateBuild_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateBuildRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BuildCollectorServer).CreateBuild(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/build_collector.v1alpha1.BuildCollector/CreateBuild",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BuildCollectorServer).CreateBuild(ctx, req.(*CreateBuildRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BuildCollector_UpdateBuildArtifacts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateBuildArtifactsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BuildCollectorServer).UpdateBuildArtifacts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/build_collector.v1alpha1.BuildCollector/UpdateBuildArtifacts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BuildCollectorServer).UpdateBuildArtifacts(ctx, req.(*UpdateBuildArtifactsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// BuildCollector_ServiceDesc is the grpc.ServiceDesc for BuildCollector service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var BuildCollector_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "build_collector.v1alpha1.BuildCollector",
	HandlerType: (*BuildCollectorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateBuild",
			Handler:    _BuildCollector_CreateBuild_Handler,
		},
		{
			MethodName: "UpdateBuildArtifacts",
			Handler:    _BuildCollector_UpdateBuildArtifacts_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/v1alpha1/build_collector.proto",
}
