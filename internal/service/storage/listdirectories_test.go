package storage

import (
	"io"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestListDirectories(t *testing.T) {
	testDir := path.Join(testStorage.Path, t.Name())
	require.NoError(t, os.MkdirAll(path.Dir(testDir), 0755))

	repoPaths := []string{
		"foo/bar1.git",
		"foo/bar2.git",
		"baz/foo/qux3.git",
		"baz/foo/bar1.git",
	}

	for _, p := range repoPaths {
		fullPath := path.Join(testStorage.Path, p)
		require.NoError(t, os.MkdirAll(fullPath, 0755))
		require.NoError(t, exec.Command("git", "init", "--bare", fullPath).Run())
	}
	defer os.RemoveAll(testStorage.Path)

	server, socketPath := runStorageServer(t)
	defer server.Stop()

	client, conn := newStorageClient(t, socketPath)
	defer conn.Close()

	ctx, cancel := testhelper.Context()
	defer cancel()

	testCases := []struct {
		path   string
		depth  uint32
		subset []string
	}{
		{
			path:   "",
			depth:  1,
			subset: []string{"foo", "baz"},
		},
		{
			path:   "",
			depth:  0,
			subset: []string{"foo", "baz"},
		},
		{
			path:   "foo",
			depth:  0,
			subset: []string{"bar1.git", "foo/bar2.git"},
		},
		{
			path:   "",
			depth:  5,
			subset: repoPaths,
		},
	}

	for _, tc := range testCases {
		stream, err := client.ListDirectories(ctx, &pb.ListDirectoriesRequest{StorageName: testStorage.Name, Path: tc.path, Depth: tc.depth})

		var dirs []string
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}

			require.NoError(t, err)
			dirs = append(dirs, resp.GetPaths()...)
		}

		require.NoError(t, err)
		assert.NotContains(t, dirs, "HEAD")
		assert.Subset(t, tc.subset, dirs)
	}
}
