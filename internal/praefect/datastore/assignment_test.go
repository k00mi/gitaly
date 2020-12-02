// +build postgres

package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestAssignmentStore_GetHostAssignments(t *testing.T) {
	type assignment struct {
		virtualStorage string
		relativePath   string
		storage        string
	}

	configuredStorages := []string{"storage-1", "storage-2", "storage-3"}
	for _, tc := range []struct {
		desc                string
		virtualStorage      string
		existingAssignments []assignment
		expectedAssignments []string
		error               error
	}{
		{
			desc:           "virtual storage not found",
			virtualStorage: "invalid-virtual-storage",
			error:          newVirtualStorageNotFoundError("invalid-virtual-storage"),
		},
		{
			desc:                "configured storages fallback when no records",
			virtualStorage:      "virtual-storage",
			expectedAssignments: configuredStorages,
		},
		{
			desc:           "configured storages fallback when a repo exists in different virtual storage",
			virtualStorage: "virtual-storage",
			existingAssignments: []assignment{
				{virtualStorage: "other-virtual-storage", relativePath: "relative-path", storage: "storage-1"},
			},
			expectedAssignments: configuredStorages,
		},
		{
			desc:           "configured storages fallback when a different repo exists in the virtual storage ",
			virtualStorage: "virtual-storage",
			existingAssignments: []assignment{
				{virtualStorage: "virtual-storage", relativePath: "other-relative-path", storage: "storage-1"},
			},
			expectedAssignments: configuredStorages,
		},
		{
			desc:           "unconfigured storages are ignored",
			virtualStorage: "virtual-storage",
			existingAssignments: []assignment{
				{virtualStorage: "virtual-storage", relativePath: "relative-path", storage: "unconfigured-storage"},
			},
			expectedAssignments: configuredStorages,
		},
		{
			desc:           "assignments found",
			virtualStorage: "virtual-storage",
			existingAssignments: []assignment{
				{virtualStorage: "virtual-storage", relativePath: "relative-path", storage: "storage-1"},
				{virtualStorage: "virtual-storage", relativePath: "relative-path", storage: "storage-2"},
				{virtualStorage: "virtual-storage", relativePath: "relative-path", storage: "unconfigured"},
			},
			expectedAssignments: []string{"storage-1", "storage-2"},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			db := getDB(t)

			for _, assignment := range tc.existingAssignments {
				_, err := db.ExecContext(ctx, `
					INSERT INTO repositories (virtual_storage, relative_path)
					VALUES ($1, $2)
					ON CONFLICT DO NOTHING
				`, assignment.virtualStorage, assignment.relativePath)
				require.NoError(t, err)

				_, err = db.ExecContext(ctx, `
					INSERT INTO repository_assignments VALUES ($1, $2, $3)
				`, assignment.virtualStorage, assignment.relativePath, assignment.storage)
				require.NoError(t, err)
			}

			actualAssignments, err := NewAssignmentStore(
				db,
				map[string][]string{"virtual-storage": configuredStorages},
			).GetHostAssignments(ctx, tc.virtualStorage, "relative-path")
			require.Equal(t, tc.error, err)
			require.ElementsMatch(t, tc.expectedAssignments, actualAssignments)
		})
	}
}
