// +build postgres

package nodes

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/datastore"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

// HealthCheckerFunc is an adapter to turn a conforming function in to a HealthChecker.
type HealthCheckerFunc func() map[string][]string

func (fn HealthCheckerFunc) HealthyNodes() map[string][]string { return fn() }

func TestPerRepositoryElector(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	type storageRecord struct {
		generation int
		assigned   bool
	}

	type state map[string]map[string]map[string]storageRecord

	type matcher func(t testing.TB, primary string)
	any := func(expected ...string) matcher {
		return func(t testing.TB, primary string) {
			t.Helper()
			require.Contains(t, expected, primary)
		}
	}

	noPrimary := func() matcher {
		return func(t testing.TB, primary string) {
			t.Helper()
			require.Empty(t, primary)
		}
	}

	type steps []struct {
		healthyNodes map[string][]string
		error        error
		primary      matcher
	}

	for _, tc := range []struct {
		desc  string
		state state
		steps steps
	}{
		{
			desc: "elects the most up to date storage",
			state: state{
				"virtual-storage-1": {
					"relative-path-1": {
						"gitaly-1": {generation: 1},
						"gitaly-2": {generation: 0},
					},
				},
			},
			steps: steps{
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-1", "gitaly-2", "gitaly-3"},
					},
					primary: any("gitaly-1"),
				},
			},
		},
		{
			desc: "elects the most up to date healthy storage",
			state: state{
				"virtual-storage-1": {
					"relative-path-1": {
						"gitaly-1": {generation: 1},
						"gitaly-2": {generation: 0},
					},
				},
			},
			steps: steps{
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-2", "gitaly-3"},
					},
					primary: any("gitaly-2"),
				},
			},
		},
		{
			desc: "no valid primary",
			steps: steps{
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-2", "gitaly-3"},
					},
					error:   ErrNoPrimary,
					primary: noPrimary(),
				},
			},
		},
		{
			desc: "random healthy node on the latest generation",
			state: state{
				"virtual-storage-1": {
					"relative-path-1": {
						"gitaly-1": {generation: 0},
						"gitaly-2": {generation: 0},
					},
				},
			},
			steps: steps{
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-1", "gitaly-2", "gitaly-3"},
					},
					primary: any("gitaly-1", "gitaly-2"),
				},
			},
		},
		{
			desc: "fails over to up to date healthy note",
			state: state{
				"virtual-storage-1": {
					"relative-path-1": {
						"gitaly-1": {generation: 1},
						"gitaly-2": {generation: 1},
						"gitaly-3": {generation: 0},
					},
				},
			},
			steps: steps{
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-1", "gitaly-3"},
					},
					primary: any("gitaly-1"),
				},
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-2", "gitaly-3"},
					},
					primary: any("gitaly-2"),
				},
			},
		},
		{
			desc: "fails over to most up to date healthy note",
			state: state{
				"virtual-storage-1": {
					"relative-path-1": {
						"gitaly-1": {generation: 1},
						"gitaly-3": {generation: 0},
					},
				},
			},
			steps: steps{
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-1", "gitaly-2", "gitaly-3"},
					},
					primary: any("gitaly-1"),
				},
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-2", "gitaly-3"},
					},
					primary: any("gitaly-3"),
				},
			},
		},
		{
			desc: "fails over only to assigned nodes when assignments are set",
			state: state{
				"virtual-storage-1": {
					"relative-path-1": {
						"gitaly-1": {generation: 2, assigned: true},
						"gitaly-2": {generation: 1, assigned: true},
						"gitaly-3": {generation: 2, assigned: false},
					},
				},
			},
			steps: steps{
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-1", "gitaly-2", "gitaly-3"},
					},
					primary: any("gitaly-1"),
				},
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-2", "gitaly-3"},
					},
					primary: any("gitaly-2"),
				},
			},
		},
		{
			desc: "demotes primary when no valid candidates",
			state: state{
				"virtual-storage-1": {
					"relative-path-1": {
						"gitaly-1": {generation: 1, assigned: true},
						"gitaly-2": {generation: 1, assigned: false},
					},
				},
			},
			steps: steps{
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-1", "gitaly-2", "gitaly-3"},
					},
					primary: any("gitaly-1"),
				},
				{
					healthyNodes: map[string][]string{
						"virtual-storage-1": {"gitaly-2", "gitaly-3"},
					},
					error:   ErrNoPrimary,
					primary: noPrimary(),
				},
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			db := getDB(t)

			_, err := db.ExecContext(ctx,
				`INSERT INTO repositories (virtual_storage, relative_path) VALUES ('virtual-storage-1', 'relative-path-1')`,
			)
			require.NoError(t, err)

			rs := datastore.NewPostgresRepositoryStore(db, nil)
			for virtualStorage, relativePaths := range tc.state {
				for relativePath, storages := range relativePaths {
					for storage, record := range storages {
						require.NoError(t, rs.SetGeneration(ctx, virtualStorage, relativePath, storage, record.generation))

						if record.assigned {
							_, err := db.ExecContext(ctx, `
								INSERT INTO repository_assignments VALUES ($1, $2, $3)
							`, virtualStorage, relativePath, storage)
							require.NoError(t, err)
						}
					}
				}
			}

			for _, step := range tc.steps {
				elector := NewPerRepositoryElector(testhelper.DiscardTestLogger(t), db,
					HealthCheckerFunc(func() map[string][]string { return step.healthyNodes }),
				)
				elector.handleError = func(err error) error { return err }

				trigger := make(chan struct{}, 1)
				trigger <- struct{}{}
				close(trigger)

				require.NoError(t, elector.Run(ctx, trigger))

				primary, err := elector.GetPrimary(ctx, "virtual-storage-1", "relative-path-1")
				require.Equal(t, step.error, err)
				step.primary(t, primary)
			}
		})
	}
}
