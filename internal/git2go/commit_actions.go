package git2go

// Action represents an action taken to build a commit.
type Action interface{ action() }

// isAction is used ensuring type safety for actions.
type isAction struct{}

func (isAction) action() {}

// ChangeFileMode sets a file's mode to either regular or executable file.
// FileNotFoundError is returned when attempting to change a non-existent
// file's mode.
type ChangeFileMode struct {
	isAction
	// Path is the path of the whose mode to change.
	Path string
	// ExecutableMode indicates whether the file mode should be changed to executable or not.
	ExecutableMode bool
}

// CreateDirectory creates a directory in the given path with a '.gitkeep' file inside.
// FileExistsError is returned if a file already exists at the provided path.
// DirectoryExistsError is returned if a directory already exists at the provided
// path.
type CreateDirectory struct {
	isAction
	// Path is the path of the directory to create.
	Path string
}

// CreateFile creates a file using the provided path, mode and oid as the blob.
// FileExistsError is returned if a file exists at the given path.
type CreateFile struct {
	isAction
	// Path is the path of the file to create.
	Path string
	// ExecutableMode indicates whether the file mode should be executable or not.
	ExecutableMode bool
	// OID is the id of the object that contains the content of the file.
	OID string
}

// DeleteFile deletes a file or a directory from the provided path.
// FileNotFoundError is returned if the file does not exist.
type DeleteFile struct {
	isAction
	// Path is the path of the file to delete.
	Path string
}

// MoveFile moves a file or a directory to the new path.
// FileNotFoundError is returned if the file does not exist.
type MoveFile struct {
	isAction
	// Path is the path of the file to move.
	Path string
	// NewPath is the new path of the file.
	NewPath string
	// OID is the id of the object that contains the content of the file. If set,
	// the file contents are updated to match the object, otherwise the file keeps
	// the existing content.
	OID string
}

// UpdateFile updates a file at the given path to point to the provided
// OID. FileNotFoundError is returned if the file does not exist.
type UpdateFile struct {
	isAction
	// Path is the path of the file to update.
	Path string
	// OID is the id of the object that contains the new content of the file.
	OID string
}
