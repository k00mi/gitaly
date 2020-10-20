package git_test

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/client"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper/testserver"
)

const (
	MasterID      = "1e292f8fedd741b75372e19097c76d327140c312"
	NonexistentID = "ba4f184e126b751d1bffad5897f263108befc780"
)

func TestRepository_ResolveRefish(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testcases := []struct {
		desc     string
		refish   string
		expected string
	}{
		{
			desc:     "unqualified master branch",
			refish:   "master",
			expected: MasterID,
		},
		{
			desc:     "fully qualified master branch",
			refish:   "refs/heads/master",
			expected: MasterID,
		},
		{
			desc:     "typed commit",
			refish:   "refs/heads/master^{commit}",
			expected: MasterID,
		},
		{
			desc:     "extended SHA notation",
			refish:   "refs/heads/master^2",
			expected: "c1c67abbaf91f624347bb3ae96eabe3a1b742478",
		},
		{
			desc:   "nonexistent branch",
			refish: "refs/heads/foobar",
		},
		{
			desc:   "SHA notation gone wrong",
			refish: "refs/heads/master^3",
		},
	}

	_, serverSocketPath, cleanup := testserver.RunInternalGitalyServer(t, config.Config.Storages, config.Config.Auth.Token)
	defer cleanup()

	for _, repo := range []git.Repository{
		git.NewRepository(testhelper.TestRepository()),
		func() git.Repository {
			ctx, err := helper.InjectGitalyServers(ctx, "default", serverSocketPath, config.Config.Auth.Token)
			require.NoError(t, err)

			r, err := git.NewRemoteRepository(
				helper.OutgoingToIncoming(ctx),
				testhelper.TestRepository(),
				client.NewPool(),
			)
			require.NoError(t, err)
			return r
		}(),
	} {
		t.Run(fmt.Sprintf("%T", repo), func(t *testing.T) {
			for _, tc := range testcases {
				t.Run(tc.desc, func(t *testing.T) {
					oid, err := repo.ResolveRefish(ctx, tc.refish)

					if tc.expected == "" {
						require.Equal(t, err, git.ErrReferenceNotFound)
						return
					}

					require.NoError(t, err)
					require.Equal(t, tc.expected, oid)
				})
			}
		})
	}
}

func TestLocalRepository_ContainsRef(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	repo := git.NewRepository(testhelper.TestRepository())

	testcases := []struct {
		desc      string
		ref       string
		contained bool
	}{
		{
			desc:      "unqualified master branch",
			ref:       "master",
			contained: true,
		},
		{
			desc:      "fully qualified master branch",
			ref:       "refs/heads/master",
			contained: true,
		},
		{
			desc:      "nonexistent branch",
			ref:       "refs/heads/nonexistent",
			contained: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			contained, err := repo.ContainsRef(ctx, tc.ref)
			require.NoError(t, err)
			require.Equal(t, tc.contained, contained)
		})
	}
}

func TestLocalRepository_GetReference(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	repo := git.NewRepository(testhelper.TestRepository())

	testcases := []struct {
		desc     string
		ref      string
		expected git.Reference
	}{
		{
			desc:     "fully qualified master branch",
			ref:      "refs/heads/master",
			expected: git.NewReference("refs/heads/master", MasterID),
		},
		{
			desc:     "unqualified master branch fails",
			ref:      "master",
			expected: git.Reference{},
		},
		{
			desc:     "nonexistent branch",
			ref:      "refs/heads/nonexistent",
			expected: git.Reference{},
		},
		{
			desc:     "nonexistent branch",
			ref:      "nonexistent",
			expected: git.Reference{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			ref, err := repo.GetReference(ctx, tc.ref)
			if tc.expected.Name == "" {
				require.True(t, errors.Is(err, git.ErrReferenceNotFound))
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, ref)
			}
		})
	}
}

