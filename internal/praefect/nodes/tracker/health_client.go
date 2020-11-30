package tracker

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// HealthClient wraps a gRPC HealthClient and circuit breaks its health checks if
// the error threshold has been reached.
type HealthClient struct {
	storage string
	tracker ErrorTracker
	grpc_health_v1.HealthClient
}

// NewHealthClient returns the HealthClient wrapped with error threshold circuit breaker.
func NewHealthClient(client grpc_health_v1.HealthClient, storage string, tracker ErrorTracker) HealthClient {
	return HealthClient{
		tracker:      tracker,
		HealthClient: client,
		storage:      storage,
	}
}

// Check circuit breaks the health check if write or read error thresholds have been reached. If not, it performs
// the health check.
func (hc HealthClient) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest, opts ...grpc.CallOption) (*grpc_health_v1.HealthCheckResponse, error) {
	if hc.tracker.ReadThresholdReached(hc.storage) {
		return nil, fmt.Errorf("read error threshold reached for storage %q", hc.storage)
	}

	if hc.tracker.WriteThresholdReached(hc.storage) {
		return nil, fmt.Errorf("write error threshold reached for storage %q", hc.storage)
	}

	return hc.HealthClient.Check(ctx, req, opts...)
}
