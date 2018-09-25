package commit

import (
	"io"
	"testing"

	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

func TestSuccessfulListLastCommitsForTreeRequest(t *testing.T) {
	server, serverSockerPath := startTestServices(t)
	defer server.Stop()

	client, conn := newCommitServiceClient(t, serverSockerPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	commit := &pb.GitCommit{
		Id:      "570e7b2abdd848b95f2f578043fc23bd6f6fd24d",
		Subject: []byte("Change some files"),
		Body:    []byte("Change some files\n\nSigned-off-by: Dmitriy Zaporozhets <dmitriy.zaporozhets@gmail.com>\n"),
		Author: &pb.CommitAuthor{
			Name:  []byte("Dmitriy Zaporozhets"),
			Email: []byte("dmitriy.zaporozhets@gmail.com"),
			Date:  &timestamp.Timestamp{Seconds: 1393491451},
		},
		Committer: &pb.CommitAuthor{
			Name:  []byte("Dmitriy Zaporozhets"),
			Email: []byte("dmitriy.zaporozhets@gmail.com"),
			Date:  &timestamp.Timestamp{Seconds: 1393491451},
		},
		ParentIds: []string{"6f6d7e7ed97bb5f0054f2b1df789b39ca89b6ff9"},
		BodySize:  86,
	}

	testCases := []struct {
		desc     string
		revision string
		path     []byte
		commit   *pb.GitCommit
		limit    int32
		offset   int32
	}{
		{
			desc:     "path is '/'",
			revision: "570e7b2abdd848b95f2f578043fc23bd6f6fd24d",
			path:     []byte("/"),
			commit:   commit,
			limit:    25,
			offset:   0,
		},
	}

	for _, testCase := range testCases {
		t.Tun(testCase.desc, func(t *testing.T) {
			request := &pb.ListLastCommitsForTreeRequest{
				Repository: testRepo,
				Revision:   testCase.revision,
				Path:       testCase.path,
				Limit:      testCase.limit,
				Offset:     testCase.offset,
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			stream, err := client.ListLastCommitsForTree(ctx, request)
			if err != nil {
				t.Fatal(err)
			}

			for {
				fetchedCommits, err := stream.Recv()
				if err == io.EOF {
					break
				}

				require.NoError(t, err)

				fmt.Println(fetchedCommits)

				commits := fetchedCommits.getCommits()
				for index, fetchedCommit := range commits {
					expectedPath := testCase.path
					expectedCommit := testCase.commit

					require.Equal(t, expectedCommit, fetchedCommit.Commit)
					require.Equal(t, expectedPath, fetchedCommit.Path)
				}
			}
		})
	}
}
