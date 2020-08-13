// +build postgres

package datastore

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func BenchmarkPostgresReplicationEventQueue_Acknowledge(b *testing.B) {
	benchCases := []struct {
		desc  string
		setup map[JobState]int
		// this parameter needed as Acknowledge could run really fast for tiny setup, but the setup itself takes a way more time
		// that is why we limit amount of runs artificially, otherwise it could run almost indefinitely.
		maxRuns int
	}{
		{
			desc:    "small",
			setup:   map[JobState]int{JobStateReady: 10, JobStateInProgress: 10, JobStateFailed: 10},
			maxRuns: 100,
		},
		{
			desc:    "medium",
			setup:   map[JobState]int{JobStateReady: 1_000, JobStateInProgress: 100, JobStateFailed: 100},
			maxRuns: 20,
		},
		{
			desc:    "big",
			setup:   map[JobState]int{JobStateReady: 100_000, JobStateInProgress: 100, JobStateFailed: 100},
			maxRuns: 5,
		},
		{
			desc:    "huge",
			setup:   map[JobState]int{JobStateReady: 1_000_000, JobStateInProgress: 100, JobStateFailed: 100},
			maxRuns: 1,
		},
	}

	db := getDB(b)

	ctx, cancel := testhelper.Context()
	defer cancel()

	queue := PostgresReplicationEventQueue{db.DB}
	eventTmpl := ReplicationEvent{
		Job: ReplicationJob{
			Change:            UpdateRepo,
			RelativePath:      "/project/path-",
			TargetNodeStorage: "gitaly-1",
			SourceNodeStorage: "gitaly-0",
			VirtualStorage:    "praefect",
		},
	}

	getEventIDs := func(events []ReplicationEvent) []uint64 {
		ids := make([]uint64, len(events))
		for i, event := range events {
			ids[i] = event.ID
		}
		return ids
	}

	for _, bc := range benchCases {
		b.Run(bc.desc, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				if n > bc.maxRuns {
					b.Logf("max amount of runs done: %d", bc.maxRuns)
					break
				}

				b.StopTimer()
				b.ResetTimer()

				db.TruncateAll(b)

				total := 0
				for _, count := range bc.setup {
					// at first we need to enqueue all events and then move them to proper states
					total += count
				}

				_, err := db.DB.ExecContext(
					ctx,
					`INSERT INTO replication_queue (state, lock_id, job)
					SELECT 'ready', 'praefect|gitaly-1|/project/path-'|| T.I, ('{"change":"update","relative_path":"/project/path-'|| T.I || '","virtual_storage":"praefect","source_node_storage":"gitaly-0","target_node_storage":"gitaly-1"}')::JSONB
					FROM GENERATE_SERIES(1, $1) T(I)`,
					total,
				)
				require.NoError(b, err)

				_, err = db.DB.ExecContext(
					ctx,
					`INSERT INTO replication_queue_lock
					SELECT DISTINCT lock_id FROM replication_queue`,
				)
				require.NoError(b, err)

				events, err := queue.Dequeue(ctx, eventTmpl.Job.VirtualStorage, eventTmpl.Job.TargetNodeStorage, bc.setup[JobStateFailed]+bc.setup[JobStateInProgress])
				require.NoError(b, err)

				_, err = queue.Acknowledge(ctx, JobStateFailed, getEventIDs(events[:bc.setup[JobStateFailed]]))
				require.NoError(b, err)

				events, err = queue.Dequeue(ctx, eventTmpl.Job.VirtualStorage, eventTmpl.Job.TargetNodeStorage, 10)
				require.NoError(b, err)

				b.StartTimer()
				_, err = queue.Acknowledge(ctx, JobStateCompleted, getEventIDs(events))
				b.StopTimer()
				require.NoError(b, err)
			}
		})
	}
}
