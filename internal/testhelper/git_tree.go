package testhelper

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

type TreeEntry struct {
	Mode    string
	Path    string
	Content string
}

func RequireTree(t testing.TB, repoPath, treeish string, expectedEntries []TreeEntry) {
	t.Helper()

	var actualEntries []TreeEntry

	output := bytes.TrimSpace(MustRunCommand(t, nil, "git", "-C", repoPath, "ls-tree", "-r", treeish))

	if len(output) > 0 {
		for _, line := range bytes.Split(output, []byte("\n")) {
			// Format: <mode> SP <type> SP <object> TAB <file>
			tabSplit := bytes.Split(line, []byte("\t"))
			spaceSplit := bytes.Split(tabSplit[0], []byte(" "))
			path := string(tabSplit[1])
			actualEntries = append(actualEntries, TreeEntry{
				Mode:    string(spaceSplit[0]),
				Path:    path,
				Content: string(MustRunCommand(t, nil, "git", "-C", repoPath, "show", treeish+":"+path)),
			})
		}
	}

	require.Equal(t, expectedEntries, actualEntries)
}
