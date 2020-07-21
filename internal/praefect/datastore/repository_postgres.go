package datastore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore/glsql"
)

// GenerationUnknown is used to indicate lack of generation number in
// a replication job. Older instances can produce replication jobs
// without a generation number.
const GenerationUnknown = -1

// DowngradeAttemptedError is returned when attempting to get the replicated generation for a source repository
// that does not upgrade the target repository.
type DowngradeAttemptedError struct {
	virtualStorage      string
	relativePath        string
	storage             string
	currentGeneration   int
	attemptedGeneration int
}

func (err DowngradeAttemptedError) Error() string {
	return fmt.Sprintf("attempted downgrading %q -> %q -> %q from generation %d to %d",
		err.virtualStorage, err.relativePath, err.storage, err.currentGeneration, err.attemptedGeneration,
	)
}

// RepositoryNotExistsError is returned when trying to perform an operation on a non-existent repository.
type RepositoryNotExistsError struct {
	virtualStorage string
	relativePath   string
	storage        string
}

// Is checks whetehr the other errors is of the same type.
func (err RepositoryNotExistsError) Is(other error) bool {
	_, ok := other.(RepositoryNotExistsError)
	return ok
}

// Error returns the errors message.
func (err RepositoryNotExistsError) Error() string {
	return fmt.Sprintf("repository %q -> %q -> %q does not exist",
		err.virtualStorage, err.relativePath, err.storage,
	)
}

// RepositoryStore provides access to repository state.
type RepositoryStore interface {
	// GetGeneration gets the repository's generation on a given storage.
	GetGeneration(ctx context.Context, virtualStorage, relativePath, storage string) (int, error)
	// IncrementGeneration increments the primary's and the up to date secondaries' generations.
	IncrementGeneration(ctx context.Context, virtualStorage, relativePath, primary string, secondaries []string) error
	// SetGeneration sets the repository's generation on the given storage. If the generation is higher
	// than the virtual storage's generation, it is set to match as well to guarantee monotonic increments.
	SetGeneration(ctx context.Context, virtualStorage, relativePath, storage string, generation int) error
	// GetReplicatedGeneration returns the generation propagated by applying the replication. If the generation would
	// downgrade, a DowngradeAttemptedError is returned.
	GetReplicatedGeneration(ctx context.Context, virtualStorage, relativePath, source, target string) (int, error)
	// DeleteRepository deletes the repository from the virtual storage and the storage. Returns
	// RepositoryNotExistsError when trying to delete a repository which has no record in the virtual storage
	// or the storage.
	DeleteRepository(ctx context.Context, virtualStorage, relativePath, storage string) error
	// RenameRepository updates a repository's relative path. It renames the virtual storage wide record as well
	// as the storage's which is calling it. Returns RepositoryNotExistsError when trying to rename a repository
	// which has no record in the virtual storage or the storage.
	RenameRepository(ctx context.Context, virtualStorage, relativePath, storage, newRelativePath string) error
	// GetConsistentSecondaries checks which secondaries are on the same generation as the primary and returns them.
	// If the primary's generation is unknown, all secondaries are considered inconsistent.
	GetConsistentSecondaries(ctx context.Context, virtualStorage, relativePath, primary string) (map[string]struct{}, error)
	// IsLatestGeneration checks whether the repository is on the latest generation or not. If the repository does not
	// have an expected generation, every storage is considered to be on the latest version.
	IsLatestGeneration(ctx context.Context, virtualStorage, relativePath, storage string) (bool, error)
}

// PostgresRepositoryStore is a Postgres implementation of RepositoryStore.
// Refer to the interface for method documentation.
type PostgresRepositoryStore struct {
	db glsql.Querier
	storages
}

// NewLocalRepositoryStore returns a Postgres implementation of RepositoryStore.
func NewPostgresRepositoryStore(db glsql.Querier, configuredStorages map[string][]string) *PostgresRepositoryStore {
	return &PostgresRepositoryStore{db: db, storages: storages(configuredStorages)}
}

