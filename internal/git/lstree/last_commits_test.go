package lstree

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	testCases := []struct {
		desc     string
		filename string
		entries  Entries
	}{
		{
			desc:     "regular entries",
			filename: "testdata/z-lstree.txt",
			entries: Entries{
				{
					Mode: []byte("100644"),
					Type: 1,
					Oid:  "b78f2bdd90e85de463bd091622efcc70489de893",
					Path: ".gitmodules",
				},
				{
					Mode: []byte("040000"),
					Type: 0,
					Oid:  "85ecfbd13807e6374407ba97d252bfe0cf2403fe",
					Path: "_locales",
				},
				{
					Mode: []byte("160000"),
					Type: 2,
					Oid:  "b2291647b9346873501cedf482270495cd85b7b9",
					Path: "bar",
				},
			},
		},
		{
			desc:     "irregular path",
			filename: "testdata/z-lstree-irregular.txt",
			entries: Entries{
				{
					Mode: []byte("100644"),
					Type: 1,
					Oid:  "b78f2bdd90e85de463bd091622efcc70489de893",
					Path: ".gitmodules",
				},
				{
					Mode: []byte("040000"),
					Type: 0,
					Oid:  "85ecfbd13807e6374407ba97d252bfe0cf2403fe",
					Path: "_locales",
				},
				{
					Mode: []byte("160000"),
					Type: 2,
					Oid:  "b2291647b9346873501cedf482270495cd85b7b9",
					Path: "foo bar",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			file, err := os.Open(testCase.filename)

			require.NoError(t, err)
			defer file.Close()

			parsedEntries := Entries{}

			parser := NewParser(file)
			for {
				entry, err := parser.NextEntry()
				if err == io.EOF {
					break
				}

				require.NoError(t, err)
				parsedEntries = append(parsedEntries, *entry)
			}

			expectedEntries := testCase.entries
			require.Equal(t, len(expectedEntries), len(parsedEntries))

			for index, parsedEntry := range parsedEntries {
				expectedEntry := expectedEntries[index]

				require.Equal(t, expectedEntry.Mode, parsedEntry.Mode)
				require.Equal(t, expectedEntry.Type, parsedEntry.Type)
				require.Equal(t, expectedEntry.Oid, parsedEntry.Oid)
				require.Equal(t, expectedEntry.Path, parsedEntry.Path)
			}
		})
	}
}
