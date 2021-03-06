package git

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/helper/text"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// TestRepository tests an implementation of Repository.
func TestRepository(t *testing.T, getRepository func(testing.TB, *gitalypb.Repository) Repository) {
	for _, tc := range []struct {
		desc string
		test func(*testing.T, func(testing.TB, *gitalypb.Repository) Repository)
	}{
		{
			desc: "ResolveRefish",
			test: testRepositoryResolveRefish,
		},
		{
			desc: "HasBranches",
			test: testRepositoryHasBranches,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			tc.test(t, getRepository)
		})
	}
}

func testRepositoryResolveRefish(t *testing.T, getRepository func(testing.TB, *gitalypb.Repository) Repository) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	pbRepo, _, clean := testhelper.NewTestRepo(t)
	defer clean()

	for _, tc := range []struct {
		desc     string
		refish   string
		expected string
	}{
		{
			desc:     "unqualified master branch",
			refish:   "master",
			expected: "1e292f8fedd741b75372e19097c76d327140c312",
		},
		{
			desc:     "fully qualified master branch",
			refish:   "refs/heads/master",
			expected: "1e292f8fedd741b75372e19097c76d327140c312",
		},
		{
			desc:     "typed commit",
			refish:   "refs/heads/master^{commit}",
			expected: "1e292f8fedd741b75372e19097c76d327140c312",
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
	} {
		t.Run(tc.desc, func(t *testing.T) {
			oid, err := getRepository(t, pbRepo).ResolveRefish(ctx, tc.refish)
			if tc.expected == "" {
				require.Equal(t, err, ErrReferenceNotFound)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, oid)
		})
	}
}

func testRepositoryHasBranches(t *testing.T, getRepository func(testing.TB, *gitalypb.Repository) Repository) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	pbRepo, repoPath, clean := testhelper.InitBareRepo(t)
	defer clean()

	repo := getRepository(t, pbRepo)

	emptyCommit := text.ChompBytes(testhelper.MustRunCommand(t, nil,
		"git", "-C", repoPath, "commit-tree", EmptyTreeID,
	))

	testhelper.MustRunCommand(t, nil,
		"git", "-C", repoPath, "update-ref", "refs/headsbranch", emptyCommit,
	)

	hasBranches, err := repo.HasBranches(ctx)
	require.NoError(t, err)
	require.False(t, hasBranches)

	testhelper.MustRunCommand(t, nil,
		"git", "-C", repoPath, "update-ref", "refs/heads/branch", emptyCommit,
	)

	hasBranches, err = repo.HasBranches(ctx)
	require.NoError(t, err)
	require.True(t, hasBranches)
}
