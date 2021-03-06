package nodes

import (
	"context"

	"google.golang.org/grpc"
)

// MockManager is a helper for tests that implements Manager and allows
// for parametrizing behavior.
type MockManager struct {
	Manager
	GetShardFunc func(string) (Shard, error)
}

func (m *MockManager) GetShard(_ context.Context, storage string) (Shard, error) {
	return m.GetShardFunc(storage)
}

// MockNode is a helper for tests that implements Node and allows
// for parametrizing behavior.
type MockNode struct {
	Node
	GetStorageMethod func() string
	Conn             *grpc.ClientConn
	Healthy          bool
}

func (m *MockNode) GetStorage() string { return m.GetStorageMethod() }

func (m *MockNode) IsHealthy() bool { return m.Healthy }

func (m *MockNode) GetConnection() *grpc.ClientConn { return m.Conn }

func (m *MockNode) GetAddress() string { return "" }

func (m *MockNode) GetToken() string { return "" }
