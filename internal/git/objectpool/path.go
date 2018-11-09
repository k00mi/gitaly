package objectpool

import "path/filepath"

// GetRelativePath will create the relative path to the ObjectPool from the
// storage path.
func (o *ObjectPool) GetRelativePath() string {
	return o.relativePath
}

// GetStorageName exposes the shard name, to satisfy the repository.GitRepo
// interface
func (o *ObjectPool) GetStorageName() string {
	return o.storageName
}

// FullPath on disk, depending on the storage path, and the pools relative path
func (o *ObjectPool) FullPath() string {
	return filepath.Join(o.storagePath, o.GetRelativePath())
}
