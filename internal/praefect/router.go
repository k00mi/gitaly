package praefect

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/nodes"
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

type nodeManagerRouter struct {
	mgr nodes.Manager
	rs  datastore.RepositoryStore
}

func toRouterNode(node nodes.Node) RouterNode {
	return RouterNode{
		Storage:    node.GetStorage(),
		Connection: node.GetConnection(),
	}
}

func toRouterNodes(nodes []nodes.Node) []RouterNode {
	out := make([]RouterNode, len(nodes))
	for i := range nodes {
		out[i] = toRouterNode(nodes[i])
	}
	return out
}

// NeWNodeManagerRouter returns a router that uses the NodeManager to make routing decisions.
func NewNodeManagerRouter(mgr nodes.Manager, rs datastore.RepositoryStore) Router {
	return &nodeManagerRouter{mgr: mgr, rs: rs}
}

func (r *nodeManagerRouter) RouteRepositoryAccessor(ctx context.Context, virtualStorage, relativePath string) (RouterNode, error) {
	node, err := r.mgr.GetSyncedNode(ctx, virtualStorage, relativePath)
	if err != nil {
		return RouterNode{}, fmt.Errorf("get synced node: %w", err)
	}

	return toRouterNode(node), nil
}

func (r *nodeManagerRouter) RouteStorageAccessor(ctx context.Context, virtualStorage string) (RouterNode, error) {
	shard, err := r.mgr.GetShard(virtualStorage)
	if err != nil {
		return RouterNode{}, err
	}

	return toRouterNode(shard.Primary), nil
}

func (r *nodeManagerRouter) RouteStorageMutator(ctx context.Context, virtualStorage string) (StorageMutatorRoute, error) {
	shard, err := r.mgr.GetShard(virtualStorage)
	if err != nil {
		return StorageMutatorRoute{}, err
	}

	return StorageMutatorRoute{
		Primary:     toRouterNode(shard.Primary),
		Secondaries: toRouterNodes(shard.GetHealthySecondaries()),
	}, nil
}

func (r *nodeManagerRouter) RouteRepositoryMutator(ctx context.Context, virtualStorage, relativePath string) (RepositoryMutatorRoute, error) {
	shard, err := r.mgr.GetShard(virtualStorage)
	if err != nil {
		return RepositoryMutatorRoute{}, fmt.Errorf("get shard: %w", err)
	}

	if latest, err := r.rs.IsLatestGeneration(ctx, virtualStorage, relativePath, shard.Primary.GetStorage()); err != nil {
		return RepositoryMutatorRoute{}, fmt.Errorf("check generation: %w", err)
	} else if !latest {
		return RepositoryMutatorRoute{}, ErrRepositoryReadOnly
	}

	// Only healthy secondaries which are consistent with the primary are allowed to take
	// part in the transaction. Unhealthy nodes would block the transaction until they come back.
	// Inconsistent nodes will anyway need repair so including them doesn't make sense. They
	// also might vote to abort which might unnecessarily fail the transaction.
	consistentSecondaries, err := r.rs.GetConsistentSecondaries(ctx, virtualStorage, relativePath, shard.Primary.GetStorage())
	if err != nil {
		return RepositoryMutatorRoute{}, fmt.Errorf("consistent secondaries: %w", err)
	}

	var replicationTargets []string
	participatingSecondaries := make([]nodes.Node, 0, len(consistentSecondaries))
	for _, secondary := range shard.Secondaries {
		if _, ok := consistentSecondaries[secondary.GetStorage()]; ok && secondary.IsHealthy() {
			participatingSecondaries = append(participatingSecondaries, secondary)
			continue
		}

		replicationTargets = append(replicationTargets, secondary.GetStorage())
	}

	return RepositoryMutatorRoute{
		Primary:            toRouterNode(shard.Primary),
		Secondaries:        toRouterNodes(participatingSecondaries),
		ReplicationTargets: replicationTargets,
	}, nil
}
