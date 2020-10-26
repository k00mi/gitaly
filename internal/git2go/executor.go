package git2go

// Executor executes gitaly-git2go.
type Executor struct {
	binaryPath string
}

// New returns a new gitaly-git2go executor using the provided binary.
func New(binaryPath string) Executor {
	return Executor{binaryPath: binaryPath}
}
