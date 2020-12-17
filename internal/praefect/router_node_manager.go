package praefect

import (
	"context"
	"fmt"

	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/nodes"
)

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

// NewNodeManagerRouter returns a router that uses the NodeManager to make routing decisions.
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
	shard, err := r.mgr.GetShard(ctx, virtualStorage)
	if err != nil {
		return RouterNode{}, err
	}

	return toRouterNode(shard.Primary), nil
}

func (r *nodeManagerRouter) RouteStorageMutator(ctx context.Context, virtualStorage string) (StorageMutatorRoute, error) {
	shard, err := r.mgr.GetShard(ctx, virtualStorage)
	if err != nil {
		return StorageMutatorRoute{}, err
	}

	return StorageMutatorRoute{
		Primary:     toRouterNode(shard.Primary),
		Secondaries: toRouterNodes(shard.GetHealthySecondaries()),
	}, nil
}

func (r *nodeManagerRouter) RouteRepositoryMutator(ctx context.Context, virtualStorage, relativePath string) (RepositoryMutatorRoute, error) {
	shard, err := r.mgr.GetShard(ctx, virtualStorage)
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
