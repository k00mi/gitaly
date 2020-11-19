package datastore

import (
	"context"
	"fmt"

	"github.com/lib/pq"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore/glsql"
)

func newVirtualStorageNotFoundError(virtualStorage string) error {
	return fmt.Errorf("virtual storage %q not found", virtualStorage)
}

func newAssignmentsNotFoundError(virtualStorage, relativePath string) error {
	return fmt.Errorf("host assignments for repository %q/%q not found", virtualStorage, relativePath)
}

// AssignmentStore manages host assignments in Postgres.
type AssignmentStore struct {
	db                 glsql.Querier
	configuredStorages map[string][]string
}

// NewAssignmentsStore returns a new AssignmentStore using the passed in database.
func NewAssignmentStore(db glsql.Querier, configuredStorages map[string][]string) AssignmentStore {
	return AssignmentStore{db: db, configuredStorages: configuredStorages}
}

func (s AssignmentStore) GetHostAssignments(ctx context.Context, virtualStorage, relativePath string) ([]string, error) {
	configuredStorages, ok := s.configuredStorages[virtualStorage]
	if !ok {
		return nil, newVirtualStorageNotFoundError(virtualStorage)
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT storage
FROM repositories
JOIN storage_repositories USING (virtual_storage, relative_path)
WHERE virtual_storage = $1
AND   relative_path = $2
AND   assigned
AND   storage = ANY($3::text[])
	`, virtualStorage, relativePath, pq.StringArray(configuredStorages))
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var assignedStorages []string
	for rows.Next() {
		var storage string
		if err := rows.Scan(&storage); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}

		assignedStorages = append(assignedStorages, storage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating rows: %w", err)
	}

	if len(assignedStorages) == 0 {
		return nil, newAssignmentsNotFoundError(virtualStorage, relativePath)
	}

	return assignedStorages, nil
}
