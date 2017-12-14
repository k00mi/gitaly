package remote

import (
	"fmt"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"

	"github.com/stretchr/testify/require"

	"golang.org/x/net/context"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestSuccessfulAddRemote(t *testing.T) {
	server, serverSocketPath := runRemoteServiceServer(t)
	defer server.Stop()

	client, conn := newRemoteClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testCases := []struct {
		description  string
		remoteName   string
		url          string
		mirrorRefmap string
	}{
		{
			description:  "creates a new remote",
			remoteName:   "my-remote",
			url:          "http://my-repo.git",
			mirrorRefmap: "",
		},
		{
			description:  "if a remote with the same name exists, it updates it",
			remoteName:   "my-remote",
			url:          "johndoe@host:my-new-repo.git",
			mirrorRefmap: "",
		},
		{
			description:  "sets the remote as mirror if mirror_refmap is present",
			remoteName:   "my-mirror-remote",
			url:          "http://my-mirror-repo.git",
			mirrorRefmap: "all_refs",
		},
		{
			description:  "doesn't set the remote as mirror if mirror_refmap is blank",
			remoteName:   "my-non-mirror-remote",
			url:          "http://my-non-mirror-repo.git",
			mirrorRefmap: "    ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			request := &pb.AddRemoteRequest{
				Repository:   testRepo,
				Name:         tc.remoteName,
				Url:          tc.url,
				MirrorRefmap: tc.mirrorRefmap,
			}

			_, err := client.AddRemote(ctx, request)
			require.NoError(t, err)

			remotes := testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "remote", "-v")

			require.Contains(t, string(remotes), fmt.Sprintf("%s\t%s (fetch)", tc.remoteName, tc.url))
			require.Contains(t, string(remotes), fmt.Sprintf("%s\t%s (push)", tc.remoteName, tc.url))

			mirrorConfigRegexp := fmt.Sprintf("remote.%s", tc.remoteName)
			mirrorConfig := string(testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "config", "--get-regexp", mirrorConfigRegexp))
			if strings.TrimSpace(tc.mirrorRefmap) != "" {
				require.Contains(t, mirrorConfig, "fetch +refs/*:refs/*")
				require.Contains(t, mirrorConfig, "mirror true")
				require.Contains(t, mirrorConfig, "prune true")
			} else {
				require.NotContains(t, mirrorConfig, "mirror true")
			}
		})
	}
}

func TestFailedAddRemoteDueToValidation(t *testing.T) {
	server, serverSocketPath := runRemoteServiceServer(t)
	defer server.Stop()

	client, conn := newRemoteClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testCases := []struct {
		description string
		remoteName  string
		url         string
	}{
		{
			description: "Remote name empty",
			url:         "http://my-repo.git",
		},
		{
			description: "Remote name blank",
			remoteName:  "    ",
			url:         "http://my-repo.git",
		},
		{
			description: "URL empty",
			remoteName:  "my-remote",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			request := &pb.AddRemoteRequest{
				Repository:   testRepo,
				Name:         tc.remoteName,
				Url:          tc.url,
				MirrorRefmap: "",
			}

			_, err := client.AddRemote(ctx, request)
			testhelper.AssertGrpcError(t, err, codes.InvalidArgument, "")
		})
	}
}
