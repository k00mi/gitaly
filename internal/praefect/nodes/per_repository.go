package nodes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore/glsql"
)

// ErrNoPrimary is returned if the repository does not have a primary.
var ErrNoPrimary = errors.New("no primary")

// PerRepositoryElector implements an elector that selects a primary for each repository.
// It elects a healthy node with most recent generation as the primary. If all nodes are
// on the same generation, it picks one randomly to balance repositories in simple fashion.
type PerRepositoryElector struct {
	log         logrus.FieldLogger
	db          glsql.Querier
	hc          HealthChecker
	handleError func(error) error
}

// HealthChecker maintains node health statuses.
type HealthChecker interface {
	// HealthyNodes returns lists of healthy nodes by virtual storages.
	HealthyNodes() map[string][]string
}

// NewPerRepositoryElector returns a new per repository primary elector.
func NewPerRepositoryElector(log logrus.FieldLogger, db glsql.Querier, hc HealthChecker) *PerRepositoryElector {
	log = log.WithField("component", "PerRepositoryElector")
	return &PerRepositoryElector{
		log: log,
		db:  db,
		hc:  hc,
		handleError: func(err error) error {
			log.WithError(err).Error("failed performing failovers")
			return nil
		},
	}
}

// Run listens on the trigger channel for updates. On each update, it tries to elect new primaries for
// repositories which have an unhealthy primary. Blocks until the context is canceled or the trigger
// channel is closed. Returns the error from the context.
func (pr *PerRepositoryElector) Run(ctx context.Context, trigger <-chan struct{}) error {
	pr.log.Info("per repository elector started")
	defer pr.log.Info("per repository elector stopped")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-trigger:
			if !ok {
				return nil
			}

			if err := pr.performFailovers(ctx); err != nil {
				if err := pr.handleError(err); err != nil {
					return err
				}
			}
		}
	}
}

func (pr *PerRepositoryElector) performFailovers(ctx context.Context) error {
	healthyNodes := pr.hc.HealthyNodes()

	var virtualStorages, physicalStorages []string
	for virtualStorage, nodes := range healthyNodes {
		for _, node := range nodes {
			virtualStorages = append(virtualStorages, virtualStorage)
			physicalStorages = append(physicalStorages, node)
		}
	}

	if _, err := pr.db.ExecContext(ctx, `
WITH healthy_storages AS (
    SELECT unnest($1::text[]) AS virtual_storage, unnest($2::text[]) AS storage
)

UPDATE repositories
	SET "primary" = (
		SELECT storage
		FROM healthy_storages
		LEFT JOIN storage_repositories USING (virtual_storage, storage)
		WHERE virtual_storage = repositories.virtual_storage
		AND storage_repositories.relative_path = repositories.relative_path
		AND assigned
		ORDER BY generation DESC NULLS LAST, random()
		LIMIT 1
	)
WHERE NOT EXISTS (
	SELECT 1
	FROM healthy_storages
	WHERE virtual_storage = repositories.virtual_storage
	AND storage = repositories."primary"
)`, pq.StringArray(virtualStorages), pq.StringArray(physicalStorages)); err != nil {
		return fmt.Errorf("query: %w", err)
	}

	return nil
}

// GetPrimary returns the primary storage of a repository.
func (pr *PerRepositoryElector) GetPrimary(ctx context.Context, virtualStorage, relativePath string) (string, error) {
	var primary string
	if err := pr.db.QueryRowContext(ctx, `
SELECT "primary"
FROM repositories
WHERE virtual_storage = $1
AND relative_path = $2
AND "primary" IS NOT NULL
		`,
		virtualStorage,
		relativePath,
	).Scan(&primary); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNoPrimary
		}

		return "", fmt.Errorf("scan: %w", err)
	}

	return primary, nil
}
