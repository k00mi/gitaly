package diff

import (
	"log"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/diff"
	"gitlab.com/gitlab-org/gitaly/internal/helper"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *server) CommitDiff(in *pb.CommitDiffRequest, stream pb.Diff_CommitDiffServer) error {
	if err := validateRequest(in); err != nil {
		return err
	}

	repoPath := in.Repository.GetPath()
	leftSha := in.LeftCommitId
	rightSha := in.RightCommitId

	log.Printf("CommitDiff: RepoPath=%q LeftCommitId=%q RightCommitId=%q", repoPath, leftSha, rightSha)

	cmd := helper.GitCommand("git", "--git-dir", repoPath, "diff", "--full-index", "--find-renames", leftSha, rightSha)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return grpc.Errorf(codes.Unavailable, "CommitDiff: Failed obtaining command stdout pipe")
	}
	defer stdout.Close()

	if err := cmd.Start(); err != nil {
		return grpc.Errorf(codes.Unavailable, "CommitDiff: Failed starting command")
	}
	defer helper.CleanUpProcessGroup(cmd) // Ensure brute force subprocess clean-up

	diffParser := diff.NewDiffParser(stdout)

	for diffParser.Parse() {
		diff := diffParser.Diff()
		err = stream.Send(&pb.CommitDiffResponse{
			FromPath:  diff.FromPath,
			ToPath:    diff.ToPath,
			FromId:    diff.FromID,
			ToId:      diff.ToID,
			OldMode:   diff.OldMode,
			NewMode:   diff.NewMode,
			Binary:    diff.Binary,
			RawChunks: diff.RawChunks,
		})

		if err != nil {
			return grpc.Errorf(codes.Unavailable, "CommitDiff: Failed sending diff")
		}
	}

	if err := diffParser.Err(); err != nil {
		log.Printf("CommitDiff: Parsing diff in repo %q between %q and %q failed: %v", repoPath, leftSha, rightSha, err)
		return grpc.Errorf(codes.Internal, "CommitDiff: Parsing diff output failed")
	}

	if err := cmd.Wait(); err != nil {
		return grpc.Errorf(codes.Unavailable, "CommitDiff: Command failed to complete successfully")
	}

	return nil
}

func validateRequest(in *pb.CommitDiffRequest) error {
	if in.Repository == nil || in.Repository.GetPath() == "" {
		return grpc.Errorf(codes.InvalidArgument, "CommitDiff: Repository is empty")
	}
	if in.LeftCommitId == "" {
		return grpc.Errorf(codes.InvalidArgument, "CommitDiff: LeftCommitId is empty")
	}
	if in.RightCommitId == "" {
		return grpc.Errorf(codes.InvalidArgument, "CommitDiff: RightCommitId is empty")
	}

	return nil
}
