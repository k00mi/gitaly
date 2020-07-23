package datastore

import "context"

// MockReplicationEventQueue is a helper for tests that implements ReplicationEventQueue
// and allows for parametrizing behavior.
type MockReplicationEventQueue struct {
	ReplicationEventQueue
	GetOutdatedRepositoriesFunc func(context.Context, string, string) (map[string][]string, error)
	EnqueueFunc                 func(context.Context, ReplicationEvent) (ReplicationEvent, error)
}

func (m *MockReplicationEventQueue) GetOutdatedRepositories(ctx context.Context, virtualStorage string, referenceStorage string) (map[string][]string, error) {
	return m.GetOutdatedRepositoriesFunc(ctx, virtualStorage, referenceStorage)
}

func (m *MockReplicationEventQueue) Enqueue(ctx context.Context, event ReplicationEvent) (ReplicationEvent, error) {
	return m.EnqueueFunc(ctx, event)
}
