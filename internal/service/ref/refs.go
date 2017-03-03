package ref

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	master = []byte("refs/heads/master")
	// We declare the following functions in variables so that we can override them in our tests
	findBranchNames = _findBranchNames
	headReference   = _headReference
)

func handleGitCommand(w refNamesWriter, r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := w.AddRef(scanner.Bytes()); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return w.Flush()
}

func findRefs(writer refNamesWriter, repo *pb.Repository, pattern string) error {
	if repo == nil {
		message := "Bad Request (empty repository)"
		log.Printf("FindRefs: %q", message)
		return grpc.Errorf(codes.InvalidArgument, message)
	}

	repoPath := repo.Path

	log.Printf("FindRefs: RepoPath=%q Pattern=%q", repoPath, pattern)

	cmd, err := helper.GitCommandReader("--git-dir", repoPath, "for-each-ref", pattern, "--format=%(refname)")
	if err != nil {
		return err
	}
	defer cmd.Kill()

	handleGitCommand(writer, cmd)

	return cmd.Wait()
}

// FindAllBranchNames creates a stream of ref names for all branches in the given repository
func (s *server) FindAllBranchNames(in *pb.FindAllBranchNamesRequest, stream pb.Ref_FindAllBranchNamesServer) error {
	return findRefs(newFindAllBranchNamesWriter(stream, s.MaxMsgSize), in.Repository, "refs/heads")
}

// FindAllTagNames creates a stream of ref names for all tags in the given repository
func (s *server) FindAllTagNames(in *pb.FindAllTagNamesRequest, stream pb.Ref_FindAllTagNamesServer) error {
	return findRefs(newFindAllTagNamesWriter(stream, s.MaxMsgSize), in.Repository, "refs/tags")
}

func _findBranchNames(repoPath string) ([][]byte, error) {
	var names [][]byte

	cmd, err := helper.GitCommandReader("--git-dir", repoPath, "for-each-ref", "refs/heads", "--format=%(refname)")
	if err != nil {
		return nil, err
	}
	defer cmd.Kill()

	scanner := bufio.NewScanner(cmd)
	for scanner.Scan() {
		names, _ = appendRef(names, scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading standard input: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return names, nil
}

func _headReference(repoPath string) ([]byte, error) {
	var headRef []byte

	cmd, err := helper.GitCommandReader("--git-dir", repoPath, "rev-parse", "--symbolic-full-name", "HEAD")
	if err != nil {
		return nil, err
	}
	defer cmd.Kill()

	scanner := bufio.NewScanner(cmd)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	headRef = scanner.Bytes()

	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return headRef, nil
}

func defaultBranchName(repoPath string) ([]byte, error) {
	branches, err := findBranchNames(repoPath)

	if err != nil {
		return nil, err
	}

	// Return empty ref name if there are no branches
	if len(branches) == 0 {
		return nil, nil
	}

	// Return first branch name if there's only one
	if len(branches) == 1 {
		return branches[0], nil
	}

	hasMaster := false
	headRef, err := headReference(repoPath)
	if err != nil {
		return nil, err
	}
	for _, branch := range branches {
		// Return HEAD if it corresponds to a branch
		if bytes.Equal(headRef, branch) {
			return headRef, nil
		}
		if bytes.Equal(branch, master) {
			hasMaster = true
		}
	}
	// Return `ref/names/master` if it exists
	if hasMaster {
		return master, nil
	}
	// If all else fails, return the first branch name
	return branches[0], nil
}

// FindDefaultBranchName returns the default branch name for the given repository
func (s *server) FindDefaultBranchName(ctx context.Context, in *pb.FindDefaultBranchNameRequest) (*pb.FindDefaultBranchNameResponse, error) {
	if in.Repository == nil {
		message := "Bad Request (empty repository)"
		log.Printf("FindDefaultBranchName: %q", message)
		return nil, grpc.Errorf(codes.InvalidArgument, message)
	}

	repoPath := in.Repository.Path

	log.Printf("FindDefaultBranchName: RepoPath=%q", repoPath)

	defaultBranchName, err := defaultBranchName(repoPath)
	if err != nil {
		return nil, err
	}

	return &pb.FindDefaultBranchNameResponse{Name: defaultBranchName}, nil
}

// FindRefName returns the first refname of a Repository
func (s *server) FindRefName(ctx context.Context, in *pb.FindRefNameRequest) (*pb.FindRefNameResponse, error) {
	return nil, nil
}
