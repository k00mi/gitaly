package datastore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

type requireState func(t *testing.T, ctx context.Context, vss virtualStorageState, ss storageState)
type repositoryStoreFactory func(t *testing.T, storages map[string][]string) (RepositoryStore, requireState)

func TestRepositoryStore_Memory(t *testing.T) {
	testRepositoryStore(t, func(t *testing.T, storages map[string][]string) (RepositoryStore, requireState) {
		rs := NewMemoryRepositoryStore(storages)
		return rs, func(t *testing.T, _ context.Context, vss virtualStorageState, ss storageState) {
			t.Helper()
			require.Equal(t, vss, rs.virtualStorageState)
			require.Equal(t, ss, rs.storageState)
		}
	})
}

func testRepositoryStore(t *testing.T, newStore repositoryStoreFactory) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	const (
		vs   = "virtual-storage-1"
		repo = "repository-1"
		stor = "storage-1"
	)

	t.Run("IncrementGeneration", func(t *testing.T) {
		t.Run("creates a new record for primary", func(t *testing.T) {
			rs, requireState := newStore(t, nil)

			require.NoError(t, rs.IncrementGeneration(ctx, vs, repo, "primary", []string{"secondary-1"}))
			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"repository-1": 0,
					},
				},
				storageState{
					"virtual-storage-1": {
						"repository-1": {
							"primary": 0,
						},
					},
				},
			)
		})

		t.Run("increments existing record for primary", func(t *testing.T) {
			rs, requireState := newStore(t, nil)

			require.NoError(t, rs.SetGeneration(ctx, vs, repo, "primary", 0))
			require.NoError(t, rs.IncrementGeneration(ctx, vs, repo, "primary", []string{"secondary-1"}))
			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"repository-1": 1,
					},
				},
				storageState{
					"virtual-storage-1": {
						"repository-1": {
							"primary": 1,
						},
					},
				},
			)
		})

		t.Run("increments existing for up to date secondary", func(t *testing.T) {
			rs, requireState := newStore(t, nil)

			require.NoError(t, rs.SetGeneration(ctx, vs, repo, "primary", 1))
			require.NoError(t, rs.SetGeneration(ctx, vs, repo, "up-to-date-secondary", 1))
			require.NoError(t, rs.SetGeneration(ctx, vs, repo, "outdated-secondary", 0))
			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"repository-1": 1,
					},
				},
				storageState{
					"virtual-storage-1": {
						"repository-1": {
							"primary":              1,
							"up-to-date-secondary": 1,
							"outdated-secondary":   0,
						},
					},
				},
			)

			require.NoError(t, rs.IncrementGeneration(ctx, vs, repo, "primary", []string{
				"up-to-date-secondary", "outdated-secondary", "non-existing-secondary"}))
			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"repository-1": 2,
					},
				},
				storageState{
					"virtual-storage-1": {
						"repository-1": {
							"primary":              2,
							"up-to-date-secondary": 2,
							"outdated-secondary":   0,
						},
					},
				},
			)
		})
	})

	t.Run("SetGeneration", func(t *testing.T) {
		t.Run("creates a record", func(t *testing.T) {
			rs, requireState := newStore(t, nil)

			err := rs.SetGeneration(ctx, vs, repo, stor, 1)
			require.NoError(t, err)
			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"repository-1": 1,
					},
				},
				storageState{
					"virtual-storage-1": {
						"repository-1": {
							"storage-1": 1,
						},
					},
				},
			)
		})

		t.Run("updates existing record", func(t *testing.T) {
			rs, requireState := newStore(t, nil)

			require.NoError(t, rs.SetGeneration(ctx, vs, repo, stor, 1))
			require.NoError(t, rs.SetGeneration(ctx, vs, repo, stor, 0))
			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"repository-1": 1,
					},
				},
				storageState{
					"virtual-storage-1": {
						"repository-1": {
							"storage-1": 0,
						},
					},
				},
			)
		})

		t.Run("increments stays monotonic", func(t *testing.T) {
			rs, requireState := newStore(t, nil)

			require.NoError(t, rs.SetGeneration(ctx, vs, repo, stor, 1))
			require.NoError(t, rs.SetGeneration(ctx, vs, repo, stor, 0))
			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"repository-1": 1,
					},
				},
				storageState{
					"virtual-storage-1": {
						"repository-1": {
							"storage-1": 0,
						},
					},
				},
			)

			require.NoError(t, rs.IncrementGeneration(ctx, vs, repo, stor, nil))
			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"repository-1": 2,
					},
				},
				storageState{
					"virtual-storage-1": {
						"repository-1": {
							"storage-1": 2,
						},
					},
				},
			)

			require.NoError(t, rs.IncrementGeneration(ctx, vs, repo, "storage-2", nil))
			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"repository-1": 3,
					},
				},
				storageState{
					"virtual-storage-1": {
						"repository-1": {
							"storage-1": 2,
							"storage-2": 3,
						},
					},
				},
			)
		})
	})

	t.Run("GetGeneration", func(t *testing.T) {
		rs, _ := newStore(t, nil)

		generation, err := rs.GetGeneration(ctx, vs, repo, stor)
		require.NoError(t, err)
		require.Equal(t, GenerationUnknown, generation)

		require.NoError(t, rs.SetGeneration(ctx, vs, repo, stor, 0))

		generation, err = rs.GetGeneration(ctx, vs, repo, stor)
		require.NoError(t, err)
		require.Equal(t, 0, generation)
	})

	t.Run("GetReplicatedGeneration", func(t *testing.T) {
		t.Run("no previous record allowed", func(t *testing.T) {
			rs, _ := newStore(t, nil)

			gen, err := rs.GetReplicatedGeneration(ctx, vs, repo, "source", "target")
			require.NoError(t, err)
			require.Equal(t, GenerationUnknown, gen)

			require.NoError(t, rs.SetGeneration(ctx, vs, repo, "source", 0))
			gen, err = rs.GetReplicatedGeneration(ctx, vs, repo, "source", "target")
			require.NoError(t, err)
			require.Equal(t, 0, gen)
		})

		t.Run("upgrade allowed", func(t *testing.T) {
			rs, _ := newStore(t, nil)

			require.NoError(t, rs.SetGeneration(ctx, vs, repo, "source", 1))
			gen, err := rs.GetReplicatedGeneration(ctx, vs, repo, "source", "target")
			require.NoError(t, err)
			require.Equal(t, 1, gen)

			require.NoError(t, rs.SetGeneration(ctx, vs, repo, "target", 0))
			gen, err = rs.GetReplicatedGeneration(ctx, vs, repo, "source", "target")
			require.NoError(t, err)
			require.Equal(t, 1, gen)
		})

		t.Run("downgrade prevented", func(t *testing.T) {
			rs, _ := newStore(t, nil)

			require.NoError(t, rs.SetGeneration(ctx, vs, repo, "target", 1))

			_, err := rs.GetReplicatedGeneration(ctx, vs, repo, "source", "target")
			require.Equal(t, DowngradeAttemptedError{vs, repo, "target", 1, GenerationUnknown}, err)

			require.NoError(t, rs.SetGeneration(ctx, vs, repo, "source", 1))
			_, err = rs.GetReplicatedGeneration(ctx, vs, repo, "source", "target")
			require.Equal(t, DowngradeAttemptedError{vs, repo, "target", 1, 1}, err)

			require.NoError(t, rs.SetGeneration(ctx, vs, repo, "source", 0))
			_, err = rs.GetReplicatedGeneration(ctx, vs, repo, "source", "target")
			require.Equal(t, DowngradeAttemptedError{vs, repo, "target", 1, 0}, err)
		})
	})

	t.Run("DeleteRepository", func(t *testing.T) {
		t.Run("delete non-existing", func(t *testing.T) {
			rs, _ := newStore(t, nil)

			require.Equal(t,
				RepositoryNotExistsError{vs, repo, stor},
				rs.DeleteRepository(ctx, vs, repo, stor),
			)
		})

		t.Run("delete existing", func(t *testing.T) {
			rs, requireState := newStore(t, nil)

			require.NoError(t, rs.SetGeneration(ctx, "deleted", "deleted", "deleted", 0))

			require.NoError(t, rs.SetGeneration(ctx, "virtual-storage-1", "other-storages-remain", "deleted-storage", 0))
			require.NoError(t, rs.SetGeneration(ctx, "virtual-storage-1", "other-storages-remain", "remaining-storage", 0))

			require.NoError(t, rs.SetGeneration(ctx, "virtual-storage-2", "deleted-repo", "deleted-storage", 0))
			require.NoError(t, rs.SetGeneration(ctx, "virtual-storage-2", "other-repo-remains", "remaining-storage", 0))

			requireState(t, ctx,
				virtualStorageState{
					"deleted": {
						"deleted": 0,
					},
					"virtual-storage-1": {
						"other-storages-remain": 0,
					},
					"virtual-storage-2": {
						"deleted-repo":       0,
						"other-repo-remains": 0,
					},
				},
				storageState{
					"deleted": {
						"deleted": {
							"deleted": 0,
						},
					},
					"virtual-storage-1": {
						"other-storages-remain": {
							"deleted-storage":   0,
							"remaining-storage": 0,
						},
					},
					"virtual-storage-2": {
						"deleted-repo": {
							"deleted-storage": 0,
						},
						"other-repo-remains": {
							"remaining-storage": 0,
						},
					},
				},
			)

			require.NoError(t, rs.DeleteRepository(ctx, "deleted", "deleted", "deleted"))
			require.NoError(t, rs.DeleteRepository(ctx, "virtual-storage-1", "other-storages-remain", "deleted-storage"))
			require.NoError(t, rs.DeleteRepository(ctx, "virtual-storage-2", "deleted-repo", "deleted-storage"))

			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-2": {
						"other-repo-remains": 0,
					},
				},
				storageState{
					"virtual-storage-1": {
						"other-storages-remain": {
							"remaining-storage": 0,
						},
					},
					"virtual-storage-2": {
						"other-repo-remains": {
							"remaining-storage": 0,
						},
					},
				},
			)
		})
	})

	t.Run("RenameRepository", func(t *testing.T) {
		t.Run("rename non-existing", func(t *testing.T) {
			rs, _ := newStore(t, nil)

			require.Equal(t,
				RepositoryNotExistsError{vs, repo, stor},
				rs.RenameRepository(ctx, vs, repo, stor, "repository-2"),
			)
		})

		t.Run("rename existing", func(t *testing.T) {
			rs, requireState := newStore(t, nil)

			require.NoError(t, rs.SetGeneration(ctx, vs, "renamed-all", "storage-1", 0))
			require.NoError(t, rs.SetGeneration(ctx, vs, "renamed-some", "storage-1", 0))
			require.NoError(t, rs.SetGeneration(ctx, vs, "renamed-some", "storage-2", 0))

			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"renamed-all":  0,
						"renamed-some": 0,
					},
				},
				storageState{
					"virtual-storage-1": {
						"renamed-all": {
							"storage-1": 0,
						},
						"renamed-some": {
							"storage-1": 0,
							"storage-2": 0,
						},
					},
				},
			)

			require.NoError(t, rs.RenameRepository(ctx, vs, "renamed-all", "storage-1", "renamed-all-new"))
			require.NoError(t, rs.RenameRepository(ctx, vs, "renamed-some", "storage-1", "renamed-some-new"))

			requireState(t, ctx,
				virtualStorageState{
					"virtual-storage-1": {
						"renamed-all-new":  0,
						"renamed-some-new": 0,
					},
				},
				storageState{
					"virtual-storage-1": {
						"renamed-all-new": {
							"storage-1": 0,
						},
						"renamed-some-new": {
							"storage-1": 0,
						},
						"renamed-some": {
							"storage-2": 0,
						},
					},
				},
			)
		})
	})

	t.Run("GetConsistentSecondaries", func(t *testing.T) {
		rs, requireState := newStore(t, map[string][]string{
			vs: []string{"primary", "consistent-secondary", "inconsistent-secondary", "no-record"},
		})

		t.Run("unknown generations", func(t *testing.T) {
			secondaries, err := rs.GetConsistentSecondaries(ctx, vs, repo, "primary")
			require.NoError(t, err)
			require.Empty(t, secondaries)
		})

		require.NoError(t, rs.SetGeneration(ctx, vs, repo, "primary", 1))
		require.NoError(t, rs.SetGeneration(ctx, vs, repo, "consistent-secondary", 1))
		require.NoError(t, rs.SetGeneration(ctx, vs, repo, "inconsistent-secondary", 0))
		requireState(t, ctx,
			virtualStorageState{
				"virtual-storage-1": {
					"repository-1": 1,
				},
			},
			storageState{
				"virtual-storage-1": {
					"repository-1": {
						"primary":                1,
						"consistent-secondary":   1,
						"inconsistent-secondary": 0,
					},
				},
			},
		)

		t.Run("consistent secondary", func(t *testing.T) {
			secondaries, err := rs.GetConsistentSecondaries(ctx, vs, repo, "primary")
			require.NoError(t, err)
			require.Equal(t, map[string]struct{}{"consistent-secondary": struct{}{}}, secondaries)
		})

		t.Run("primary on unknown generation", func(t *testing.T) {
			secondaries, err := rs.GetConsistentSecondaries(ctx, vs, repo, "no-record")
			require.NoError(t, err)
			require.Empty(t, secondaries)
		})
	})

	t.Run("IsLatestGeneration", func(t *testing.T) {
		rs, _ := newStore(t, nil)

		latest, err := rs.IsLatestGeneration(ctx, vs, repo, "no-expected-record")
		require.NoError(t, err)
		require.True(t, latest)

		require.NoError(t, rs.SetGeneration(ctx, vs, repo, "up-to-date", 1))
		require.NoError(t, rs.SetGeneration(ctx, vs, repo, "outdated", 0))

		latest, err = rs.IsLatestGeneration(ctx, vs, repo, "no-record")
		require.NoError(t, err)
		require.False(t, latest)

		latest, err = rs.IsLatestGeneration(ctx, vs, repo, "outdated")
		require.NoError(t, err)
		require.False(t, latest)

		latest, err = rs.IsLatestGeneration(ctx, vs, repo, "up-to-date")
		require.NoError(t, err)
		require.True(t, latest)
	})
}
