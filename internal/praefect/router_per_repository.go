package praefect

import (
	"context"
	"errors"
	"fmt"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/nodes"
	"google.golang.org/grpc"
)

// errRepositoryNotFound is retuned when trying to operate on a non-existent repository.
var errRepositoryNotFound = errors.New("repository not found")

// errPrimaryUnassigned is returned when the primary node is not in the set of assigned nodes.
var errPrimaryUnassigned = errors.New("primary node is not assigned")

// AssignmentGetter is an interface for getting repository host node assignments.
type AssignmentGetter interface {
	// GetHostAssignments returns the names of the storages assigned to host the repository.
	// The primary node must always be assigned.
	GetHostAssignments(ctx context.Context, virtualStorage, relativePath string) ([]string, error)
}

// StaticStorageAssignments is a static assignment of the same storages in a virtual storage for every repository.
type StaticStorageAssignments map[string][]string

func (st StaticStorageAssignments) GetHostAssignments(ctx context.Context, virtualStorage, relativePath string) ([]string, error) {
	storages, ok := st[virtualStorage]
	if !ok {
		return nil, nodes.ErrVirtualStorageNotExist
	}

	return storages, nil
}

// ErrNoSuitableNode is returned when there is not suitable node to serve a request.
var ErrNoSuitableNode = errors.New("no suitable node to serve the request")

// ErrNoHealthyNodes is returned when there are no healthy nodes to serve a request.
var ErrNoHealthyNodes = errors.New("no healthy nodes")

// Connections is a set of connections to configured storage nodes by their virtual storages.
type Connections map[string]map[string]*grpc.ClientConn

// PrimaryGetter is an interface for getting a primary of a repository.
type PrimaryGetter interface {
	// GetPrimary returns the primary storage for a given repository.
	GetPrimary(ctx context.Context, virtualStorage string, relativePath string) (string, error)
}

// PerRepositoryRouter implements a router that routes requests respecting per repository primary nodes.
type PerRepositoryRouter struct {
	conns Connections
	ag    AssignmentGetter
	pg    PrimaryGetter
	rand  Random
	hc    HealthChecker
	rs    datastore.RepositoryStore
}

// NewPerRepositoryRouter returns a new PerRepositoryRouter using the passed configuration.
func NewPerRepositoryRouter(conns Connections, pg PrimaryGetter, hc HealthChecker, rand Random, rs datastore.RepositoryStore, ag AssignmentGetter) *PerRepositoryRouter {
	return &PerRepositoryRouter{
		conns: conns,
		pg:    pg,
		rand:  rand,
		hc:    hc,
		rs:    rs,
		ag:    ag,
	}
}

func (r *PerRepositoryRouter) healthyNodes(virtualStorage string) ([]RouterNode, error) {
	conns, ok := r.conns[virtualStorage]
	if !ok {
		return nil, nodes.ErrVirtualStorageNotExist
	}

	healthyNodes := make([]RouterNode, 0, len(conns))
	for _, storage := range r.hc.HealthyNodes()[virtualStorage] {
		conn, ok := conns[storage]
		if !ok {
			return nil, fmt.Errorf("no connection to node %q/%q", virtualStorage, storage)
		}

		healthyNodes = append(healthyNodes, RouterNode{
			Storage:    storage,
			Connection: conn,
		})
	}

	if len(healthyNodes) == 0 {
		return nil, ErrNoHealthyNodes
	}

	return healthyNodes, nil
}

func (r *PerRepositoryRouter) pickRandom(nodes []RouterNode) (RouterNode, error) {
	if len(nodes) == 0 {
		return RouterNode{}, ErrNoSuitableNode
	}

	return nodes[r.rand.Intn(len(nodes))], nil
}

// RouteStorageAccessor routes requests for storage-scoped accessor RPCs. The
// only storage scoped accessor RPC is RemoteService/FindRemoteRepository,
// which in turn executes a command without a repository. This can be done by
// any Gitaly server as it doesn't depend on the state on the server.
func (r *PerRepositoryRouter) RouteStorageAccessor(ctx context.Context, virtualStorage string) (RouterNode, error) {
	healthyNodes, err := r.healthyNodes(virtualStorage)
	if err != nil {
		return RouterNode{}, err
	}

	return r.pickRandom(healthyNodes)
}

