package metadata

import (
	"context"

	"google.golang.org/grpc/metadata"
)

// GetValue returns the first value in the metadata slice based on a key
func GetValue(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if values, ok := md[key]; ok && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
