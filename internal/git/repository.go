package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git/repository"
)

var (
	ErrReferenceNotFound = errors.New("reference not found")
)

// Repository represents a Git repository.
type Repository interface {
	// ContainsRef checks if a ref in the repository exists.
	ContainsRef(ctx context.Context, ref string) (bool, error)

	// GetReference looks up and returns the given reference. Returns a
	// ReferenceNotFound error if the reference was not found.
	GetReference(ctx context.Context, ref string) (Reference, error)

	// GetReferences returns references matching the given pattern.
	GetReferences(ctx context.Context, pattern string) ([]Reference, error)

	// GetBranch looks up and returns the given branch. Returns a
	// ErrReferenceNotFound if it wasn't found.
	GetBranch(ctx context.Context, branch string) (Reference, error)

	// GetBranches returns all branches.
	GetBranches(ctx context.Context) ([]Reference, error)
}

// localRepository represents a local Git repository.
type localRepository struct {
	repo repository.GitRepo
}

// NewRepository creates a new Repository from its protobuf representation.
func NewRepository(repo repository.GitRepo) Repository {
	return &localRepository{
		repo: repo,
	}
}

// command creates a Git Command with the given args and Repository, executed
// in the Repository. It validates the arguments in the command before
// executing.
func (repo *localRepository) command(ctx context.Context, globals []Option, cmd Cmd) (*command.Command, error) {
	return SafeCmd(ctx, repo.repo, globals, cmd)
}

func (repo *localRepository) ContainsRef(ctx context.Context, ref string) (bool, error) {
	if ref == "" {
		return false, errors.New("repository cannot contain empty reference name")
	}

	cmd, err := repo.command(ctx, nil, SubCmd{
		Name:  "log",
		Flags: []Option{Flag{Name: "--max-count=1"}},
		Args:  []string{ref},
	})
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}

	return cmd.Wait() == nil, nil
}

func (repo *localRepository) GetReference(ctx context.Context, ref string) (Reference, error) {
	refs, err := repo.GetReferences(ctx, ref)
	if err != nil {
		return Reference{}, err
	}

	if len(refs) == 0 {
		return Reference{}, ErrReferenceNotFound
	}

	return refs[0], nil
}

func (repo *localRepository) GetReferences(ctx context.Context, pattern string) ([]Reference, error) {
	var args []string
	if pattern != "" {
		args = []string{pattern}
	}

	cmd, err := repo.command(ctx, nil, SubCmd{
		Name:  "for-each-ref",
		Flags: []Option{Flag{Name: "--format=%(refname)%00%(objectname)%00%(symref)"}},
		Args:  args,
	})
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(cmd)

	var refs []Reference
	for scanner.Scan() {
		line := bytes.SplitN(scanner.Bytes(), []byte{0}, 3)
		if len(line) != 3 {
			return nil, errors.New("unexpected reference format")
		}

		if len(line[2]) == 0 {
			refs = append(refs, NewReference(string(line[0]), string(line[1])))
		} else {
			refs = append(refs, NewSymbolicReference(string(line[0]), string(line[1])))
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading standard input: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	return refs, nil
}

func (repo *localRepository) GetBranch(ctx context.Context, branch string) (Reference, error) {
	if strings.HasPrefix(branch, "refs/heads/") {
		return repo.GetReference(ctx, branch)
	}

	if strings.HasPrefix(branch, "heads/") {
		branch = strings.TrimPrefix(branch, "heads/")
	}
	return repo.GetReference(ctx, "refs/heads/"+branch)
}

func (repo *localRepository) GetBranches(ctx context.Context) ([]Reference, error) {
	return repo.GetReferences(ctx, "refs/heads/")
}
