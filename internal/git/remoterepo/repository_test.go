package remoterepo

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper/testserver"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func TestRepository(t *testing.T) {
	_, serverSocketPath, cleanup := testserver.RunInternalGitalyServer(t, config.Config.Storages, config.Config.Auth.Token)
	defer cleanup()

	ctx, cancel := testhelper.Context()
	defer cancel()

	ctx, err := helper.InjectGitalyServers(ctx, "default", serverSocketPath, config.Config.Auth.Token)
	require.NoError(t, err)

	git.TestRepository(t, func(t testing.TB, pbRepo *gitalypb.Repository) git.Repository {
		t.Helper()

		r, err := New(
			helper.OutgoingToIncoming(ctx),
			testhelper.TestRepository(),
			client.NewPool(),
		)
		require.NoError(t, err)
		return r
	})
}
