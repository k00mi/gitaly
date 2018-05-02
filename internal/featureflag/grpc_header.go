package featureflag

import (
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

// IsEnabled checks if the feature flag is enabled for the passed context.
// Only return true if the metadata for the feature flag is set to "true"
func IsEnabled(ctx context.Context, flag string) bool {
	if flag == "" {
		return false
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}

	headerKey := fmt.Sprintf("gitaly-feature-%s", flag)
	val, ok := md[headerKey]
	if !ok {
		return false
	}

	return len(val) > 0 && val[0] == "true"
}

// IsDisabled is the inverse operation of IsEnabled
func IsDisabled(ctx context.Context, flag string) bool {
	return !IsEnabled(ctx, flag)
}
