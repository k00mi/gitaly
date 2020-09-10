package hook

import (
	"bytes"
	"context"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

func getRelativeObjectDirs(repoPath, gitObjectDir, gitAlternateObjectDirs string) (string, []string, error) {
	repoPathReal, err := filepath.EvalSymlinks(repoPath)
	if err != nil {
		return "", nil, err
	}

	gitObjDirRel, err := filepath.Rel(repoPathReal, gitObjectDir)
	if err != nil {
		return "", nil, err
	}

	var gitAltObjDirsRel []string

	for _, gitAltObjDirAbs := range strings.Split(gitAlternateObjectDirs, ":") {
		gitAltObjDirRel, err := filepath.Rel(repoPathReal, gitAltObjDirAbs)
		if err != nil {
			return "", nil, err
		}

		gitAltObjDirsRel = append(gitAltObjDirsRel, gitAltObjDirRel)
	}

	return gitObjDirRel, gitAltObjDirsRel, nil
}

func (m *GitLabHookManager) executeCustomHooks(ctx context.Context, stdout, stderr io.Writer, changes []byte, repo *gitalypb.Repository, env []string) error {
	// custom hooks execution
	repoPath, err := helper.GetRepoPath(repo)
	if err != nil {
		return err
	}
	executor, err := m.NewCustomHooksExecutor(repoPath, m.hooksConfig.CustomHooksDir, "pre-receive")
	if err != nil {
		return fmt.Errorf("creating custom hooks executor: %w", err)
	}

	if err = executor(
		ctx,
		nil,
		env,
		bytes.NewReader(changes),
		stdout,
		stderr,
	); err != nil {
		return fmt.Errorf("executing custom hooks: %w", err)
	}

	return nil
}

func (m *GitLabHookManager) PreReceiveHook(ctx context.Context, repo *gitalypb.Repository, env []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if gitObjDir, gitAltObjDirs := getEnvVar("GIT_OBJECT_DIRECTORY", env), getEnvVar("GIT_ALTERNATE_OBJECT_DIRECTORIES", env); gitObjDir != "" && gitAltObjDirs != "" {
		repoPath, err := helper.GetRepoPath(repo)
		if err != nil {
			return helper.ErrInternalf("getting repo path: %v", err)
		}

		gitObjectDirRel, gitAltObjectDirRel, err := getRelativeObjectDirs(repoPath, gitObjDir, gitAltObjDirs)
		if err != nil {
			return helper.ErrInternalf("getting relative git object directories: %v", err)
		}

		repo.GitObjectDirectory = gitObjectDirRel
		repo.GitAlternateObjectDirectories = gitAltObjectDirRel
	}

	changes, err := ioutil.ReadAll(stdin)
	if err != nil {
		return helper.ErrInternalf("reading stdin from request: %w", err)
	}

	glID, glRepo, glProtocol := getEnvVar("GL_ID", env), getEnvVar("GL_REPOSITORY", env), getEnvVar("GL_PROTOCOL", env)

	primary, err := isPrimary(env)
	if err != nil {
		return helper.ErrInternalf("could not check role: %w", err)
	}

	// Only the primary should execute hooks and increment reference counters.
	if primary {
		allowed, message, err := m.gitlabAPI.Allowed(repo, glRepo, glID, glProtocol, string(changes))
		if err != nil {
			return fmt.Errorf("GitLab: %v", err)
		}

		if !allowed {
			return errors.New(message)
		}

		if err := m.executeCustomHooks(ctx, stdout, stderr, changes, repo, env); err != nil {
			return err
		}

		// reference counter
		ok, err := m.gitlabAPI.PreReceive(glRepo)
		if err != nil {
			return helper.ErrInternalf("calling pre_receive endpoint: %v", err)
		}

		if !ok {
			return errors.New("")
		}
	}

	hash := sha1.Sum(changes)
	if err := m.VoteOnTransaction(ctx, hash[:], env); err != nil {
		return helper.ErrInternalf("error voting on transaction: %v", err)
	}

	return nil
}
