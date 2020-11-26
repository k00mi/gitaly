package git2go

// Executor executes gitaly-git2go.
type Executor struct {
	binaryPath    string
	gitBinaryPath string
}

// New returns a new gitaly-git2go executor using the provided binary.
func New(binaryPath, gitBinaryPath string) Executor {
	return Executor{
		binaryPath:    binaryPath,
		gitBinaryPath: gitBinaryPath,
	}
}
