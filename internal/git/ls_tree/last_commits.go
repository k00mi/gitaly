package ls_tree

import (
	"context"
	"io/ioutil"
	"regexp"
	"strings"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	log "github.com/sirupsen/logrus"
)

func LastCommitsForTree(ctx context.Context, repo *pb.Repository, revision string, path string) (*pb.GitCommit, error) {
	var commits []*pb.GitCommit

	cmd, err := git.Command(ctx, repo, "ls-tree", "-z", "--full-name", revision, "--", path)
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadAll(cmd)
	if err != nil {
		return nil, err
	}

	re := regex.MustCompile("[0-9a-f]{40}")
	commitIDs := re.FindAllString(contents, -1)

	for _, commitID := range commitIDs {
		append(commits, GetCommit(ctx, repo, strings.TrimSpace(string(commitID))))
	}

	return commits, nil
}
