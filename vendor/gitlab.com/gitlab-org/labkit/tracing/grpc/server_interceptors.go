package grpccorrelation

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"google.golang.org/grpc"
)

// UnaryServerTracingInterceptor will create a unary server tracing interceptor
func UnaryServerTracingInterceptor() grpc.UnaryServerInterceptor {
	return grpc_opentracing.UnaryServerInterceptor()
}

// StreamServerTracingInterceptor will create a streaming server tracing interceptor
func StreamServerTracingInterceptor() grpc.StreamServerInterceptor {
	return grpc_opentracing.StreamServerInterceptor()
}