func (rs *PostgresRepositoryStore) GetGeneration(ctx context.Context, virtualStorage, relativePath, storage string) (int, error) {
	const q = `
SELECT generation
FROM storage_repositories
WHERE virtual_storage = $1
AND relative_path = $2
AND storage = $3
`

	var gen int
	if err := rs.db.QueryRowContext(ctx, q, virtualStorage, relativePath, storage).Scan(&gen); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return GenerationUnknown, nil
		}

		return 0, err
	}

	return gen, nil
}

func (rs *PostgresRepositoryStore) IncrementGeneration(ctx context.Context, virtualStorage, relativePath, primary string, secondaries []string) error {
	// The query works as follows:
	//   1. `next_generation` CTE increments the latest generation by 1. If no previous records exists,
	//      the generation starts from 0.
	//   2. `base_generation` CTE gets the primary's current generation. A secondary has to be on the primary's
	//      generation, otherwise its generation won't be incremented. This avoids any issues where a concurrent
	//      reference transaction has failed and the secondary is no longer up to date when we are incrementing
	//      the generations.
	//   3. `eligible_secondaries` filters out secondaries which participated in a transaction but failed a
	///     concurrent transaction.
	//   4. `eligible_storages` CTE combines the primary and the up to date secondaries in a list of storages to
	//      to increment the generation for.
	//   5. Finally, we upsert the records in 'storage_repositories' table to match the new generation for the
	//      eligble storages.

	const q = `
WITH next_generation AS (
	INSERT INTO repositories (
		virtual_storage,
		relative_path,
		generation
	) VALUES ($1, $2, 0)
	ON CONFLICT (virtual_storage, relative_path) DO
		UPDATE SET generation = repositories.generation + 1
	RETURNING virtual_storage, relative_path, generation
), base_generation AS (
	SELECT virtual_storage, relative_path, generation 
	FROM storage_repositories
	WHERE virtual_storage = $1
	AND relative_path = $2
	AND storage = $3 
	FOR UPDATE
), eligible_secondaries AS (
	SELECT storage
	FROM storage_repositories
	NATURAL JOIN base_generation
	WHERE storage = ANY($4::text[])
	FOR UPDATE
), eligible_storages AS (
	SELECT storage
	FROM eligible_secondaries
		UNION
	SELECT $3
)

INSERT INTO storage_repositories AS sr (
	virtual_storage,
	relative_path,
	storage,
	generation
)
SELECT virtual_storage, relative_path, storage, generation
FROM eligible_storages
CROSS JOIN next_generation
ON CONFLICT (virtual_storage, relative_path, storage) DO
	UPDATE SET generation = EXCLUDED.generation
`
	_, err := rs.db.ExecContext(ctx, q, virtualStorage, relativePath, primary, pq.StringArray(secondaries))
	return err
}

func (rs *PostgresRepositoryStore) SetGeneration(ctx context.Context, virtualStorage, relativePath, storage string, generation int) error {
	const q = `
WITH repository AS (
	INSERT INTO repositories (
		virtual_storage,
		relative_path,
		generation
	) VALUES ($1, $2, $4)
	ON CONFLICT (virtual_storage, relative_path) DO
		UPDATE SET generation = EXCLUDED.generation
		WHERE repositories.generation < EXCLUDED.generation
)

INSERT INTO storage_repositories (
	virtual_storage,
	relative_path,
	storage,
	generation
)
VALUES ($1, $2, $3, $4)
ON CONFLICT (virtual_storage, relative_path, storage) DO UPDATE SET
	generation = EXCLUDED.generation
`

	_, err := rs.db.ExecContext(ctx, q, virtualStorage, relativePath, storage, generation)
	return err
}

