package git2go

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGit2Go_RevertCommandSerialization(t *testing.T) {
	testcases := []struct {
		desc string
		cmd  RevertCommand
		err  string
	}{
		{
			desc: "missing repository",
			cmd:  RevertCommand{},
			err:  "missing repository",
		},
		{
			desc: "missing author name",
			cmd: RevertCommand{
				Repository: "foo",
			},
			err: "missing author name",
		},
		{
			desc: "missing author mail",
			cmd: RevertCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
			},
			err: "missing author mail",
		},
		{
			desc: "missing author message",
			cmd: RevertCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
			},
			err: "missing message",
		},
		{
			desc: "missing author ours",
			cmd: RevertCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				Message:    "Message",
			},
			err: "missing ours",
		},
		{
			desc: "missing revert",
			cmd: RevertCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				Message:    "Message",
				Ours:       "refs/heads/master",
			},
			err: "missing revert",
		},
		{
			desc: "valid command",
			cmd: RevertCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				Message:    "Message",
				Ours:       "refs/heads/master",
				Revert:     "refs/heads/master",
			},
		},
		{
			desc: "valid command with date",
			cmd: RevertCommand{
				Repository: "foo",
				AuthorName: "Au Thor",
				AuthorMail: "au@thor.com",
				AuthorDate: time.Now().UTC(),
				Message:    "Message",
				Ours:       "refs/heads/master",
				Revert:     "refs/heads/master",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			serialized, err := serialize(tc.cmd)
			require.NoError(t, err)

			deserialized, err := RevertCommandFromSerialized(serialized)

			if tc.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.cmd, deserialized)
			}
		})
	}
}

func TestGit2Go_RevertResultSerialization(t *testing.T) {
	serializeResult := func(t *testing.T, result RevertResult) string {
		t.Helper()
		var buf bytes.Buffer
		err := result.SerializeTo(&buf)
		require.NoError(t, err)
		return buf.String()
	}

	testcases := []struct {
		desc       string
		serialized string
		expected   RevertResult
		err        string
	}{
		{
			desc:       "empty revert result",
			serialized: serializeResult(t, RevertResult{}),
			expected:   RevertResult{},
		},
		{
			desc: "revert result with commit",
			serialized: serializeResult(t, RevertResult{
				CommitID: "1234",
			}),
			expected: RevertResult{
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
			var deserialized RevertResult
			err := deserialize(tc.serialized, &deserialized)

			if tc.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, deserialized)
			}
		})
	}
}
