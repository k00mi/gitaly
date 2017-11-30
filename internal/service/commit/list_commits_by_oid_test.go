package commit

import (
	"io"
	"testing"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestSuccessfulListCommitsByOidRequest(t *testing.T) {
	server, serverSocketPath := startTestServices(t)
	defer server.Stop()

	client, conn := newCommitServiceClient(t, serverSocketPath)
	defer conn.Close()

	commits := []*pb.GitCommit{
		{
			Id:        "bf6e164cac2dc32b1f391ca4290badcbe4ffc5fb",
			Subject:   []byte("Commit #10"),
			Body:      []byte("Commit #10\n"),
			Author:    dummyCommitAuthor(1500320272),
			Committer: dummyCommitAuthor(1500320272),
			ParentIds: []string{"9d526f87b82e2b2fd231ca44c95508e5e85624ca"},
		},
		{
			Id:        "79b06233d3dc769921576771a4e8bee4b439595d",
			Subject:   []byte("Commit #1"),
			Body:      []byte("Commit #1\n"),
			Author:    dummyCommitAuthor(1500320254),
			Committer: dummyCommitAuthor(1500320254),
			ParentIds: []string{"1a0b36b3cdad1d2ee32457c102a8c0b7056fa863"},
		},
	}

	testCases := []struct {
		desc            string
		request         *pb.ListCommitsByOidRequest
		expectedCommits []*pb.GitCommit
	}{
		{
			desc: "find one commit",
			request: &pb.ListCommitsByOidRequest{
				Oid: []string{commits[0].Id},
			},
			expectedCommits: commits[0:1],
		},
		{
			desc: "find multiple commits",
			request: &pb.ListCommitsByOidRequest{
				Oid: []string{commits[0].Id, commits[1].Id},
			},
			expectedCommits: commits,
		},
		{
			desc: "no query",
			request: &pb.ListCommitsByOidRequest{
				Oid: []string{},
			},
			expectedCommits: []*pb.GitCommit{},
		},
		{
			desc: "empty query",
			request: &pb.ListCommitsByOidRequest{
				Oid: []string{""},
			},
			expectedCommits: []*pb.GitCommit{},
		},
		{
			desc: "partial oids",
			request: &pb.ListCommitsByOidRequest{
				Oid: []string{commits[0].Id[0:10], commits[1].Id[0:8]},
			},
			expectedCommits: commits,
		},
		{
			desc: "unknown oids",
			request: &pb.ListCommitsByOidRequest{
				Oid: []string{"deadbeef", "987654321"},
			},
			expectedCommits: []*pb.GitCommit{},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {

			request := testCase.request
			request.Repository = testRepo

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c, err := client.ListCommitsByOid(ctx, request)
			if err != nil {
				t.Fatal(err)
			}

			receivedCommits := []*pb.GitCommit{}
			for {
				resp, err := c.Recv()
				if err == io.EOF {
					break
				} else if err != nil {
					t.Fatal(err)
				}

				receivedCommits = append(receivedCommits, resp.GetCommits()...)
			}

			require.Equal(t, len(testCase.expectedCommits), len(receivedCommits), "number of commits received")

			for i, receivedCommit := range receivedCommits {
				require.Equal(t, testCase.expectedCommits[i], receivedCommit, "mismatched commit")
			}
		})
	}
}