func (rs *PostgresRepositoryStore) GetReplicatedGeneration(ctx context.Context, virtualStorage, relativePath, source, target string) (int, error) {
	const q = `
SELECT storage, generation
FROM storage_repositories
WHERE virtual_storage = $1
AND relative_path = $2
AND storage = ANY($3)
`

	rows, err := rs.db.QueryContext(ctx, q, virtualStorage, relativePath, pq.StringArray([]string{source, target}))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	sourceGeneration := GenerationUnknown
	targetGeneration := GenerationUnknown
	for rows.Next() {
		var storage string
		var generation int
		if err := rows.Scan(&storage, &generation); err != nil {
			return 0, err
		}

		switch storage {
		case source:
			sourceGeneration = generation
		case target:
			targetGeneration = generation
		default:
			return 0, fmt.Errorf("unexpected storage: %s", storage)
		}
	}

	if err := rows.Err(); err != nil {
		return 0, err
	}

	if targetGeneration != GenerationUnknown && targetGeneration >= sourceGeneration {
		return 0, DowngradeAttemptedError{
			virtualStorage:      virtualStorage,
			relativePath:        relativePath,
			storage:             target,
			currentGeneration:   targetGeneration,
			attemptedGeneration: sourceGeneration,
		}
	}

	return sourceGeneration, nil
}

func (rs *PostgresRepositoryStore) DeleteRepository(ctx context.Context, virtualStorage, relativePath, storage string) error {
	const q = `
WITH repo AS (
	DELETE FROM repositories
	WHERE virtual_storage = $1
	AND relative_path = $2
)

DELETE FROM storage_repositories
WHERE virtual_storage = $1
AND relative_path = $2
AND storage = $3
`

	result, err := rs.db.ExecContext(ctx, q, virtualStorage, relativePath, storage)
	if err != nil {
		return err
	}

	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return RepositoryNotExistsError{
			virtualStorage: virtualStorage,
			relativePath:   relativePath,
			storage:        storage,
		}
	}

	return nil
}

func (rs *PostgresRepositoryStore) RenameRepository(ctx context.Context, virtualStorage, relativePath, storage, newRelativePath string) error {
	const q = `
WITH repo AS (
	UPDATE repositories
	SET relative_path = $4
	WHERE virtual_storage = $1
	AND relative_path = $2
)

UPDATE storage_repositories
SET relative_path = $4
WHERE virtual_storage = $1
AND relative_path = $2
AND storage = $3
`

	result, err := rs.db.ExecContext(ctx, q, virtualStorage, relativePath, storage, newRelativePath)
	if err != nil {
		return err
	}

	if n, err := result.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return RepositoryNotExistsError{
			virtualStorage: virtualStorage,
			relativePath:   relativePath,
			storage:        storage,
		}
	}

	return err
}

func (rs *PostgresRepositoryStore) GetConsistentSecondaries(ctx context.Context, virtualStorage, relativePath, primary string) (map[string]struct{}, error) {
	const q = `
WITH expected AS (
	SELECT virtual_storage, relative_path, generation
	FROM storage_repositories
	WHERE virtual_storage = $1
	AND relative_path = $2
	AND storage = $3
)

SELECT storage
FROM storage_repositories
NATURAL JOIN expected
WHERE storage = ANY($4::text[])
`
	secondaries, err := rs.storages.secondaries(virtualStorage, primary)
	if err != nil {
		return nil, err
	}

	rows, err := rs.db.QueryContext(ctx, q, virtualStorage, relativePath, primary, pq.StringArray(secondaries))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	consistentSecondaries := make(map[string]struct{}, len(secondaries))
	for rows.Next() {
		var storage string
		if err := rows.Scan(&storage); err != nil {
			return nil, err
		}

		consistentSecondaries[storage] = struct{}{}
	}

	return consistentSecondaries, rows.Err()
}

func (rs *PostgresRepositoryStore) IsLatestGeneration(ctx context.Context, virtualStorage, relativePath, storage string) (bool, error) {
	const q = `
SELECT COALESCE(r.generation = sr.generation, false)
FROM repositories AS r
LEFT JOIN storage_repositories AS sr
	ON sr.virtual_storage = r.virtual_storage
	AND sr.relative_path = r.relative_path
	AND sr.storage = $3
WHERE r.virtual_storage = $1
AND r.relative_path = $2
`

	var isLatest bool
	if err := rs.db.QueryRowContext(ctx, q, virtualStorage, relativePath, storage).Scan(&isLatest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// if there is no record of the expected generation, we'll have to consider the storage
			// up to date as this will be the case on repository creation
			return true, nil
		}

		return false, err
	}

	return isLatest, nil
}
