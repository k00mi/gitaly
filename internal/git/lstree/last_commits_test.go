package lstree

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	file, err := os.Open("testdata/z-lstree.txt")

	require.NoError(t, err)
	defer file.Close()

	var parsedEntries Entries

	parser := NewParser(file)

	for {
		entry, err := parser.NextEntry()
		if err == io.EOF {
			break
		}

		require.NoError(t, err)
		parsedEntries = append(parsedEntries, entry)
	}

	expectedEntries := Entries{
		{
			Mode:   []byte("100644"),
			Type:   1,
			Object: "b78f2bdd90e85de463bd091622efcc70489de893",
			Path:   ".gitmodules",
		},
		{
			Mode:   []byte("040000"),
			Type:   0,
			Object: "85ecfbd13807e6374407ba97d252bfe0cf2403fe",
			Path:   "_locales",
		},
		{
			Mode:   []byte("160000"),
			Type:   2,
			Object: "b2291647b9346873501cedf482270495cd85b7b9",
			Path:   "bar",
		},
	}

	require.Equal(t, len(expectedEntries), len(parsedEntries))

	for index, parsedEntry := range parsedEntries {
		expectedEntry := expectedEntries[index]

		require.Equal(t, expectedEntry.Mode, parsedEntry.Mode)
		require.Equal(t, expectedEntry.Type, parsedEntry.Type)
		require.Equal(t, expectedEntry.Object, parsedEntry.Object)
		require.Equal(t, expectedEntry.Path, parsedEntry.Path)
	}
}
