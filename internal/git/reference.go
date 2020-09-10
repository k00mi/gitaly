package git

// Reference represents a Git reference.
type Reference struct {
	// Name is the name of the reference
	Name string
	// Target is the target of the reference. For direct references it
	// contains the object ID, for symbolic references it contains the
	// target branch name.
	Target string
	// IsSymbolic tells whether the reference is direct or symbolic
	IsSymbolic bool
}

// NewReference creates a direct reference to an object.
func NewReference(name, target string) Reference {
	return Reference{
		Name:       name,
		Target:     target,
		IsSymbolic: false,
	}
}

// NewSymbolicReference creates a symbolic reference to another reference.
func NewSymbolicReference(name, target string) Reference {
	return Reference{
		Name:       name,
		Target:     target,
		IsSymbolic: true,
	}
}
