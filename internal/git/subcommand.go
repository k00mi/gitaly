package git

const (
	// scReadOnly denotes a read-only command
	scReadOnly = 1 << iota
	// scNoRefUpdates denotes a command which will never update refs
	scNoRefUpdates
)

// subcommands is a curated list of Git command names for special git.SafeCmd
// validation logic
var subcommands = map[string]uint{
	"archive":          scReadOnly,
	"blame":            scReadOnly,
	"bundle":           scReadOnly,
	"cat-file":         scReadOnly,
	"config":           scNoRefUpdates,
	"count-objects":    scReadOnly,
	"diff":             scReadOnly,
	"diff-tree":        scReadOnly,
	"for-each-ref":     scReadOnly,
	"format-patch":     scReadOnly,
	"fsck":             scReadOnly,
	"grep":             scReadOnly,
	"hash-object":      scNoRefUpdates,
	"init":             scNoRefUpdates,
	"log":              scReadOnly,
	"ls-remote":        scReadOnly,
	"ls-tree":          scReadOnly,
	"merge-base":       scReadOnly,
	"multi-pack-index": scNoRefUpdates,
	"pack-refs":        scNoRefUpdates,
	"repack":           scNoRefUpdates,
	"rev-list":         scReadOnly,
	"rev-parse":        scReadOnly,
	"show-ref":         scReadOnly,
	"upload-archive":   scReadOnly,
	"upload-pack":      scReadOnly,
}

// mayUpdateRef indicates if a subcommand is known to update references.
// This is useful to determine if a command requires reference hook
// configuration. A non-exhaustive list of commands is consulted to determine if
// refs are updated. When unknown, true is returned to err on the side of
// caution.
func mayUpdateRef(subcmd string) bool {
	flags, ok := subcommands[subcmd]
	if !ok {
		return true
	}
	return flags&(scReadOnly|scNoRefUpdates) == 0
}