// RouteStorageMutator is not implemented here. The only storage scoped mutator RPC is related to namespace operations.
// These are not relevant anymore, given hashed storage is default everywhere, and should be eventually removed.
func (r *PerRepositoryRouter) RouteStorageMutator(ctx context.Context, virtualStorage string) (StorageMutatorRoute, error) {
	return StorageMutatorRoute{}, errors.New("RouteStorageMutator is not implemented on PerRepositoryRouter")
}

func (r *PerRepositoryRouter) RouteRepositoryAccessor(ctx context.Context, virtualStorage, relativePath string) (RouterNode, error) {
	healthyNodes, err := r.healthyNodes(virtualStorage)
	if err != nil {
		return RouterNode{}, err
	}

	primary, err := r.pg.GetPrimary(ctx, virtualStorage, relativePath)
	if err != nil {
		return RouterNode{}, fmt.Errorf("get primary: %w", err)
	}

	consistentSecondaries, err := r.rs.GetConsistentSecondaries(ctx, virtualStorage, relativePath, primary)
	if err != nil {
		// this is recoverable error - proceed with primary node
		ctxlogrus.Extract(ctx).WithError(err).Warn("get up to date secondaries")
	}

	consistentSecondaries[primary] = struct{}{}

	healthyConsistentNodes := make([]RouterNode, 0, len(healthyNodes))
	for _, node := range healthyNodes {
		if _, ok := consistentSecondaries[node.Storage]; !ok {
			continue
		}

		healthyConsistentNodes = append(healthyConsistentNodes, node)
	}

	return r.pickRandom(healthyConsistentNodes)
}

func (r *PerRepositoryRouter) RouteRepositoryMutator(ctx context.Context, virtualStorage, relativePath string) (RepositoryMutatorRoute, error) {
	healthyNodes, err := r.healthyNodes(virtualStorage)
	if err != nil {
		return RepositoryMutatorRoute{}, err
	}

	primary, err := r.pg.GetPrimary(ctx, virtualStorage, relativePath)
	if err != nil {
		return RepositoryMutatorRoute{}, fmt.Errorf("get primary: %w", err)
	}

	healthySet := make(map[string]RouterNode)
	for _, node := range healthyNodes {
		healthySet[node.Storage] = node
	}

	if _, ok := healthySet[primary]; !ok {
		return RepositoryMutatorRoute{}, nodes.ErrPrimaryNotHealthy
	}

	if latest, err := r.rs.IsLatestGeneration(ctx, virtualStorage, relativePath, primary); err != nil {
		return RepositoryMutatorRoute{}, fmt.Errorf("is latest generation: %w", err)
	} else if !latest {
		return RepositoryMutatorRoute{}, ErrRepositoryReadOnly
	}

	consistentSecondaries, err := r.rs.GetConsistentSecondaries(ctx, virtualStorage, relativePath, primary)
	if err != nil {
		return RepositoryMutatorRoute{}, fmt.Errorf("consistent secondaries: %w", err)
	}

	assignedStorages, err := r.ag.GetHostAssignments(ctx, virtualStorage, relativePath)
	if err != nil {
		return RepositoryMutatorRoute{}, fmt.Errorf("get host assignments: %w", err)
	}

	var route RepositoryMutatorRoute
	for _, assigned := range assignedStorages {
		node, healthy := healthySet[assigned]
		if assigned == primary {
			route.Primary = node
			continue
		}

		if _, consistent := consistentSecondaries[node.Storage]; !consistent || !healthy {
			route.ReplicationTargets = append(route.ReplicationTargets, assigned)
			continue
		}

		route.Secondaries = append(route.Secondaries, node)
	}

	if (route.Primary == RouterNode{}) {
		// AssignmentGetter interface defines that the primary must always be assigned.
		// While this case should not commonly happen, we must handle it here for now.
		// This can be triggered if the primary is demoted and unassigned during the RPC call.
		// The three SQL queries above are done non-transactionally. Once the variable
		// replication factor and repository specific primaries are enabled by default, we should
		// combine the above queries in to a single call to remove this case and make the
		// whole operation more efficient.
		return RepositoryMutatorRoute{}, errPrimaryUnassigned
	}

	return route, nil
}
