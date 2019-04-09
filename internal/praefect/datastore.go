/*Package praefect provides data models and datastore persistence abstractions
for tracking the state of repository replicas.

See original design discussion:
https://gitlab.com/gitlab-org/gitaly/issues/1495


*/
package praefect

import (
	"errors"
	"sync"
	"time"

	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
)

// ReplJob is an instance of a queued replication job. A replication job is
// meant for updating the repository so that it is synced with the primary
// copy. Scheduled indicates when a replication job should be performed.
type ReplJob struct {
	Target    string     // which storage location to replicate to?
	Source    Repository // source for replication
	Scheduled time.Time
}

// Datastore is a data persistence abstraction for all of Praefect's
// persistence needs
type Datastore interface {
	ReplJobsDatastore
	ReplicasDatastore
}

// ReplicasDatastore manages accessing and setting which secondary replicas
// backup a repository
type ReplicasDatastore interface {
	// GetSecondaries will retrieve all secondary replica storage locations for
	// a primary replica
	GetSecondaries(primary Repository) ([]string, error)

	// SetSecondaries will set the secondary storage locations for a repository
	// in a primary replica.
	SetSecondaries(primary Repository, secondaries []string) error
}

// ReplJobsDatastore represents the behavior needed for fetching and updating
// replication jobs from the datastore
type ReplJobsDatastore interface {
	// GetReplJobs fetches a list of chronologically ordered replication
	// jobs for the given storage replica
	GetReplJobs(storage string, since time.Time, count int) ([]ReplJob, error)

	// PutReplJob will update or create a replication job for the specified repo
	// on a specific storage node
	PutReplJob(repo Repository, when time.Time) error
}

// shard is a set of primary and secondary storage replicas for a project
type shard struct {
	primary     string
	secondaries []string
}

// MemoryDatastore is a simple datastore that isn't persisted to disk. It is
// only intended for early beta requirements and as a reference implementation
// for the eventual SQL implementation
type MemoryDatastore struct {
	mu          sync.RWMutex                    // locks entire datastore
	replicas    map[string]shard                // projectHash keyed to shards
	storageJobs map[string]map[string]time.Time // keyed by storage then project
}

// NewMemoryDatastore returns an initialized in-memory datastore
func NewMemoryDatastore(cfg config.Config, immediate time.Time) *MemoryDatastore {
	m := &MemoryDatastore{
		replicas:    map[string]shard{},
		storageJobs: map[string]map[string]time.Time{},
	}

	for _, project := range cfg.Whitelist {
		// store the configuration file specified shard
		m.replicas[project] = shard{
			primary: cfg.PrimaryServer.Name,
			secondaries: func() []string {
				servers := make([]string, len(cfg.SecondaryServers))
				for i, server := range cfg.SecondaryServers {
					servers[i] = server.Name
				}
				return servers
			}(),
		}

		// initialize replication job queue to replicate all whitelisted repos
		// to every secondary server
		for _, secondary := range cfg.SecondaryServers {
			projectJobs, ok := m.storageJobs[secondary.Name]
			if !ok {
				projectJobs = map[string]time.Time{}
				m.storageJobs[secondary.Name] = projectJobs
			}

			projectJobs[project] = immediate
		}

	}

	return m
}

// GetSecondaries will return the set of secondary storage locations for a
// given repository if they exist
func (md *MemoryDatastore) GetSecondaries(primary Repository) ([]string, error) {
	shard, _ := md.getShard(primary.RelativePath)

	return shard.secondaries, nil
}

// SetSecondaries will replace the set of replicas for a repository
func (md *MemoryDatastore) SetSecondaries(primary Repository, secondaries []string) error {
	md.mu.Lock()
	md.replicas[primary.RelativePath] = shard{
		primary:     primary.Storage,
		secondaries: secondaries,
	}
	md.mu.Unlock()

	return nil
}

func (md *MemoryDatastore) getShard(project string) (shard, bool) {
	md.mu.RLock()
	replicas, ok := md.replicas[project]
	md.mu.RUnlock()

	return replicas, ok
}

// ErrSecondariesMissing indicates the repository does not have any backup
// replicas
var ErrSecondariesMissing = errors.New("repository missing secondary replicas")

// GetReplJobs will return any replications jobs for the specified storage
// since the specified scheduled time up to the specified result limit.
func (md *MemoryDatastore) GetReplJobs(storage string, since time.Time, count int) ([]ReplJob, error) {
	md.mu.RLock()
	jobs := md.storageJobs[storage]
	md.mu.RUnlock()

	var results []ReplJob

	for project, scheduled := range jobs {
		if len(results) >= count {
			break
		}

		if scheduled.Before(since) {
			continue
		}

		shard, ok := md.getShard(project)
		if !ok {
			return nil, ErrSecondariesMissing
		}

		results = append(results, ReplJob{
			Source: Repository{
				RelativePath: project,
				Storage:      shard.primary,
			},
			Target:    storage,
			Scheduled: scheduled,
		})
	}

	return results, nil
}

// ErrInvalidReplTarget indicates a target repository cannot be chosen because
// it fails preconditions for being replicatable
var ErrInvalidReplTarget = errors.New("target repository fails preconditions for replication")

// PutReplJob will create or update an existing replication job by scheduling
// it at the specified time
func (md *MemoryDatastore) PutReplJob(target Repository, scheduled time.Time) error {
	md.mu.RLock()
	storageProjectJobs, ok := md.storageJobs[target.Storage]
	md.mu.RUnlock()

	if !ok {
		storageProjectJobs = map[string]time.Time{}
	}

	// target must be a secondary replica. By definition, a secondary replica
	// must have a corresponding primary to replicate from
	shard, ok := md.getShard(target.RelativePath)
	if !ok {
		return ErrInvalidReplTarget
	}

	found := false
	for _, secondary := range shard.secondaries {
		if secondary == target.Storage {
			found = true
			break
		}
	}

	if !found {
		return ErrInvalidReplTarget
	}

	storageProjectJobs[target.RelativePath] = scheduled

	md.mu.Lock()
	md.storageJobs[target.Storage] = storageProjectJobs
	md.mu.Unlock()

	return nil
}
