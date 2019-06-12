package fstype

import "golang.org/x/sys/unix"

func statFileSystemType(path string) (int64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}

	// This explicit cast to int64 is required for systems where the syscall
	// returns an int32 instead.
	return int64(stat.Type), nil
}
