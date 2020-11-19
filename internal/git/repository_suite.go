package git

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
)

// TestRepository tests an implementation of Repository.
func TestRepository(t *testing.T, getRepository func(testing.TB, *gitalypb.Repository) Repository) {
	for _, tc := range []struct {
		desc string
		test func(*testing.T, Repository)
	}{
		{
			desc: "ResolveRefish",
			test: testRepositoryResolveRefish,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			tc.test(t, getRepository(t, testhelper.TestRepository()))
		})
	}
}

func testRepositoryResolveRefish(t *testing.T, repo Repository) {
	ctx, cancel := testhelper.Context()
	defer cancel()

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
			oid, err := repo.ResolveRefish(ctx, tc.refish)
			if tc.expected == "" {
				require.Equal(t, err, ErrReferenceNotFound)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, oid)
		})
	}
}
