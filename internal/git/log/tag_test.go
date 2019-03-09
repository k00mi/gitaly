package log

import (
	"fmt"
	"strings"
	"testing"

	"gitlab.com/gitlab-org/gitaly/internal/git/catfile"
	"gitlab.com/gitlab-org/gitaly/internal/helper"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestGetTag(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, testRepoPath, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	testCases := []struct {
		tagName string
		rev     string
		message string
	}{
		{
			tagName: fmt.Sprintf("%s-v1.0.0", t.Name()),
			rev:     "master^^^",
			message: "Prod Release v1.0.0",
		},
		{
			tagName: fmt.Sprintf("%s-v1.0.1", t.Name()),
			rev:     "master^^",
			message: strings.Repeat("a", helper.MaxCommitOrTagMessageSize+1),
		},
	}

	c, err := catfile.New(ctx, testRepo)
	require.NoError(t, err)
	for _, testCase := range testCases {
		t.Run(testCase.tagName, func(t *testing.T) {
			tagID := testhelper.CreateTag(t, testRepoPath, testCase.tagName, testCase.rev, &testhelper.CreateTagOpts{Message: testCase.message})

			tag, err := GetTagCatfile(c, string(tagID), testCase.tagName)
			require.NoError(t, err)
			if len(testCase.message) >= helper.MaxCommitOrTagMessageSize {
				testCase.message = testCase.message[:helper.MaxCommitOrTagMessageSize]
			}

			require.Equal(t, testCase.message, string(tag.Message))
			require.Equal(t, testCase.tagName, string(tag.GetName()))
		})
	}
}
