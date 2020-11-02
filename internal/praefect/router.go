package praefect

import (
	"context"

	"google.golang.org/grpc"
)

// RouterNode represents a Node the router in a routing decision.
type RouterNode struct {
	// Storage is storage of the node.
	Storage string
	// Connection is the connection to the node.
	Connection *grpc.ClientConn
}

// StorageMutatorRoute describes how to route a storage scoped mutator call.
type StorageMutatorRoute struct {
	// Primary is the primary node of the routing decision.
	Primary RouterNode
	// Secondaries are the secondary nodes of the routing decision.
	Secondaries []RouterNode
}

// StorageMutatorRoute describes how to route a repository scoped mutator call.
type RepositoryMutatorRoute struct {
	// Primary is the primary node of the transaction.
	Primary RouterNode
	// Secondaries are the secondary participating in a transaction.
	Secondaries []RouterNode
	// ReplicationTargets are additional nodes that do not participate in a transaction
	// but need the changes replicated.
	ReplicationTargets []string
}

// Router decides which nodes to direct accessor and mutator RPCs to.
type Router interface {
	// RouteStorageAccessor returns the node which should serve the storage accessor request.
	RouteStorageAccessor(ctx context.Context, virtualStorage string) (RouterNode, error)
	// RouteStorageAccessor returns the primary and secondaries that should handle the storage
	// mutator request.
	RouteStorageMutator(ctx context.Context, virtualStorage string) (StorageMutatorRoute, error)
	// RouteRepositoryAccessor returns the node that should serve the repository accessor request.
	RouteRepositoryAccessor(ctx context.Context, virtualStorage, relativePath string) (RouterNode, error)
	// RouteRepositoryMutatorTransaction returns the primary and secondaries that should handle the repository mutator request.
	// Additionally, it returns nodes which should have the change replicated to.
	RouteRepositoryMutator(ctx context.Context, virtualStorage, relativePath string) (RepositoryMutatorRoute, error)
}