func TestLocalRepository_GetBranch(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	repo := git.NewRepository(testhelper.TestRepository())

	testcases := []struct {
		desc     string
		ref      string
		expected git.Reference
	}{
		{
			desc:     "fully qualified master branch",
			ref:      "refs/heads/master",
			expected: git.NewReference("refs/heads/master", MasterID),
		},
		{
			desc:     "half-qualified master branch",
			ref:      "heads/master",
			expected: git.NewReference("refs/heads/master", MasterID),
		},
		{
			desc:     "fully qualified master branch",
			ref:      "master",
			expected: git.NewReference("refs/heads/master", MasterID),
		},
		{
			desc:     "nonexistent branch",
			ref:      "nonexistent",
			expected: git.Reference{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			ref, err := repo.GetBranch(ctx, tc.ref)
			if tc.expected.Name == "" {
				require.True(t, errors.Is(err, git.ErrReferenceNotFound))
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, ref)
			}
		})
	}
}

func TestLocalRepository_GetReferences(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	repo := git.NewRepository(testhelper.TestRepository())

	testcases := []struct {
		desc    string
		pattern string
		match   func(t *testing.T, refs []git.Reference)
	}{
		{
			desc:    "master branch",
			pattern: "refs/heads/master",
			match: func(t *testing.T, refs []git.Reference) {
				require.Equal(t, []git.Reference{
					git.NewReference("refs/heads/master", MasterID),
				}, refs)
			},
		},
		{
			desc:    "all references",
			pattern: "",
			match: func(t *testing.T, refs []git.Reference) {
				require.Len(t, refs, 94)
			},
		},
		{
			desc:    "branches",
			pattern: "refs/heads/",
			match: func(t *testing.T, refs []git.Reference) {
				require.Len(t, refs, 91)
			},
		},
		{
			desc:    "branches",
			pattern: "refs/heads/nonexistent",
			match: func(t *testing.T, refs []git.Reference) {
				require.Empty(t, refs)
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			refs, err := repo.GetReferences(ctx, tc.pattern)
			require.NoError(t, err)
			tc.match(t, refs)
		})
	}
}

type ReaderFunc func([]byte) (int, error)

func (fn ReaderFunc) Read(b []byte) (int, error) { return fn(b) }

func TestLocalRepository_WriteBlob(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	pbRepo, repoPath, clean := testhelper.InitBareRepo(t)
	defer clean()

	// write attributes file so we can verify WriteBlob runs the files through filters as
	// appropriate
	require.NoError(t, ioutil.WriteFile(filepath.Join(repoPath, "info", "attributes"), []byte(`
crlf binary
lf   text
	`), os.ModePerm))

	repo := git.NewRepository(pbRepo)

	for _, tc := range []struct {
		desc    string
		path    string
		input   io.Reader
		sha     string
		error   error
		content string
	}{
		{
			desc:  "error reading",
			input: ReaderFunc(func([]byte) (int, error) { return 0, assert.AnError }),
			error: fmt.Errorf("%w, stderr: %q", assert.AnError, []byte{}),
		},
		{
			desc:    "successful empty blob",
			input:   strings.NewReader(""),
			sha:     "e69de29bb2d1d6434b8b29ae775ad8c2e48c5391",
			content: "",
		},
		{
			desc:    "successful blob",
			input:   strings.NewReader("some content"),
			sha:     "f0eec86f614944a81f87d879ebdc9a79aea0d7ea",
			content: "some content",
		},
		{
			desc:    "line endings not normalized",
			path:    "crlf",
			input:   strings.NewReader("\r\n"),
			sha:     "d3f5a12faa99758192ecc4ed3fc22c9249232e86",
			content: "\r\n",
		},
		{
			desc:    "line endings normalized",
			path:    "lf",
			input:   strings.NewReader("\r\n"),
			sha:     "8b137891791fe96927ad78e64b0aad7bded08bdc",
			content: "\n",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			sha, err := repo.WriteBlob(ctx, tc.path, tc.input)
			require.Equal(t, tc.error, err)
			if tc.error != nil {
				return
			}

			assert.Equal(t, tc.sha, sha)
			content, err := repo.ReadObject(ctx, sha)
			require.NoError(t, err)
			assert.Equal(t, tc.content, string(content))
		})
	}
}

func TestLocalRepository_ReadObject(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	repo := git.NewRepository(testhelper.TestRepository())

	for _, tc := range []struct {
		desc    string
		oid     string
		content string
		error   error
	}{
		{
			desc:  "invalid object",
			oid:   git.NullSHA,
			error: git.InvalidObjectError(git.NullSHA),
		},
		{
			desc: "valid object",
			// README in gitlab-test
			oid:     "3742e48c1108ced3bf45ac633b34b65ac3f2af04",
			content: "Sample repo for testing gitlab features\n",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			content, err := repo.ReadObject(ctx, tc.oid)
			require.Equal(t, tc.error, err)
			require.Equal(t, tc.content, string(content))
		})
	}
}

