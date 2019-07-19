package praefect

import (
	"context"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	gitalyauth "gitlab.com/gitlab-org/gitaly/auth"
	gitaly_config "gitlab.com/gitlab-org/gitaly/internal/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/models"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"
	serverPkg "gitlab.com/gitlab-org/gitaly/internal/server"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

// TestReplicatorProcessJobs verifies that a replicator will schedule jobs for
// all whitelisted repos
func TestReplicatorProcessJobsWhitelist(t *testing.T) {
	var (
		cfg = config.Config{
			PrimaryServer: &models.GitalyServer{
				Name:       "default",
				ListenAddr: "tcp://gitaly-primary.example.com",
			},
			SecondaryServers: []*models.GitalyServer{
				{
					Name:       "backup1",
					ListenAddr: "tcp://gitaly-backup1.example.com",
				},
				{
					Name:       "backup2",
					ListenAddr: "tcp://gitaly-backup2.example.com",
				},
			},
			Whitelist: []string{"abcd1234", "edfg5678"},
		}
		datastore   = NewMemoryDatastore(cfg)
		coordinator = NewCoordinator(logrus.New(), datastore)
		resultsCh   = make(chan result)
		replman     = NewReplMgr(
			cfg.SecondaryServers[1].Name,
			logrus.New(),
			datastore,
			coordinator,
			WithWhitelist(cfg.Whitelist),
			WithReplicator(&mockReplicator{resultsCh}),
		)
	)

	for _, node := range []*models.GitalyServer{
		cfg.PrimaryServer,
		cfg.SecondaryServers[0],
		cfg.SecondaryServers[1],
	} {
		err := coordinator.RegisterNode(node.Name, node.ListenAddr)
		require.NoError(t, err)
	}

	ctx, cancel := testhelper.Context()

	errQ := make(chan error)

	go func() {
		errQ <- replman.ProcessBacklog(ctx)
	}()

	success := make(chan struct{})

	go func() {
		// we expect one job per whitelisted repo with each backend server
		for i := 0; i < len(cfg.Whitelist); i++ {
			result := <-resultsCh

			assert.Contains(t, cfg.Whitelist, result.source.RelativePath)
			assert.Equal(t, cfg.SecondaryServers[1].Name, result.target.Storage)
			assert.Equal(t, cfg.PrimaryServer.Name, result.source.Storage)
		}

		cancel()
		require.EqualError(t, <-errQ, context.Canceled.Error())
		success <- struct{}{}
	}()

	select {

	case <-success:
		return

	case <-time.After(time.Second):
		t.Fatalf("unable to iterate over expected jobs")

	}

}

type result struct {
	source models.Repository
	target Node
}

type mockReplicator struct {
	resultsCh chan<- result
}

func (mr *mockReplicator) Replicate(ctx context.Context, source models.Repository, target Node) error {
	select {

	case mr.resultsCh <- result{source, target}:
		return nil

	case <-ctx.Done():
		return ctx.Err()

	}

	return nil
}

func TestReplicate(t *testing.T) {
	srv, srvSocketPath := runFullGitalyServer(t)
	defer srv.Stop()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	backupStorageName := "backup"

	backupDir, err := ioutil.TempDir(testhelper.GitlabTestStoragePath(), backupStorageName)
	require.NoError(t, err)
	defer func() {
		os.RemoveAll(backupDir)
	}()

	oldStorages := gitaly_config.Config.Storages
	defer func() {
		gitaly_config.Config.Storages = oldStorages
	}()

	gitaly_config.Config.Storages = append(gitaly_config.Config.Storages, gitaly_config.Storage{
		Name: backupStorageName,
		Path: backupDir,
	},
		gitaly_config.Storage{
			Name: "default",
			Path: testhelper.GitlabTestStoragePath(),
		},
	)

	ctx, cancel := testhelper.Context()
	defer cancel()

	connOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(gitalyauth.RPCCredentials(testhelper.RepositoryAuthToken)),
	}
	conn, err := grpc.Dial(srvSocketPath, connOpts...)
	require.NoError(t, err)

	commitID := testhelper.CreateCommit(t, testRepoPath, "master", &testhelper.CreateCommitOpts{
		Message: "a commit",
	})

	ctx, err = helper.InjectGitalyServers(ctx, "default", srvSocketPath, testhelper.RepositoryAuthToken)
	require.NoError(t, err)

	var replicator defaultReplicator
	require.NoError(t, replicator.Replicate(
		ctx,
		models.Repository{Storage: "default", RelativePath: testRepo.GetRelativePath()},
		Node{
			cc:      conn,
			Storage: backupStorageName,
		}))

	replicatedPath := filepath.Join(backupDir, filepath.Base(testRepoPath))
	testhelper.MustRunCommand(t, nil, "git", "-C", replicatedPath, "cat-file", "-e", commitID)
}

func runFullGitalyServer(t *testing.T) (*grpc.Server, string) {
	server := serverPkg.NewInsecure(RubyServer)
	serverSocketPath := testhelper.GetTemporaryGitalySocketFileName()

	listener, err := net.Listen("unix", serverSocketPath)
	if err != nil {
		t.Fatal(err)
	}

	go server.Serve(listener)

	return server, "unix://" + serverSocketPath
}

var RubyServer *rubyserver.Server

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	defer testhelper.MustHaveNoChildProcess()

	gitaly_config.Config.Auth = gitaly_config.Auth{Token: testhelper.RepositoryAuthToken}

	var err error
	gitaly_config.Config.GitlabShell.Dir, err = filepath.Abs("testdata/gitlab-shell")
	if err != nil {
		log.Fatal(err)
	}

	testhelper.ConfigureGitalySSH()

	RubyServer, err = rubyserver.Start()
	if err != nil {
		log.Fatal(err)
	}
	defer RubyServer.Stop()

	return m.Run()
}
