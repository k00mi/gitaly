package git2go

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGit2Go_MergeCommandSerialization(t *testing.T) {
	testcases := []struct {
		desc string
		cmd  MergeCommand
		err  string
	}{
		{
			desc: "missing repository",
			cmd:  MergeCommand{},
			err:  "missing repository",
		},
		{
			desc: "missing author name",
			cmd: MergeCommand{
				Repository: "foo",
			},
			err: "missing author name",
		},
		{
			desc: "missing author mail",
			cmd: MergeCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
			},
			err: "missing author mail",
		},
		{
			desc: "missing author message",
			cmd: MergeCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
			},
			err: "missing message",
		},
		{
			desc: "missing author ours",
			cmd: MergeCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				Message:    "Message",
			},
			err: "missing ours",
		},
		{
			desc: "missing theirs",
			cmd: MergeCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				Message:    "Message",
				Ours:       "refs/heads/master",
			},
			err: "missing theirs",
		},
		{
			desc: "valid command",
			cmd: MergeCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				Message:    "Message",
				Ours:       "refs/heads/master",
				Theirs:     "refs/heads/foo",
			},
		},
		{
			desc: "valid command with date",
			cmd: MergeCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				AuthorDate: time.Now().UTC(),
				Message:    "Message",
				Ours:       "refs/heads/master",
				Theirs:     "refs/heads/foo",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			serialized, err := serialize(tc.cmd)
			require.NoError(t, err)

			deserialized, err := MergeCommandFromSerialized(serialized)

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

func TestGit2Go_MergeResultSerialization(t *testing.T) {
	serializeResult := func(t *testing.T, result MergeResult) string {
		var buf bytes.Buffer
		err := result.SerializeTo(&buf)
		require.NoError(t, err)
		return buf.String()
	}

	testcases := []struct {
		desc       string
		serialized string
		expected   MergeResult
		err        string
	}{
		{
			desc:       "empty merge result",
			serialized: serializeResult(t, MergeResult{}),
			expected:   MergeResult{},
		},
		{
			desc: "merge result with commit",
			serialized: serializeResult(t, MergeResult{
				CommitID: "1234",
			}),
			expected: MergeResult{
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
			var deserialized MergeResult
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
