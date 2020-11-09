// +build static,system_libgit2

package main

import (
	"testing"
	"time"

	git "github.com/libgit2/git2go/v30"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	cmdtesthelper "gitlab.com/gitlab-org/gitaly/cmd/gitaly-git2go/testhelper"
	"gitlab.com/gitlab-org/gitaly/internal/git2go"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestMergeFailsWithMissingArguments(t *testing.T) {
	_, repoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	testcases := []struct {
		desc        string
		request     git2go.MergeCommand
		expectedErr string
	}{
		{
			desc:        "no arguments",
			expectedErr: "merge: invalid parameters: missing repository",
		},
		{
			desc:        "missing repository",
			request:     git2go.MergeCommand{AuthorName: "Foo", AuthorMail: "foo@example.com", Message: "Foo", Ours: "HEAD", Theirs: "HEAD"},
			expectedErr: "merge: invalid parameters: missing repository",
		},
		{
			desc:        "missing author name",
			request:     git2go.MergeCommand{Repository: repoPath, AuthorMail: "foo@example.com", Message: "Foo", Ours: "HEAD", Theirs: "HEAD"},
			expectedErr: "merge: invalid parameters: missing author name",
		},
		{
			desc:        "missing author mail",
			request:     git2go.MergeCommand{Repository: repoPath, AuthorName: "Foo", Message: "Foo", Ours: "HEAD", Theirs: "HEAD"},
			expectedErr: "merge: invalid parameters: missing author mail",
		},
		{
			desc:        "missing message",
			request:     git2go.MergeCommand{Repository: repoPath, AuthorName: "Foo", AuthorMail: "foo@example.com", Ours: "HEAD", Theirs: "HEAD"},
			expectedErr: "merge: invalid parameters: missing message",
		},
		{
			desc:        "missing ours",
			request:     git2go.MergeCommand{Repository: repoPath, AuthorName: "Foo", AuthorMail: "foo@example.com", Message: "Foo", Theirs: "HEAD"},
			expectedErr: "merge: invalid parameters: missing ours",
		},
		{
			desc:        "missing theirs",
			request:     git2go.MergeCommand{Repository: repoPath, AuthorName: "Foo", AuthorMail: "foo@example.com", Message: "Foo", Ours: "HEAD"},
			expectedErr: "merge: invalid parameters: missing theirs",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			_, err := tc.request.Run(ctx, config.Config)
			require.Error(t, err)
			require.Equal(t, tc.expectedErr, err.Error())
		})
	}
}

func TestMergeFailsWithInvalidRepositoryPath(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	_, err := git2go.MergeCommand{
		Repository: "/does/not/exist", AuthorName: "Foo", AuthorMail: "foo@example.com", Message: "Foo", Ours: "HEAD", Theirs: "HEAD",
	}.Run(ctx, config.Config)
	require.Error(t, err)
	require.Contains(t, err.Error(), "merge: could not open repository")
}

func TestMergeTrees(t *testing.T) {
	testcases := []struct {
		desc             string
		base             map[string]string
		ours             map[string]string
		theirs           map[string]string
		expected         map[string]string
		expectedResponse git2go.MergeResult
		expectedStderr   string
	}{
		{
			desc: "trivial merge succeeds",
			base: map[string]string{
				"file": "a",
			},
			ours: map[string]string{
				"file": "a",
			},
			theirs: map[string]string{
				"file": "a",
			},
			expected: map[string]string{
				"file": "a",
			},
			expectedResponse: git2go.MergeResult{
				CommitID: "7d5ae8fb6d2b301c53560bd728004d77778998df",
			},
		},
		{
			desc: "non-trivial merge succeeds",
			base: map[string]string{
				"file": "a\nb\nc\nd\ne\nf\n",
			},
			ours: map[string]string{
				"file": "0\na\nb\nc\nd\ne\nf\n",
			},
			theirs: map[string]string{
				"file": "a\nb\nc\nd\ne\nf\n0\n",
			},
			expected: map[string]string{
				"file": "0\na\nb\nc\nd\ne\nf\n0\n",
			},
			expectedResponse: git2go.MergeResult{
				CommitID: "348b9b489c3ca128a4555c7a51b20335262519c7",
			},
		},
		{
			desc: "multiple files succeed",
			base: map[string]string{
				"1": "foo",
				"2": "bar",
				"3": "qux",
			},
			ours: map[string]string{
				"1": "foo",
				"2": "modified",
				"3": "qux",
			},
			theirs: map[string]string{
				"1": "modified",
				"2": "bar",
				"3": "qux",
			},
			expected: map[string]string{
				"1": "modified",
				"2": "modified",
				"3": "qux",
			},
			expectedResponse: git2go.MergeResult{
				CommitID: "e9be4578f89ea52d44936fb36517e837d698b34b",
			},
		},
		{
			desc: "conflicting merge fails",
			base: map[string]string{
				"1": "foo",
			},
			ours: map[string]string{
				"1": "bar",
			},
			theirs: map[string]string{
				"1": "qux",
			},
			expectedStderr: "merge: could not auto-merge due to conflicts\n",
		},
	}

	for _, tc := range testcases {
		_, repoPath, cleanup := testhelper.NewTestRepo(t)
		defer cleanup()

		base := cmdtesthelper.BuildCommit(t, repoPath, []*git.Oid{nil}, tc.base)
		ours := cmdtesthelper.BuildCommit(t, repoPath, []*git.Oid{base}, tc.ours)
		theirs := cmdtesthelper.BuildCommit(t, repoPath, []*git.Oid{base}, tc.theirs)

		authorDate := time.Date(2020, 7, 30, 7, 45, 50, 0, time.FixedZone("UTC+2", +2*60*60))

		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			response, err := git2go.MergeCommand{
				Repository: repoPath,
				AuthorName: "John Doe",
				AuthorMail: "john.doe@example.com",
				AuthorDate: authorDate,
				Message:    "Merge message",
				Ours:       ours.String(),
				Theirs:     theirs.String(),
			}.Run(ctx, config.Config)

			if tc.expectedStderr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expectedStderr)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expectedResponse, response)

			repo, err := git.OpenRepository(repoPath)
			require.NoError(t, err)
			defer repo.Free()

			commitOid, err := git.NewOid(response.CommitID)
			require.NoError(t, err)

			commit, err := repo.LookupCommit(commitOid)
			require.NoError(t, err)

			tree, err := commit.Tree()
			require.NoError(t, err)
			require.EqualValues(t, len(tc.expected), tree.EntryCount())

			for name, contents := range tc.expected {
				entry := tree.EntryByName(name)
				require.NotNil(t, entry)

				blob, err := repo.LookupBlob(entry.Id)
				require.NoError(t, err)
				require.Equal(t, []byte(contents), blob.Contents())
			}
		})
	}
}
