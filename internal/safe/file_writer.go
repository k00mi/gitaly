package safe

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/sirupsen/logrus"
)

// FileWriter is a thread safe writer that does an atomic write to the target file. It allows one
// writer at a time to acquire a lock, write the file, and atomically replace the contents of the target file.
type FileWriter struct {
	tmpFile   *os.File
	path      string
	closeErr  error
	closeOnce sync.Once
}

// CreateFileWriter takes path as an absolute path of the target file and creates a new FileWriter by attempting to create a tempfile
func CreateFileWriter(ctx context.Context, path string) (*FileWriter, error) {
	if ctx.Done() == nil {
		return nil, errors.New("context cannot be cancelled")
	}
	var err error
	writer := &FileWriter{path: path}

	directory := filepath.Dir(path)

	tmpFile, err := ioutil.TempFile(directory, filepath.Base(path))
	if err != nil {
		return nil, err
	}

	writer.tmpFile = tmpFile

	go writer.cleanupOnContextDone(ctx)

	return writer, nil
}

func (fw *FileWriter) cleanupOnContextDone(ctx context.Context) {
	<-ctx.Done()
	if err := fw.cleanup(); err != nil {
		logrus.WithField("path", fw.path).WithError(err).Error("error when closing FileWriter")
	}
}

// Write wraps the temporary file's Write.
func (fw *FileWriter) Write(p []byte) (n int, err error) {
	return fw.tmpFile.Write(p)
}

// Commit will close the temporary file and rename it to the target file name the first call to Commit() will close and
// delete the temporary file, so subsequenty calls to Commit() are gauaranteed to return an error.
func (fw *FileWriter) Commit() error {
	if err := fw.tmpFile.Sync(); err != nil {
		return fmt.Errorf("syncing temp file: %v", err)
	}

	if err := fw.close(); err != nil {
		return fmt.Errorf("closing temp file: %v", err)
	}

	if err := fw.rename(); err != nil {
		return fmt.Errorf("renaming temp file: %v", err)
	}

	if err := fw.syncDir(); err != nil {
		return fmt.Errorf("syncing dir: %v", err)
	}

	return nil
}

// rename renames the temporary file to the target file
func (fw *FileWriter) rename() error {
	return os.Rename(fw.tmpFile.Name(), fw.path)
}

// syncDir will sync the directory
func (fw *FileWriter) syncDir() error {
	f, err := os.Open(filepath.Dir(fw.path))
	if err != nil {
		return err
	}
	defer f.Close()

	return f.Sync()
}

// cleanup will close the temporary file and remove it.
func (fw *FileWriter) cleanup() error {
	var err error

	if err = fw.close(); err != nil {
		return err
	}

	if err = os.Remove(fw.tmpFile.Name()); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// close uses sync.Once to guarantee that the file gets closed only once
func (fw *FileWriter) close() error {
	fw.closeOnce.Do(func() {
		fw.closeErr = fw.tmpFile.Close()
	})

	return fw.closeErr
}
