// +build static,system_libgit2

package main

import (
	"bytes"
	"os/exec"
	"path"
	"strings"
	"testing"
	"time"

	git "github.com/libgit2/git2go/v30"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func merge(t *testing.T, opts mergeSubcommand) (string, string, error) {
	t.Helper()

	valuesByArg := map[string]string{
		"-repository":  opts.repository,
		"-author-name": opts.authorName,
		"-author-mail": opts.authorMail,
		"-author-date": opts.authorDate,
		"-message":     opts.message,
		"-ours":        opts.ours,
		"-theirs":      opts.theirs,
	}

	args := make([]string, 0, len(valuesByArg)*2+1)
	args = append(args, "merge")
	for arg, value := range valuesByArg {
		if value == "" {
			continue
		}
		args = append(args, arg, value)
	}

	binary := path.Join(config.Config.BinDir, "gitaly-git2go")

	ctx, cancel := testhelper.Context()
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd, err := command.New(ctx, exec.Command(binary, args...), nil, &stdout, &stderr)
	require.NoError(t, err)

	err = cmd.Wait()

	return stdout.String(), stderr.String(), err
}

func TestMergeFailsWithMissingArguments(t *testing.T) {
	_, repoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	testcases := []struct {
		desc           string
		opts           mergeSubcommand
		expectedStderr string
	}{
		{
			desc:           "no arguments",
			expectedStderr: "merge: invalid options: missing repository\n",
		},
		{
			desc:           "missing repository",
			opts:           mergeSubcommand{authorName: "Foo", authorMail: "foo@example.com", message: "Foo", ours: "HEAD", theirs: "HEAD"},
			expectedStderr: "merge: invalid options: missing repository\n",
		},
		{
			desc:           "missing author name",
			opts:           mergeSubcommand{repository: repoPath, authorMail: "foo@example.com", message: "Foo", ours: "HEAD", theirs: "HEAD"},
			expectedStderr: "merge: invalid options: missing author name\n",
		},
		{
			desc:           "missing author mail",
			opts:           mergeSubcommand{repository: repoPath, authorName: "Foo", message: "Foo", ours: "HEAD", theirs: "HEAD"},
			expectedStderr: "merge: invalid options: missing author mail\n",
		},
		{
			desc:           "missing message",
			opts:           mergeSubcommand{repository: repoPath, authorName: "Foo", authorMail: "foo@example.com", ours: "HEAD", theirs: "HEAD"},
			expectedStderr: "merge: invalid options: missing message\n",
		},
		{
			desc:           "missing ours",
			opts:           mergeSubcommand{repository: repoPath, authorName: "Foo", authorMail: "foo@example.com", message: "Foo", theirs: "HEAD"},
			expectedStderr: "merge: invalid options: missing ours\n",
		},
		{
			desc:           "missing theirs",
			opts:           mergeSubcommand{repository: repoPath, authorName: "Foo", authorMail: "foo@example.com", message: "Foo", ours: "HEAD"},
			expectedStderr: "merge: invalid options: missing theirs\n",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			stdout, stderr, err := merge(t, tc.opts)
			require.Error(t, err)
			require.Equal(t, "", stdout)
			require.Equal(t, tc.expectedStderr, stderr)
		})
	}
}

func TestMergeFailsWithInvalidRepositoryPath(t *testing.T) {
	stdout, stderr, err := merge(t, mergeSubcommand{
		repository: "/does/not/exist", authorName: "Foo", authorMail: "foo@example.com", message: "Foo", ours: "HEAD", theirs: "HEAD",
	})
	require.Error(t, err)
	require.Equal(t, "", stdout)
	require.Contains(t, stderr, "merge: could not open repository")
}

func buildCommit(t *testing.T, repoPath string, parent *git.Oid, fileContents map[string]string) *git.Oid {
	repo, err := git.OpenRepository(repoPath)
	require.NoError(t, err)

	odb, err := repo.Odb()
	require.NoError(t, err)

	treeBuilder, err := repo.TreeBuilder()
	require.NoError(t, err)

	for file, contents := range fileContents {
		oid, err := odb.Write([]byte(contents), git.ObjectBlob)
		require.NoError(t, err)
		treeBuilder.Insert(file, oid, git.FilemodeBlob)
	}

	tree, err := treeBuilder.Write()
	require.NoError(t, err)

	committer := git.Signature{
		Name:  "Foo",
		Email: "foo@example.com",
		When:  time.Date(2020, 1, 1, 1, 1, 1, 1, time.FixedZone("UTC+2", 2*60*60)),
	}

	var commit *git.Oid
	if parent != nil {
		commit, err = repo.CreateCommitFromIds("", &committer, &committer, "Message", tree, parent)
	} else {
		commit, err = repo.CreateCommitFromIds("", &committer, &committer, "Message", tree)
	}
	require.NoError(t, err)

	return commit
}

func TestMergeTrees(t *testing.T) {
	testcases := []struct {
		desc           string
		base           map[string]string
		ours           map[string]string
		theirs         map[string]string
		expected       map[string]string
		expectedStdout string
		expectedStderr string
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
			expectedStdout: "7d5ae8fb6d2b301c53560bd728004d77778998df\n",
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
			expectedStdout: "348b9b489c3ca128a4555c7a51b20335262519c7\n",
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
			expectedStdout: "e9be4578f89ea52d44936fb36517e837d698b34b\n",
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

		base := buildCommit(t, repoPath, nil, tc.base)
		ours := buildCommit(t, repoPath, base, tc.ours)
		theirs := buildCommit(t, repoPath, base, tc.theirs)

		t.Run(tc.desc, func(t *testing.T) {
			stdout, stderr, err := merge(t, mergeSubcommand{
				repository: repoPath,
				authorName: "John Doe",
				authorMail: "john.doe@example.com",
				authorDate: "Thu Jul 30 07:45:50 2020 +0200",
				message:    "Merge message",
				ours:       ours.String(),
				theirs:     theirs.String(),
			})

			if tc.expectedStderr != "" {
				assert.Error(t, err)
				assert.Equal(t, "", stdout)
				assert.Contains(t, stderr, tc.expectedStderr)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "", stderr)
				assert.Equal(t, tc.expectedStdout, stdout)

				repo, err := git.OpenRepository(repoPath)
				require.NoError(t, err)

				commitOid, err := git.NewOid(strings.TrimSpace(stdout))
				require.NoError(t, err)

				commit, err := repo.LookupCommit(commitOid)
				require.NoError(t, err)

				tree, err := commit.Tree()
				require.NoError(t, err)
				require.Equal(t, uint64(len(tc.expected)), tree.EntryCount())

				for name, contents := range tc.expected {
					entry := tree.EntryByName(name)
					require.NotNil(t, entry)

					blob, err := repo.LookupBlob(entry.Id)
					require.NoError(t, err)
					require.Equal(t, []byte(contents), blob.Contents())
				}
			}
		})
	}
}
