package grpccorrelation

import (
	"github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	"google.golang.org/grpc"
)

// UnaryClientTracingInterceptor will create a unary client tracing interceptor
func UnaryClientTracingInterceptor() grpc.UnaryClientInterceptor {
	return grpc_opentracing.UnaryClientInterceptor()
}

// StreamClientTracingInterceptor will create a streaming client tracing interceptor
func StreamClientTracingInterceptor() grpc.StreamClientInterceptor {
	return grpc_opentracing.StreamClientInterceptor()
}
