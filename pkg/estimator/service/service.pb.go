// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: service.proto

package service

import (
	context "context"
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	pb "github.com/karmada-io/karmada/pkg/estimator/pb"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

func init() { proto.RegisterFile("service.proto", fileDescriptor_a0b84a42fa06f626) }

var fileDescriptor_a0b84a42fa06f626 = []byte{
	// 249 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x92, 0x3f, 0x4a, 0x03, 0x41,
	0x14, 0x87, 0x5d, 0x10, 0x24, 0x0b, 0x36, 0x83, 0x85, 0xa6, 0x12, 0x0f, 0x30, 0x0b, 0x7a, 0x82,
	0x2c, 0x48, 0x0a, 0x0d, 0xc8, 0x82, 0x16, 0x36, 0x32, 0x3b, 0xfb, 0x63, 0xf2, 0xd8, 0x9d, 0x3f,
	0xce, 0xbc, 0x0d, 0x1e, 0x28, 0x37, 0xb0, 0xf2, 0x76, 0x16, 0x66, 0x02, 0x42, 0x9a, 0x25, 0xe9,
	0x86, 0xe1, 0xbd, 0xef, 0xfb, 0x8a, 0x57, 0x5e, 0x26, 0xc4, 0x0d, 0x69, 0xc8, 0x10, 0x3d, 0x7b,
	0xf1, 0x60, 0x88, 0xd7, 0x63, 0x2b, 0xb5, 0xb7, 0xb2, 0x57, 0xd1, 0xaa, 0x4e, 0x7d, 0x90, 0xcf,
	0x4f, 0x19, 0x7a, 0x23, 0x91, 0x98, 0xac, 0x62, 0x1f, 0xe5, 0x6e, 0x75, 0x7e, 0x1b, 0x7a, 0x53,
	0xed, 0xbf, 0xab, 0xd0, 0x56, 0x06, 0x0e, 0x51, 0x31, 0xba, 0x3f, 0xec, 0xfd, 0xf6, 0xbc, 0x9c,
	0x3d, 0xe6, 0x01, 0xf1, 0x5d, 0x94, 0x57, 0x2b, 0xf5, 0xb5, 0xd8, 0x28, 0x1a, 0x54, 0x3b, 0xa0,
	0x41, 0x18, 0x48, 0xab, 0x24, 0x9e, 0xe4, 0x14, 0x7d, 0x68, 0xe5, 0x21, 0x4a, 0x83, 0xcf, 0x11,
	0x89, 0xe7, 0xcf, 0xa7, 0x81, 0xa5, 0xe0, 0x5d, 0xc2, 0xdd, 0x99, 0xf8, 0x29, 0xca, 0xeb, 0x25,
	0xf8, 0xd5, 0x25, 0xbd, 0x46, 0x37, 0xfe, 0x2f, 0x9f, 0x2c, 0x3b, 0x88, 0xc9, 0xe9, 0xab, 0x13,
	0xd1, 0xf6, 0xed, 0xdb, 0xa2, 0xbc, 0x59, 0x82, 0xdf, 0x48, 0x33, 0xd9, 0x06, 0xc9, 0x8f, 0x51,
	0xa3, 0x26, 0xd7, 0x91, 0x33, 0x49, 0x2c, 0xa6, 0xea, 0x5e, 0x22, 0x60, 0x03, 0x93, 0x77, 0xb9,
	0xb8, 0x3e, 0x06, 0x91, 0x33, 0xeb, 0xd9, 0xfb, 0xc5, 0xee, 0xa6, 0x7e, 0x03, 0x00, 0x00, 0xff,
	0xff, 0xc7, 0xc7, 0xe2, 0x27, 0x98, 0x02, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// EstimatorClient is the client API for Estimator service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type EstimatorClient interface {
	MaxAvailableReplicas(ctx context.Context, in *pb.MaxAvailableReplicasRequest, opts ...grpc.CallOption) (*pb.MaxAvailableReplicasResponse, error)
	GetUnschedulableReplicas(ctx context.Context, in *pb.UnschedulableReplicasRequest, opts ...grpc.CallOption) (*pb.UnschedulableReplicasResponse, error)
	GetVictimResourceBindings(ctx context.Context, in *pb.PreemptionRequest, opts ...grpc.CallOption) (*pb.PreemptionResponse, error)
}

type estimatorClient struct {
	cc *grpc.ClientConn
}

func NewEstimatorClient(cc *grpc.ClientConn) EstimatorClient {
	return &estimatorClient{cc}
}

func (c *estimatorClient) MaxAvailableReplicas(ctx context.Context, in *pb.MaxAvailableReplicasRequest, opts ...grpc.CallOption) (*pb.MaxAvailableReplicasResponse, error) {
	out := new(pb.MaxAvailableReplicasResponse)
	err := c.cc.Invoke(ctx, "/github.com.karmada_io.karmada.pkg.estimator.service.Estimator/MaxAvailableReplicas", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *estimatorClient) GetUnschedulableReplicas(ctx context.Context, in *pb.UnschedulableReplicasRequest, opts ...grpc.CallOption) (*pb.UnschedulableReplicasResponse, error) {
	out := new(pb.UnschedulableReplicasResponse)
	err := c.cc.Invoke(ctx, "/github.com.karmada_io.karmada.pkg.estimator.service.Estimator/GetUnschedulableReplicas", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *estimatorClient) GetVictimResourceBindings(ctx context.Context, in *pb.PreemptionRequest, opts ...grpc.CallOption) (*pb.PreemptionResponse, error) {
	out := new(pb.PreemptionResponse)
	err := c.cc.Invoke(ctx, "/github.com.karmada_io.karmada.pkg.estimator.service.Estimator/GetVictimResourceBindings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EstimatorServer is the server API for Estimator service.
type EstimatorServer interface {
	MaxAvailableReplicas(context.Context, *pb.MaxAvailableReplicasRequest) (*pb.MaxAvailableReplicasResponse, error)
	GetUnschedulableReplicas(context.Context, *pb.UnschedulableReplicasRequest) (*pb.UnschedulableReplicasResponse, error)
	GetVictimResourceBindings(context.Context, *pb.PreemptionRequest) (*pb.PreemptionResponse, error)
}

// UnimplementedEstimatorServer can be embedded to have forward compatible implementations.
type UnimplementedEstimatorServer struct {
}

func (*UnimplementedEstimatorServer) MaxAvailableReplicas(ctx context.Context, req *pb.MaxAvailableReplicasRequest) (*pb.MaxAvailableReplicasResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MaxAvailableReplicas not implemented")
}
func (*UnimplementedEstimatorServer) GetUnschedulableReplicas(ctx context.Context, req *pb.UnschedulableReplicasRequest) (*pb.UnschedulableReplicasResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetUnschedulableReplicas not implemented")
}
func (*UnimplementedEstimatorServer) GetVictimResourceBindings(ctx context.Context, req *pb.PreemptionRequest) (*pb.PreemptionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetVictimResourceBindings not implemented")
}

func RegisterEstimatorServer(s *grpc.Server, srv EstimatorServer) {
	s.RegisterService(&_Estimator_serviceDesc, srv)
}

func _Estimator_MaxAvailableReplicas_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(pb.MaxAvailableReplicasRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EstimatorServer).MaxAvailableReplicas(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/github.com.karmada_io.karmada.pkg.estimator.service.Estimator/MaxAvailableReplicas",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EstimatorServer).MaxAvailableReplicas(ctx, req.(*pb.MaxAvailableReplicasRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Estimator_GetUnschedulableReplicas_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(pb.UnschedulableReplicasRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EstimatorServer).GetUnschedulableReplicas(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/github.com.karmada_io.karmada.pkg.estimator.service.Estimator/GetUnschedulableReplicas",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EstimatorServer).GetUnschedulableReplicas(ctx, req.(*pb.UnschedulableReplicasRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Estimator_GetVictimResourceBindings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(pb.PreemptionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EstimatorServer).GetVictimResourceBindings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/github.com.karmada_io.karmada.pkg.estimator.service.Estimator/GetVictimResourceBindings",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EstimatorServer).GetVictimResourceBindings(ctx, req.(*pb.PreemptionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _Estimator_serviceDesc = grpc.ServiceDesc{
	ServiceName: "github.com.karmada_io.karmada.pkg.estimator.service.Estimator",
	HandlerType: (*EstimatorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "MaxAvailableReplicas",
			Handler:    _Estimator_MaxAvailableReplicas_Handler,
		},
		{
			MethodName: "GetUnschedulableReplicas",
			Handler:    _Estimator_GetUnschedulableReplicas_Handler,
		},
		{
			MethodName: "GetVictimResourceBindings",
			Handler:    _Estimator_GetVictimResourceBindings_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "service.proto",
}
