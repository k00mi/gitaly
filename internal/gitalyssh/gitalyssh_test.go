package gitalyssh

import (
	"context"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/metadata"
)

func TestUploadPackEnv(t *testing.T) {
	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	md := metadata.Pairs("gitaly-servers", base64.StdEncoding.EncodeToString([]byte(`{"default":{"address":"unix:///tmp/sock","token":"hunter1"}}`)))
	ctx = metadata.NewIncomingContext(ctx, md)

	req := gitalypb.SSHUploadPackRequest{
		Repository: testRepo,
	}

	var pbMarshaler jsonpb.Marshaler
	expectedPayload, err := pbMarshaler.MarshalToString(&req)
	require.NoError(t, err)

	env, err := UploadPackEnv(ctx, &req)

	require.NoError(t, err)
	require.Subset(t, env, []string{
		fmt.Sprintf("GIT_SSH_COMMAND=%s upload-pack", filepath.Join(config.Config.BinDir, "gitaly-ssh")),
		fmt.Sprintf("GITALY_PAYLOAD=%s", expectedPayload),
	})
}
