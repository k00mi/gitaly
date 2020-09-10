package git

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

const (
	MasterID = "1e292f8fedd741b75372e19097c76d327140c312"
)

func TestLocalRepository_ContainsRef(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	repo := NewRepository(testhelper.TestRepository())

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

	repo := NewRepository(testhelper.TestRepository())

	testcases := []struct {
		desc     string
		ref      string
		expected Reference
	}{
		{
			desc:     "fully qualified master branch",
			ref:      "refs/heads/master",
			expected: NewReference("refs/heads/master", MasterID),
		},
		{
			desc:     "unqualified master branch fails",
			ref:      "master",
			expected: Reference{},
		},
		{
			desc:     "nonexistent branch",
			ref:      "refs/heads/nonexistent",
			expected: Reference{},
		},
		{
			desc:     "nonexistent branch",
			ref:      "nonexistent",
			expected: Reference{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			ref, err := repo.GetReference(ctx, tc.ref)
			if tc.expected.Name == "" {
				require.True(t, errors.Is(err, ErrReferenceNotFound))
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

	repo := NewRepository(testhelper.TestRepository())

	testcases := []struct {
		desc     string
		ref      string
		expected Reference
	}{
		{
			desc:     "fully qualified master branch",
			ref:      "refs/heads/master",
			expected: NewReference("refs/heads/master", MasterID),
		},
		{
			desc:     "half-qualified master branch",
			ref:      "heads/master",
			expected: NewReference("refs/heads/master", MasterID),
		},
		{
			desc:     "fully qualified master branch",
			ref:      "master",
			expected: NewReference("refs/heads/master", MasterID),
		},
		{
			desc:     "nonexistent branch",
			ref:      "nonexistent",
			expected: Reference{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			ref, err := repo.GetBranch(ctx, tc.ref)
			if tc.expected.Name == "" {
				require.True(t, errors.Is(err, ErrReferenceNotFound))
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

	repo := NewRepository(testhelper.TestRepository())

	testcases := []struct {
		desc    string
		pattern string
		match   func(t *testing.T, refs []Reference)
	}{
		{
			desc:    "master branch",
			pattern: "refs/heads/master",
			match: func(t *testing.T, refs []Reference) {
				require.Equal(t, []Reference{
					NewReference("refs/heads/master", MasterID),
				}, refs)
			},
		},
		{
			desc:    "all references",
			pattern: "",
			match: func(t *testing.T, refs []Reference) {
				require.Len(t, refs, 94)
			},
		},
		{
			desc:    "branches",
			pattern: "refs/heads/",
			match: func(t *testing.T, refs []Reference) {
				require.Len(t, refs, 91)
			},
		},
		{
			desc:    "branches",
			pattern: "refs/heads/nonexistent",
			match: func(t *testing.T, refs []Reference) {
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

func TestLocalRepository_GetBranches(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	repo := NewRepository(testhelper.TestRepository())

	refs, err := repo.GetBranches(ctx)
	require.NoError(t, err)
	require.Len(t, refs, 91)
}
