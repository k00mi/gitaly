package wiki

import (
	"bytes"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWikiGetPageVersionsRequest(t *testing.T) {
	wikiRepo, wikiRepoPath, cleanupFunc := setupWikiRepo(t)
	defer cleanupFunc()

	ctx, cancel := testhelper.Context()
	defer cancel()

	server, serverSocketPath := runWikiServiceServer(t)
	defer server.Stop()

	client, conn := newWikiClient(t, serverSocketPath)
	defer conn.Close()
	pageTitle := "WikiGetPageVersions"

	content := bytes.Repeat([]byte("Mock wiki page content"), 10000)
	writeWikiPage(t, client, wikiRepo, pageTitle, content)
	v1cid := testhelper.MustRunCommand(t, nil, "git", "-C", wikiRepoPath, "log", "-1", "--format=%H")
	updateWikiPage(t, client, wikiRepo, pageTitle, []byte("New content"))
	v2cid := testhelper.MustRunCommand(t, nil, "git", "-C", wikiRepoPath, "log", "-1", "--format=%H")

	gitAuthor := &pb.CommitAuthor{
		Name:  []byte("Ahmad Sherif"),
		Email: []byte("ahmad@gitlab.com"),
	}

	testCases := []struct {
		desc     string
		request  *pb.WikiGetPageVersionsRequest
		versions []*pb.WikiPageVersion
	}{
		{
			desc: "No page found",
			request: &pb.WikiGetPageVersionsRequest{
				Repository: wikiRepo,
				PagePath:   []byte("not-found"),
			},
			versions: nil,
		},
		{
			desc: "2 version found",
			request: &pb.WikiGetPageVersionsRequest{
				Repository: wikiRepo,
				PagePath:   []byte(pageTitle),
			},
			versions: []*pb.WikiPageVersion{
				{
					Commit: &pb.GitCommit{
						Id:        strings.TrimRight(string(v2cid), "\n"),
						Body:      []byte("Update WikiGetPageVersions"),
						Subject:   []byte("Update WikiGetPageVersions"),
						Author:    gitAuthor,
						Committer: gitAuthor,
						ParentIds: []string{strings.TrimRight(string(v1cid), "\n")},
					},
					Format: "markdown",
				},
				{
					Commit: &pb.GitCommit{
						Id:        strings.TrimRight(string(v1cid), "\n"),
						Body:      []byte("Add WikiGetPageVersions"),
						Subject:   []byte("Add WikiGetPageVersions"),
						Author:    gitAuthor,
						Committer: gitAuthor,
					},
					Format: "markdown",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			stream, err := client.WikiGetPageVersions(ctx, tc.request)
			require.NoError(t, err)
			require.NoError(t, stream.CloseSend())

			response, err := stream.Recv()
			require.NoError(t, err)

			require.Len(t, response.GetVersions(), len(tc.versions))
			for i, version := range response.GetVersions() {
				v2 := tc.versions[i]
				assertWikiPageVersionEqual(t, version, v2, "%d blew up", i)
			}
		})
	}
}

func assertWikiPageVersionEqual(t *testing.T, expected, actual *pb.WikiPageVersion, msg ...interface{}) bool {
	assert.Equal(t, expected.GetFormat(), actual.GetFormat())
	assert.NoError(t, testhelper.GitCommitEqual(expected.GetCommit(), actual.GetCommit()))

	switch len(msg) {
	case 0:
		t.Logf("WikiPageVersion differs: %v != %v", expected, actual)
	case 1:
		t.Log(msg[0])
	default:
		t.Logf(msg[0].(string), msg[1:])
	}

	return t.Failed()
}
