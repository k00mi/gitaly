package git2go

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGit2Go_SubmoduleCommandSerialization(t *testing.T) {
	testcases := []struct {
		desc string
		cmd  SubmoduleCommand
		err  string
	}{
		{
			desc: "missing repository",
			cmd:  SubmoduleCommand{},
			err:  "missing repository",
		},
		{
			desc: "missing author name",
			cmd: SubmoduleCommand{
				Repository: "foo",
			},
			err: "missing author name",
		},
		{
			desc: "missing author mail",
			cmd: SubmoduleCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
			},
			err: "missing author mail",
		},
		{
			desc: "missing commit SHA",
			cmd: SubmoduleCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
			},
			err: "missing commit SHA",
		},
		{
			desc: "missing branch",
			cmd: SubmoduleCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				CommitSHA:  "deadbeef1010",
			},
			err: "missing branch name",
		},
		{
			desc: "missing submodule path",
			cmd: SubmoduleCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				CommitSHA:  "deadbeef1010",
				Branch:     "master",
			},
			err: "missing submodule",
		},
		{
			desc: "valid command",
			cmd: SubmoduleCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				CommitSHA:  "deadbeef1010",
				Branch:     "master",
				Submodule:  "path/to/my/subby",
			},
		},
		{
			desc: "valid command with message",
			cmd: SubmoduleCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				Message:    "meow to you my friend",
				CommitSHA:  "deadbeef1010",
				Branch:     "master",
				Submodule:  "path/to/my/subby",
			},
		},
		{
			desc: "valid command with date",
			cmd: SubmoduleCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				AuthorDate: time.Now().UTC(),
				Message:    "Message",
				CommitSHA:  "deadbeef1010",
				Branch:     "master",
				Submodule:  "path/to/my/subby",
			},
		},
		{
			desc: "valid command with message and date",
			cmd: SubmoduleCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				AuthorDate: time.Now().UTC(),
				Message:    "woof for dayz",
				CommitSHA:  "deadbeef1010",
				Branch:     "master",
				Submodule:  "path/to/my/subby",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			serialized, err := serialize(tc.cmd)
			require.NoError(t, err)

			deserialized, err := SubmoduleCommandFromSerialized(serialized)

			if tc.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, deserialized, tc.cmd)
			}
		})
	}
}

func TestGit2Go_SubmoduleResultSerialization(t *testing.T) {
	serializeResult := func(t *testing.T, result SubmoduleResult) string {
		t.Helper()
		var buf bytes.Buffer
		err := result.SerializeTo(&buf)
		require.NoError(t, err)
		return buf.String()
	}

	testcases := []struct {
		desc       string
		serialized string
		expected   SubmoduleResult
		err        string
	}{
		{
			desc:       "empty merge result",
			serialized: serializeResult(t, SubmoduleResult{}),
			expected:   SubmoduleResult{},
		},
		{
			desc: "merge result with commit",
			serialized: serializeResult(t, SubmoduleResult{
				CommitID: "1234",
			}),
			expected: SubmoduleResult{
				CommitID: "1234",
			},
		},
		{
			desc:       "invalid serialized representation",
			serialized: "xvlc",
			err:        "invalid character",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			var deserialized SubmoduleResult
			err := deserialize(tc.serialized, &deserialized)

			if tc.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, deserialized, tc.expected)
			}
		})
	}
}
