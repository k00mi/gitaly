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

		{"1.1.GIT", "1.1.1", true},
		{"1.1.GIT", "1.1.0", false},
		{"1.1.GIT", "1.0.0", false},
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
		{"2.24.0", false},
		{"2.25.0", false},
		{"2.29.0-rc0", false},
		{"2.29.0", true},
		{"2.29.1", true},
		{"3.0.0", true},
	} {
		actual, err := git.SupportedVersion(tc.version)
		require.NoError(t, err)
		require.Equal(t, tc.expect, actual)
	}
}
