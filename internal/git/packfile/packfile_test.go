package packfile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	defer testhelper.MustHaveNoChildProcess()
	cleanup := testhelper.Configure()
	defer cleanup()
	return m.Run()
}

func TestList(t *testing.T) {
	tempDir, cleanup := testhelper.TempDir(t)
	defer cleanup()

	repoPath0 := filepath.Join(tempDir, "empty.git")
	testhelper.MustRunCommand(t, nil, "git", "init", "--bare", repoPath0)

	_, repoPath1, cleanup1 := testhelper.NewTestRepo(t)
	defer cleanup1()
	testhelper.MustRunCommand(t, nil, "git", "-C", repoPath1, "repack", "-ad")

	testCases := []struct {
		desc     string
		path     string
		numPacks int
	}{
		{desc: "empty", path: repoPath0},
		{desc: "1 pack no alternates", path: repoPath1, numPacks: 1},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			out, err := List(filepath.Join(tc.path, "objects"))
			require.NoError(t, err)
			require.Len(t, out, tc.numPacks)
		})
	}
}
