package testhelper

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/helper/text"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

const (
	GlRepository  = "project-1"
	GlProjectPath = "gitlab-org/gitlab-test"
)

// CreateRepo creates a temporary directory for a repo, without initializing it
func CreateRepo(t testing.TB, storagePath, relativePath string) *gitalypb.Repository {
	repoPath := filepath.Join(storagePath, relativePath, "..")
	require.NoError(t, os.MkdirAll(repoPath, 0755), "making repo parent dir")
	return &gitalypb.Repository{
		StorageName:   "default",
		RelativePath:  relativePath,
		GlRepository:  GlRepository,
		GlProjectPath: GlProjectPath,
	}
}

// InitBareRepo creates a new bare repository
func InitBareRepo(t testing.TB) (*gitalypb.Repository, string, func()) {
	return initRepo(t, true)
}

// InitRepoWithWorktree creates a new repository with a worktree
func InitRepoWithWorktree(t testing.TB) (*gitalypb.Repository, string, func()) {
	return initRepo(t, false)
}

// NewTestObjectPoolName returns a random pool repository name
// in format '@pools/[0-9a-z]{2}/[0-9a-z]{2}/[0-9a-z]{64}.git'.
func NewTestObjectPoolName(t testing.TB) string {
	return filepath.Join("@pools", newDiskHash(t)+".git")
}

// NewRepositoryName returns a random repository hash
// in format '@hashed/[0-9a-f]{2}/[0-9a-f]{2}/[0-9a-f]{64}(.git)?'.
func NewRepositoryName(t testing.TB, bare bool) string {
	suffix := ""
	if bare {
		suffix = ".git"
	}

	return filepath.Join("@hashed", newDiskHash(t)+suffix)
}

// newDiskHash generates a random directory path following the Rails app's
// approach in the hashed storage module, formatted as '[0-9a-f]{2}/[0-9a-f]{2}/[0-9a-f]{64}'.
// https://gitlab.com/gitlab-org/gitlab/-/blob/f5c7d8eb1dd4eee5106123e04dec26d277ff6a83/app/models/storage/hashed.rb#L38-43
func newDiskHash(t testing.TB) string {
	// rails app calculates a sha256 and uses its hex representation
	// as the directory path
	b, err := text.RandomHex(sha256.Size)
	require.NoError(t, err)
	return filepath.Join(b[0:2], b[2:4], b)
}

func initRepo(t testing.TB, bare bool) (*gitalypb.Repository, string, func()) {
	storagePath := GitlabTestStoragePath()
	relativePath := NewRepositoryName(t, bare)
	repoPath := filepath.Join(storagePath, relativePath)

	args := []string{"init"}
	if bare {
		args = append(args, "--bare")
	}

	MustRunCommand(t, nil, "git", append(args, repoPath)...)

	repo := CreateRepo(t, storagePath, relativePath)
	if !bare {
		repo.RelativePath = filepath.Join(repo.RelativePath, ".git")
	}

	return repo, repoPath, func() { require.NoError(t, os.RemoveAll(repoPath)) }
}

// NewTestRepoTo clones a new copy of test repository under a subdirectory in the storage root.
func NewTestRepoTo(t testing.TB, storageRoot, relativePath string) *gitalypb.Repository {
	repo, _, _ := cloneTestRepo(t, storageRoot, relativePath, true)
	return repo
}

// NewTestRepo creates a bare copy of the test repository..
func NewTestRepo(t testing.TB) (repo *gitalypb.Repository, repoPath string, cleanup func()) {
	return cloneTestRepo(t, GitlabTestStoragePath(), NewRepositoryName(t, true), true)
}

// NewTestRepoWithWorktree creates a copy of the test repository with a
// worktree. This is allows you to run normal 'non-bare' Git commands.
func NewTestRepoWithWorktree(t testing.TB) (repo *gitalypb.Repository, repoPath string, cleanup func()) {
	return cloneTestRepo(t, GitlabTestStoragePath(), NewRepositoryName(t, false), false)
}

// testRepositoryPath returns the absolute path of local 'gitlab-org/gitlab-test.git' clone.
// It is cloned under the path by the test preparing step of make.
func testRepositoryPath(t testing.TB) string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("could not get caller info")
	}

	path := filepath.Join(filepath.Dir(currentFile), "..", "..", "_build", "testrepos", "gitlab-test.git")
	if !isValidRepoPath(path) {
		t.Fatalf("local clone of 'gitlab-org/gitlab-test.git' not found in %q, did you run `make prepare-tests`?", path)
	}

	return path
}

// isValidRepoPath checks whether a valid git repository exists at the given path.
func isValidRepoPath(absolutePath string) bool {
	if _, err := os.Stat(filepath.Join(absolutePath, "objects")); err != nil {
		return false
	}

	return true
}

func cloneTestRepo(t testing.TB, storageRoot, relativePath string, bare bool) (repo *gitalypb.Repository, repoPath string, cleanup func()) {
	repoPath = filepath.Join(storageRoot, relativePath)

	repo = CreateRepo(t, storageRoot, relativePath)
	args := []string{"clone", "--no-hardlinks", "--dissociate"}
	if bare {
		args = append(args, "--bare")
	} else {
		// For non-bare repos the relative path is the .git folder inside the path
		repo.RelativePath = filepath.Join(relativePath, ".git")
	}

	MustRunCommand(t, nil, "git", append(args, testRepositoryPath(t), repoPath)...)

	return repo, repoPath, func() { require.NoError(t, os.RemoveAll(repoPath)) }
}

// AddWorktreeArgs returns git command arguments for adding a worktree at the
// specified repo
func AddWorktreeArgs(repoPath, worktreeName string) []string {
	return []string{"-C", repoPath, "worktree", "add", "--detach", worktreeName}
}

// AddWorktree creates a worktree in the repository path for tests
func AddWorktree(t testing.TB, repoPath string, worktreeName string) {
	MustRunCommand(t, nil, "git", AddWorktreeArgs(repoPath, worktreeName)...)
}