func TestLocalRepository_GetBranches(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	repo := git.NewRepository(testhelper.TestRepository())

	refs, err := repo.GetBranches(ctx)
	require.NoError(t, err)
	require.Len(t, refs, 91)
}

func TestLocalRepository_UpdateRef(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	otherRef, err := git.NewRepository(testRepo).GetReference(ctx, "refs/heads/gitaly-test-ref")
	require.NoError(t, err)

	testcases := []struct {
		desc   string
		ref    string
		newrev string
		oldrev string
		verify func(t *testing.T, repo git.Repository, err error)
	}{
		{
			desc:   "successfully update master",
			ref:    "refs/heads/master",
			newrev: otherRef.Target,
			oldrev: MasterID,
			verify: func(t *testing.T, repo git.Repository, err error) {
				require.NoError(t, err)
				ref, err := repo.GetReference(ctx, "refs/heads/master")
				require.NoError(t, err)
				require.Equal(t, ref.Target, otherRef.Target)
			},
		},
		{
			desc:   "update fails with stale oldrev",
			ref:    "refs/heads/master",
			newrev: otherRef.Target,
			oldrev: NonexistentID,
			verify: func(t *testing.T, repo git.Repository, err error) {
				require.Error(t, err)
				ref, err := repo.GetReference(ctx, "refs/heads/master")
				require.NoError(t, err)
				require.Equal(t, ref.Target, MasterID)
			},
		},
		{
			desc:   "update fails with invalid newrev",
			ref:    "refs/heads/master",
			newrev: NonexistentID,
			oldrev: MasterID,
			verify: func(t *testing.T, repo git.Repository, err error) {
				require.Error(t, err)
				ref, err := repo.GetReference(ctx, "refs/heads/master")
				require.NoError(t, err)
				require.Equal(t, ref.Target, MasterID)
			},
		},
		{
			desc:   "successfully update master with empty oldrev",
			ref:    "refs/heads/master",
			newrev: otherRef.Target,
			oldrev: "",
			verify: func(t *testing.T, repo git.Repository, err error) {
				require.NoError(t, err)
				ref, err := repo.GetReference(ctx, "refs/heads/master")
				require.NoError(t, err)
				require.Equal(t, ref.Target, otherRef.Target)
			},
		},
		{
			desc:   "updating unqualified branch fails",
			ref:    "master",
			newrev: otherRef.Target,
			oldrev: MasterID,
			verify: func(t *testing.T, repo git.Repository, err error) {
				require.Error(t, err)
				ref, err := repo.GetReference(ctx, "refs/heads/master")
				require.NoError(t, err)
				require.Equal(t, ref.Target, MasterID)
			},
		},
		{
			desc:   "deleting master succeeds",
			ref:    "refs/heads/master",
			newrev: strings.Repeat("0", 40),
			oldrev: MasterID,
			verify: func(t *testing.T, repo git.Repository, err error) {
				require.NoError(t, err)
				_, err = repo.GetReference(ctx, "refs/heads/master")
				require.Error(t, err)
			},
		},
		{
			desc:   "creating new branch succeeds",
			ref:    "refs/heads/new",
			newrev: MasterID,
			oldrev: strings.Repeat("0", 40),
			verify: func(t *testing.T, repo git.Repository, err error) {
				require.NoError(t, err)
				ref, err := repo.GetReference(ctx, "refs/heads/new")
				require.NoError(t, err)
				require.Equal(t, ref.Target, MasterID)
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			// Re-create repo for each testcase.
			testRepo, _, cleanup := testhelper.NewTestRepo(t)
			defer cleanup()

			repo := git.NewRepository(testRepo)
			err := repo.UpdateRef(ctx, tc.ref, tc.newrev, tc.oldrev)

			tc.verify(t, repo, err)
		})
	}
}

func TestLocalRepository_Config(t *testing.T) {
	repo := git.NewRepository(nil)
	conf := repo.Config()
	require.NotNil(t, conf)
	require.IsType(t, git.RepositoryConfig{}, conf)
}
