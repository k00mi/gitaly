package safe_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitaly/internal/safe"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestFile(t *testing.T) {
	dir, cleanup := testhelper.TempDir(t, "", t.Name())
	defer cleanup()

	ctx, cancel := testhelper.Context()
	defer cancel()

	filePath := filepath.Join(dir, "test_file_contents")
	fileContents := "very important contents"
	file, err := safe.CreateFileWriter(ctx, filePath)
	require.NoError(t, err)

	_, err = io.Copy(file, bytes.NewBufferString(fileContents))
	require.NoError(t, err)

	testhelper.AssertFileNotExists(t, filePath)

	require.NoError(t, file.Commit())

	writtenContents, err := ioutil.ReadFile(filePath)
	require.NoError(t, err)
	require.Equal(t, fileContents, string(writtenContents))

	filesInTempDir, err := ioutil.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, filesInTempDir, 1)
	require.Equal(t, filepath.Base(filePath), filesInTempDir[0].Name())
}

func TestFileContextCancelled(t *testing.T) {
	dir, cleanup := testhelper.TempDir(t, "", t.Name())
	defer cleanup()

	ctx, cancel := testhelper.Context()
	defer cancel()

	filePath := filepath.Join(dir, "test_file_contents")
	fileContents := "very important contents"
	file, err := safe.CreateFileWriter(ctx, filePath)
	require.NoError(t, err)

	_, err = io.Copy(file, bytes.NewBufferString(fileContents))
	require.NoError(t, err)

	testhelper.AssertFileNotExists(t, filePath)

	files, err := ioutil.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, files, 1, "expect only the temp file to exist")
	require.Contains(t, files[0].Name(), "test_file_contents", "the one file in the directory should be the temp file")

	cancel()

	testhelper.AssertFileNotExists(t, filePath)

	// wait for the cleanup functions to run
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		filesInTempDir, err := ioutil.ReadDir(dir)
		require.NoError(t, err)
		if len(filesInTempDir) > 0 {
			continue
		}
		return
	}
	t.Error("directory should not have any files left")
}

func TestFileRace(t *testing.T) {
	dir, cleanup := testhelper.TempDir(t, "", t.Name())
	defer cleanup()

	filePath := filepath.Join(dir, "test_file_contents")

	var wg sync.WaitGroup

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			ctx, cancel := testhelper.Context()
			defer cancel()
			w, err := safe.CreateFileWriter(ctx, filePath)
			require.NoError(t, err)
			_, err = w.Write([]byte(fmt.Sprintf("message # %d", i)))
			require.NoError(t, err)
			require.NoError(t, w.Commit())
			wg.Done()
		}(i)
	}
	wg.Wait()

	require.FileExists(t, filePath)
	filesInTempDir, err := ioutil.ReadDir(dir)
	require.NoError(t, err)
	require.Len(t, filesInTempDir, 1, "make sure no other files were written")
}
