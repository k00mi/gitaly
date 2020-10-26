package git_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/git"
)

func TestVersionComparator(t *testing.T) {
	for _, tc := range []struct {
		v1, v2 string
		expect bool
	}{
		// v1 < v2 == expect
		{"0.0.0", "0.0.0", false},
		{"0.0.0", "0.0.1", true},
		{"0.0.0", "0.1.0", true},
		{"0.0.0", "0.1.1", true},
		{"0.0.0", "1.0.0", true},
		{"0.0.0", "1.0.1", true},
		{"0.0.0", "1.1.0", true},
		{"0.0.0", "1.1.1", true},

		{"0.0.1", "0.0.0", false},
		{"0.0.1", "0.0.1", false},
		{"0.0.1", "0.1.0", true},
		{"0.0.1", "0.1.1", true},
		{"0.0.1", "1.0.0", true},
		{"0.0.1", "1.0.1", true},
		{"0.0.1", "1.1.0", true},
		{"0.0.1", "1.1.1", true},

		{"0.1.0", "0.0.0", false},
		{"0.1.0", "0.0.1", false},
		{"0.1.0", "0.1.0", false},
		{"0.1.0", "0.1.1", true},
		{"0.1.0", "1.0.0", true},
		{"0.1.0", "1.0.1", true},
		{"0.1.0", "1.1.0", true},
		{"0.1.0", "1.1.1", true},

		{"0.1.1", "0.0.0", false},
		{"0.1.1", "0.0.1", false},
		{"0.1.1", "0.1.0", false},
		{"0.1.1", "0.1.1", false},
		{"0.1.1", "1.0.0", true},
		{"0.1.1", "1.0.1", true},
		{"0.1.1", "1.1.0", true},
		{"0.1.1", "1.1.1", true},

		{"1.0.0", "0.0.0", false},
		{"1.0.0", "0.0.1", false},
		{"1.0.0", "0.1.0", false},
		{"1.0.0", "0.1.1", false},
		{"1.0.0", "1.0.0", false},
		{"1.0.0", "1.0.1", true},
		{"1.0.0", "1.1.0", true},
		{"1.0.0", "1.1.1", true},

		{"1.0.1", "0.0.0", false},
		{"1.0.1", "0.0.1", false},
		{"1.0.1", "0.1.0", false},
		{"1.0.1", "0.1.1", false},
		{"1.0.1", "1.0.0", false},
		{"1.0.1", "1.0.1", false},
		{"1.0.1", "1.1.0", true},
		{"1.0.1", "1.1.1", true},

		{"1.1.0", "0.0.0", false},
		{"1.1.0", "0.0.1", false},
		{"1.1.0", "0.1.0", false},
		{"1.1.0", "0.1.1", false},
		{"1.1.0", "1.0.0", false},
		{"1.1.0", "1.0.1", false},
		{"1.1.0", "1.1.0", false},
		{"1.1.0", "1.1.1", true},

		{"1.1.1", "0.0.0", false},
		{"1.1.1", "0.0.1", false},
		{"1.1.1", "0.1.0", false},
		{"1.1.1", "0.1.1", false},
		{"1.1.1", "1.0.0", false},
		{"1.1.1", "1.0.1", false},
		{"1.1.1", "1.1.0", false},
		{"1.1.1", "1.1.1", false},

		{"1.1.1.rc0", "1.1.1", true},
		{"1.1.1.rc0", "1.1.1.rc0", false},
		{"1.1.1.rc0", "1.1.0", false},
		{"1.1.1-rc0", "1.1.1-rc0", false},
		{"1.1.1-rc0", "1.1.1", true},
		{"1.1.1", "1.1.1-rc0", false},
	} {
		actual, err := git.VersionLessThan(tc.v1, tc.v2)
		require.NoError(t, err)
		require.Equal(t, tc.expect, actual)
	}
}

func TestSupportedVersion(t *testing.T) {
	for _, tc := range []struct {
		version string
		expect  bool
	}{
		{"2.20.0", false},
		{"2.24.0-rc0", false},
		{"2.24.0", true},
		{"2.25.0", true},
		{"3.0.0", true},
	} {
		actual, err := git.SupportedVersion(tc.version)
		require.NoError(t, err)
		require.Equal(t, tc.expect, actual)
	}
}

func TestSupportsReferenceTransactionHook(t *testing.T) {
	for _, tc := range []struct {
		version string
		expect  bool
	}{
		{"2.20.0", false},
		{"2.27.2", false},
		{"2.28.0.rc0", true},
		{"2.28.0.rc2", true},
		{"2.28.1", true},
		{"2.28.0.468.g1be91c4e2f", true},
		{"2.29.0-rc1", true},
		{"3.0.0", true},
	} {
		actual, err := git.SupportsReferenceTransactionHook(tc.version)
		require.NoError(t, err)
		require.Equal(t, tc.expect, actual)
	}
}

func TestSupportsDeltaIslands(t *testing.T) {
	testCases := []struct {
		version string
		fail    bool
		delta   bool
	}{
		{version: "2.20.0", delta: true},
		{version: "2.21.5", delta: true},
		{version: "2.19.8", delta: false},
		{version: "1.20.8", delta: false},
		{version: "1.18.0", delta: false},
		{version: "2.28.0.rc0", delta: true},
		{version: "2.20", fail: true},
		{version: "bla bla", fail: true},
	}

	for _, tc := range testCases {
		t.Run(tc.version, func(t *testing.T) {
			out, err := SupportsDeltaIslands(tc.version)

			if tc.fail {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.delta, out, "delta island support")
		})
	}
}
