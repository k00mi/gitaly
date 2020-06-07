package locator

import (
	"context"

	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"google.golang.org/grpc"
)

const (
	instanceKey = "locator.instance.key"
)

// UnaryInterceptor returns a Unary Interceptor with provided locator.
// It will be set into context with context tags.
// GetFromCtx should be used to extract it from the context.
func UnaryInterceptor(locator storage.Locator) func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		SetAtCtx(ctx, locator)
		return handler(ctx, req)
	}
}

// StreamInterceptor returns a Stream Interceptor
// It will be set into context with context tags.
// GetFromCtx should be used to extract it from the context.
func StreamInterceptor(locator storage.Locator) func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		SetAtCtx(stream.Context(), locator)
		return handler(srv, stream)
	}
}

// SetAtCtx sets provided locator to the context using context tags.
func SetAtCtx(ctx context.Context, locator storage.Locator) {
	tags := grpc_ctxtags.Extract(ctx)
	if tags.Has(instanceKey) {
		panic("locator is already set for this context")
	}

	if tags == grpc_ctxtags.NoopTags {
		panic("locator can be set without context tags support")
	}

	tags.Set(instanceKey, locator)
}

// IsAtCtx checks if locator is set at the context and it is not a nil.
func IsAtCtx(ctx context.Context) bool {
	tags := grpc_ctxtags.Extract(ctx)
	if !tags.Has(instanceKey) {
		return false
	}

	_, ok := tags.Values()[instanceKey].(storage.Locator)
	return ok
}

// GetFromCtx returns locator from the passed in context.
// If locator is not set this method will panic.
// Please ensure you have added github.com/grpc-ecosystem/go-grpc-middleware/tags middleware
// to support gRPC context based tags and a proper UnaryInterceptor | StreamInterceptor middleware
// defined in this package.
// If this error occurs in the test, you can use helper.CtxWithLocator function to inject
// locator into the context.
func GetFromCtx(ctx context.Context) storage.Locator {
	tags := grpc_ctxtags.Extract(ctx)
	if tags.Has(instanceKey) {
		locator, ok := tags.Values()[instanceKey].(storage.Locator)
		if locator != nil && ok {
			return locator
		}
	}

	panic("locator is not defined")
}
