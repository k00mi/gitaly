// +build postgres

package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestAssignmentStore_GetHostAssignments(t *testing.T) {
	type repository struct {
		virtualStorage string
		relativePath   string
	}

	type storage struct {
		virtualStorage string
		relativePath   string
		storage        string
		assigned       bool
	}

	for _, tc := range []struct {
		desc           string
		virtualStorage string
		repositories   []repository
		storages       []storage
		assignments    []string
		error          error
	}{
		{
			desc:           "virtual storage not found",
			virtualStorage: "invalid-virtual-storage",
			error:          newVirtualStorageNotFoundError("invalid-virtual-storage"),
		},
		{
			desc:           "not found when no records",
			virtualStorage: "virtual-storage",
			error:          newAssignmentsNotFoundError("virtual-storage", "relative-path"),
		},
		{
			desc:           "not found when only repository record",
			virtualStorage: "virtual-storage",
			repositories: []repository{
				{virtualStorage: "virtual-storage", relativePath: "relative-path"},
			},
			error: newAssignmentsNotFoundError("virtual-storage", "relative-path"),
		},
		{
			desc:           "not found with incorrect virtual storage",
			virtualStorage: "virtual-storage",
			repositories: []repository{
				{virtualStorage: "other-virtual-storage", relativePath: "relative-path"},
			},
			storages: []storage{
				{virtualStorage: "other-virtual-storage", relativePath: "relative-path", storage: "storage-1", assigned: true},
			},
			error: newAssignmentsNotFoundError("virtual-storage", "relative-path"),
		},
		{
			desc:           "not found with incorrect relative path",
			virtualStorage: "virtual-storage",
			repositories: []repository{
				{virtualStorage: "virtual-storage", relativePath: "other-relative-path"},
			},
			storages: []storage{
				{virtualStorage: "virtual-storage", relativePath: "other-relative-path", storage: "storage-1", assigned: true},
			},
			error: newAssignmentsNotFoundError("virtual-storage", "relative-path"),
		},

		{
			desc:           "not found when only storage record",
			virtualStorage: "virtual-storage",
			storages: []storage{
				{virtualStorage: "virtual-storage", relativePath: "relative-path", storage: "storage-1", assigned: true},
			},
			error: newAssignmentsNotFoundError("virtual-storage", "relative-path"),
		},
		{
			desc:           "unconfigured storages are ignored",
			virtualStorage: "virtual-storage",
			storages: []storage{
				{virtualStorage: "virtual-storage", relativePath: "relative-path", storage: "unconfigured-storage", assigned: true},
			},
			error: newAssignmentsNotFoundError("virtual-storage", "relative-path"),
		},
		{
			desc:           "assignments found",
			virtualStorage: "virtual-storage",
			repositories: []repository{
				{virtualStorage: "virtual-storage", relativePath: "relative-path"},
			},
			storages: []storage{
				{virtualStorage: "virtual-storage", relativePath: "relative-path", storage: "storage-1", assigned: true},
				{virtualStorage: "virtual-storage", relativePath: "relative-path", storage: "storage-2", assigned: true},
				{virtualStorage: "virtual-storage", relativePath: "relative-path", storage: "storage-3", assigned: false},
				{virtualStorage: "virtual-storage", relativePath: "relative-path", storage: "unconfigured", assigned: true},
			},
			assignments: []string{"storage-1", "storage-2"},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			db := getDB(t)

			for _, repository := range tc.repositories {
				_, err := db.ExecContext(ctx, `
					INSERT INTO repositories (virtual_storage, relative_path)
					VALUES ($1, $2)
				`, repository.virtualStorage, repository.relativePath)
				require.NoError(t, err)
			}

			for _, storage := range tc.storages {
				_, err := db.ExecContext(ctx, `
					INSERT INTO storage_repositories (virtual_storage, relative_path, storage, assigned, generation)
					VALUES ($1, $2, $3, $4, 0)
				`, storage.virtualStorage, storage.relativePath, storage.storage, storage.assigned)
				require.NoError(t, err)
			}

			assignments, err := NewAssignmentStore(
				db,
				map[string][]string{"virtual-storage": {"storage-1", "storage-2", "storage-3"}},
			).GetHostAssignments(ctx, tc.virtualStorage, "relative-path")
			require.Equal(t, tc.error, err)
			require.ElementsMatch(t, tc.assignments, assignments)
		})
	}
}
