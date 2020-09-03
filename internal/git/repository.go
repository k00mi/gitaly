package git

import (
	"context"
	"errors"
	"os/exec"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git/repository"
)

// Repository represents a Git repository.
type Repository interface {
	// ContainsRef checks if a ref in the repository exists.
	ContainsRef(ctx context.Context, ref string) (bool, error)
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
