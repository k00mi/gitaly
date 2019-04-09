package praefect_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/praefect"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
)

// TestMemoryDatastoreWhitelist verifies that the in-memory datastore will
// populate itself with the correct replication jobs and shards when initialized
// with a configuration file specifying the shard and whitelisted repositories.
func TestMemoryDatastoreWhitelist(t *testing.T) {
	var (
		immediate = time.Unix(1300000000, 0)
		cfg       = config.Config{
			PrimaryServer: &config.GitalyServer{
				Name: "default",
			},
			SecondaryServers: []*config.GitalyServer{
				{
					Name: "backup-1",
				},
				{
					Name: "backup-2",
				},
			},
			Whitelist: []string{
				"abcd1234",
				"5678efgh",
			},
		}
		mds praefect.Datastore = praefect.NewMemoryDatastore(cfg, immediate)

		repo1 = praefect.Repository{
			RelativePath: cfg.Whitelist[0],
			Storage:      cfg.PrimaryServer.Name,
		}
		repo2 = praefect.Repository{
			RelativePath: cfg.Whitelist[1],
			Storage:      cfg.PrimaryServer.Name,
		}

		expectSecondaries = []string{
			cfg.SecondaryServers[0].Name,
			cfg.SecondaryServers[1].Name,
		}
	)

	for _, repo := range []praefect.Repository{repo1, repo2} {
		actualSecondaries, err := mds.GetSecondaries(repo)
		require.NoError(t, err)
		require.ElementsMatch(t, actualSecondaries, expectSecondaries)
	}

	var (
		backup1 = cfg.SecondaryServers[0]
		backup2 = cfg.SecondaryServers[1]

		backup1ExpectedJobs = []praefect.ReplJob{
			praefect.ReplJob{
				Target:    backup1.Name,
				Source:    repo1,
				Scheduled: immediate,
			},
			praefect.ReplJob{
				Target:    backup1.Name,
				Source:    repo2,
				Scheduled: immediate,
			},
		}
		backup2ExpectedJobs = []praefect.ReplJob{
			praefect.ReplJob{
				Target:    backup2.Name,
				Source:    repo1,
				Scheduled: immediate,
			},
			praefect.ReplJob{
				Target:    backup2.Name,
				Source:    repo2,
				Scheduled: immediate,
			},
		}
	)

	backup1ActualJobs, err := mds.GetReplJobs(backup1.Name, time.Time{}, 10)
	require.NoError(t, err)
	require.ElementsMatch(t, backup1ActualJobs, backup1ExpectedJobs)

	backup2ActualJobs, err := mds.GetReplJobs(backup2.Name, time.Time{}, 10)
	require.NoError(t, err)
	require.ElementsMatch(t, backup2ActualJobs, backup2ExpectedJobs)

}
