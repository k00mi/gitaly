package featureflag

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// ContextWithFeatureFlag is used in tests to enablea a feature flag in the context metadata
func ContextWithFeatureFlag(ctx context.Context, flag string) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		md = metadata.New(map[string]string{HeaderKey(flag): "true"})
	} else {
		md.Set(HeaderKey(flag), "true")
	}

	return metadata.NewOutgoingContext(ctx, md)
}
