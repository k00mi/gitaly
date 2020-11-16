package operations

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/git/log"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper/text"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSuccessfulUserDeleteTagRequest(t *testing.T) {
	testWithFeature(t, featureflag.GoUserDeleteTag, testSuccessfulUserDeleteTagRequest)
}

func testSuccessfulUserDeleteTagRequest(t *testing.T, ctx context.Context) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	tagNameInput := "to-be-deleted-soon-tag"

	defer exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "tag", "-d", tagNameInput).Run()

	testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag", tagNameInput)

	request := &gitalypb.UserDeleteTagRequest{
		Repository: testRepo,
		TagName:    []byte(tagNameInput),
		User:       testhelper.TestUser,
	}

	_, err := client.UserDeleteTag(ctx, request)
	require.NoError(t, err)

	tags := testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag")
	require.NotContains(t, string(tags), tagNameInput, "tag name still exists in tags list")
}

func TestSuccessfulGitHooksForUserDeleteTagRequest(t *testing.T) {
	testWithFeature(t, featureflag.GoUserDeleteTag, testSuccessfulGitHooksForUserDeleteTagRequest)
}

func testSuccessfulGitHooksForUserDeleteTagRequest(t *testing.T, ctx context.Context) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	tagNameInput := "to-be-déleted-soon-tag"
	defer exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "tag", "-d", tagNameInput).Run()

	request := &gitalypb.UserDeleteTagRequest{
		Repository: testRepo,
		TagName:    []byte(tagNameInput),
		User:       testhelper.TestUser,
	}

	for _, hookName := range GitlabHooks {
		t.Run(hookName, func(t *testing.T) {
			testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag", tagNameInput)

			hookOutputTempPath, cleanup := testhelper.WriteEnvToCustomHook(t, testRepoPath, hookName)
			defer cleanup()

			_, err := client.UserDeleteTag(ctx, request)
			require.NoError(t, err)

			output := testhelper.MustReadFile(t, hookOutputTempPath)
			require.Contains(t, string(output), "GL_USERNAME="+testhelper.TestUser.GlUsername)
		})
	}
}

func writeAssertObjectTypePreReceiveHook(t *testing.T) (string, func()) {
	t.Helper()

	hook := fmt.Sprintf(`#!/usr/bin/env ruby

commands = STDIN.each_line.map(&:chomp)
unless commands.size == 1
  abort "expected 1 ref update command, got #{commands.size}"
end

new_value = commands[0].split(' ', 3)[1]
abort 'missing new_value' unless new_value

out = IO.popen(%%W[%s cat-file -t #{new_value}], &:read)
abort 'cat-file failed' unless $?.success?

unless out.chomp == ARGV[0]
  abort "error: expected #{ARGV[0]} object, got #{out}"
end`, config.Config.Git.BinPath)

	dir, cleanup := testhelper.TempDir(t)
	hookPath := filepath.Join(dir, "pre-receive")

	require.NoError(t, ioutil.WriteFile(hookPath, []byte(hook), 0755))

	return hookPath, cleanup
}

func writeAssertObjectTypeUpdateHook(t *testing.T) (string, func()) {
	t.Helper()

	hook := fmt.Sprintf(`#!/usr/bin/env ruby

expected_object_type = ARGV.shift
new_value = ARGV[2]

abort "missing new_value" unless new_value

out = IO.popen(%%W[%s cat-file -t #{new_value}], &:read)
abort 'cat-file failed' unless $?.success?

unless out.chomp == expected_object_type
  abort "error: expected #{expected_object_type} object, got #{out}"
end`, config.Config.Git.BinPath)

	dir, cleanup := testhelper.TempDir(t)
	hookPath := filepath.Join(dir, "pre-receive")

	require.NoError(t, ioutil.WriteFile(hookPath, []byte(hook), 0755))

	return hookPath, cleanup
}

func TestSuccessfulUserCreateTagRequest(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	targetRevision := "c7fbe50c7c7419d9701eebe64b1fdacc3df5b9dd"
	targetRevisionCommit, err := log.GetCommit(ctx, testRepo, targetRevision)
	require.NoError(t, err)

	inputTagName := "to-be-créated-soon"

	preReceiveHook, cleanup := writeAssertObjectTypePreReceiveHook(t)
	defer cleanup()

	updateHook, cleanup := writeAssertObjectTypeUpdateHook(t)
	defer cleanup()

	testCases := []struct {
		desc               string
		tagName            string
		message            string
		targetRevision     string
		expectedTag        *gitalypb.Tag
		expectedObjectType string
	}{
		{
			desc:           "lightweight tag",
			tagName:        inputTagName,
			targetRevision: targetRevision,
			expectedTag: &gitalypb.Tag{
				Name:         []byte(inputTagName),
				TargetCommit: targetRevisionCommit,
			},
			expectedObjectType: "commit",
		},
		{
			desc:           "annotated tag",
			tagName:        inputTagName,
			targetRevision: targetRevision,
			message:        "This is an annotated tag",
			expectedTag: &gitalypb.Tag{
				Name:         []byte(inputTagName),
				TargetCommit: targetRevisionCommit,
				Message:      []byte("This is an annotated tag"),
				MessageSize:  24,
			},
			expectedObjectType: "tag",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			for hook, content := range map[string]string{
				"pre-receive": fmt.Sprintf("#!/bin/sh\n%s %s \"$@\"", preReceiveHook, testCase.expectedObjectType),
				"update":      fmt.Sprintf("#!/bin/sh\n%s %s \"$@\"", updateHook, testCase.expectedObjectType),
			} {
				hookCleanup, err := testhelper.WriteCustomHook(testRepoPath, hook, []byte(content))
				require.NoError(t, err)
				defer hookCleanup()
			}

			request := &gitalypb.UserCreateTagRequest{
				Repository:     testRepo,
				TagName:        []byte(testCase.tagName),
				TargetRevision: []byte(testCase.targetRevision),
				User:           testhelper.TestUser,
				Message:        []byte(testCase.message),
			}

			response, err := client.UserCreateTag(ctx, request)
			require.NoError(t, err, "error from calling RPC")
			require.Empty(t, response.PreReceiveError, "PreReceiveError must be empty, signalling the push was accepted")

			defer exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "tag", "-d", inputTagName).Run()

			id := testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "rev-parse", inputTagName)
			testCase.expectedTag.Id = text.ChompBytes(id)

			require.Equal(t, testCase.expectedTag, response.Tag)

			tag := testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag")
			require.Contains(t, string(tag), inputTagName)
		})
	}
}

// TODO: Rename to TestUserDeleteTag_successfulDeletionOfPrefixedTag,
// see
// https://gitlab.com/gitlab-org/gitaly/-/merge_requests/2839#note_458751929
func TestUserDeleteTagsuccessfulDeletionOfPrefixedTag(t *testing.T) {
	testWithFeature(t, featureflag.GoUserDeleteTag, testUserDeleteTagsuccessfulDeletionOfPrefixedTag)
}

func testUserDeleteTagsuccessfulDeletionOfPrefixedTag(t *testing.T, ctx context.Context) {
	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testCases := []struct {
		desc         string
		tagNameInput string
		tagCommit    string
		user         *gitalypb.User
		response     *gitalypb.UserDeleteTagResponse
		err          error
	}{
		{
			desc:         "possible to delete a tag called refs/tags/something",
			tagNameInput: "refs/tags/can-find-this",
			tagCommit:    "c642fe9b8b9f28f9225d7ea953fe14e74748d53b",
			user:         testhelper.TestUser,
			response:     &gitalypb.UserDeleteTagResponse{},
			err:          nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag", testCase.tagNameInput, testCase.tagCommit)
			defer exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "tag", "-d", testCase.tagNameInput).Run()

			request := &gitalypb.UserDeleteTagRequest{
				Repository: testRepo,
				TagName:    []byte(testCase.tagNameInput),
				User:       testCase.user,
			}

			response, err := client.UserDeleteTag(ctx, request)
			require.Equal(t, testCase.err, err)
			testhelper.ProtoEqual(t, testCase.response, response)

			refs := testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "for-each-ref", "--", "refs/tags/"+testCase.tagNameInput)
			require.NotContains(t, string(refs), testCase.tagCommit, "tag kept because we stripped off refs/tags/*")
		})
	}
}

func TestSuccessfulGitHooksForUserCreateTagRequest(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testSuccessfulGitHooksForUserCreateTagRequest(t, ctx)
}

func testSuccessfulGitHooksForUserCreateTagRequest(t *testing.T, ctx context.Context) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	projectPath := "project/path"
	testRepo.GlProjectPath = projectPath

	tagName := "new-tag"

	request := &gitalypb.UserCreateTagRequest{
		Repository:     testRepo,
		TagName:        []byte(tagName),
		TargetRevision: []byte("c7fbe50c7c7419d9701eebe64b1fdacc3df5b9dd"),
		User:           testhelper.TestUser,
	}

	for _, hookName := range GitlabHooks {
		t.Run(hookName, func(t *testing.T) {
			defer exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "tag", "-d", tagName).Run()

			hookOutputTempPath, cleanup := testhelper.WriteEnvToCustomHook(t, testRepoPath, hookName)
			defer cleanup()

			response, err := client.UserCreateTag(ctx, request)
			require.NoError(t, err)
			require.Empty(t, response.PreReceiveError)

			output := string(testhelper.MustReadFile(t, hookOutputTempPath))
			require.Contains(t, output, "GL_USERNAME="+testhelper.TestUser.GlUsername)
			require.Contains(t, output, "GL_PROJECT_PATH="+projectPath)
		})
	}
}

func TestFailedUserDeleteTagRequestDueToValidation(t *testing.T) {
	testWithFeature(t, featureflag.GoUserDeleteTag, testFailedUserDeleteTagRequestDueToValidation)
}

func testFailedUserDeleteTagRequestDueToValidation(t *testing.T, ctx context.Context) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	testCases := []struct {
		desc     string
		request  *gitalypb.UserDeleteTagRequest
		response *gitalypb.UserDeleteTagResponse
		err      error
	}{
		{
			desc: "empty user",
			request: &gitalypb.UserDeleteTagRequest{
				Repository: testRepo,
				TagName:    []byte("does-matter-the-name-if-user-is-empty"),
			},
			response: nil,
			err:      status.Error(codes.InvalidArgument, "empty user"),
		},
		{
			desc: "empty tag name",
			request: &gitalypb.UserDeleteTagRequest{
				Repository: testRepo,
				User:       testhelper.TestUser,
			},
			response: nil,
			err:      status.Error(codes.InvalidArgument, "empty tag name"),
		},
		{
			desc: "non-existent tag name",
			request: &gitalypb.UserDeleteTagRequest{
				Repository: testRepo,
				User:       testhelper.TestUser,
				TagName:    []byte("i-do-not-exist"),
			},
			response: nil,
			err:      status.Errorf(codes.FailedPrecondition, "tag not found: %s", "i-do-not-exist"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			response, err := client.UserDeleteTag(ctx, testCase.request)
			require.Equal(t, testCase.err, err)
			testhelper.ProtoEqual(t, testCase.response, response)
		})
	}
}

func TestFailedUserDeleteTagDueToHooks(t *testing.T) {
	testWithFeature(t, featureflag.GoUserDeleteTag, testFailedUserDeleteTagDueToHooks)
}

func testFailedUserDeleteTagDueToHooks(t *testing.T, ctx context.Context) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	tagNameInput := "to-be-deleted-soon-tag"
	testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag", tagNameInput)
	defer exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "tag", "-d", tagNameInput).Run()

	request := &gitalypb.UserDeleteTagRequest{
		Repository: testRepo,
		TagName:    []byte(tagNameInput),
		User:       testhelper.TestUser,
	}

	hookContent := []byte("#!/bin/sh\necho GL_ID=$GL_ID\nexit 1")

	for _, hookName := range gitlabPreHooks {
		t.Run(hookName, func(t *testing.T) {
			remove, err := testhelper.WriteCustomHook(testRepoPath, hookName, hookContent)
			require.NoError(t, err)
			defer remove()

			response, err := client.UserDeleteTag(ctx, request)
			require.Nil(t, err)
			require.Contains(t, response.PreReceiveError, "GL_ID="+testhelper.TestUser.GlId)

			tags := testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag")
			require.Contains(t, string(tags), tagNameInput, "tag name does not exist in tags list")
		})
	}
}

func TestFailedUserCreateTagDueToHooks(t *testing.T) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	request := &gitalypb.UserCreateTagRequest{
		Repository:     testRepo,
		TagName:        []byte("new-tag"),
		TargetRevision: []byte("c7fbe50c7c7419d9701eebe64b1fdacc3df5b9dd"),
		User:           testhelper.TestUser,
	}

	hookContent := []byte("#!/bin/sh\necho GL_ID=$GL_ID\nexit 1")

	for _, hookName := range gitlabPreHooks {
		remove, err := testhelper.WriteCustomHook(testRepoPath, hookName, hookContent)
		require.NoError(t, err)
		defer remove()

		ctx, cancel := testhelper.Context()
		defer cancel()

		response, err := client.UserCreateTag(ctx, request)
		require.Nil(t, err)
		require.Contains(t, response.PreReceiveError, "GL_ID="+testhelper.TestUser.GlId)
	}
}

func TestFailedUserCreateTagRequestDueToTagExistence(t *testing.T) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	testCase := struct {
		tagName        string
		targetRevision string
		user           *gitalypb.User
	}{
		tagName:        "v1.1.0",
		targetRevision: "master",
		user:           testhelper.TestUser,
	}

	request := &gitalypb.UserCreateTagRequest{
		Repository:     testRepo,
		TagName:        []byte(testCase.tagName),
		TargetRevision: []byte(testCase.targetRevision),
		User:           testCase.user,
	}

	ctx, cancel := testhelper.Context()
	defer cancel()

	response, err := client.UserCreateTag(ctx, request)
	require.NoError(t, err)
	require.Equal(t, response.Exists, true)
}

func TestFailedUserCreateTagRequestDueToValidation(t *testing.T) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	testCases := []struct {
		desc           string
		tagName        string
		targetRevision string
		user           *gitalypb.User
		code           codes.Code
	}{
		{
			desc:           "empty target revision",
			tagName:        "shiny-new-tag",
			targetRevision: "",
			user:           testhelper.TestUser,
			code:           codes.InvalidArgument,
		},
		{
			desc:           "empty user",
			tagName:        "shiny-new-tag",
			targetRevision: "master",
			user:           nil,
			code:           codes.InvalidArgument,
		},
		{
			desc:           "non-existing starting point",
			tagName:        "new-tag",
			targetRevision: "i-dont-exist",
			user:           testhelper.TestUser,
			code:           codes.FailedPrecondition,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			request := &gitalypb.UserCreateTagRequest{
				Repository:     testRepo,
				TagName:        []byte(testCase.tagName),
				TargetRevision: []byte(testCase.targetRevision),
				User:           testCase.user,
			}

			ctx, cancel := testhelper.Context()
			defer cancel()

			_, err := client.UserCreateTag(ctx, request)
			testhelper.RequireGrpcError(t, err, testCase.code)
		})
	}
}

func TestTagHookOutput(t *testing.T) {
	testWithFeature(t, featureflag.GoUserDeleteTag, testTagHookOutput)
}

func testTagHookOutput(t *testing.T, ctx context.Context) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	testCases := []struct {
		desc        string
		hookContent string
		output      string
	}{
		{
			desc:        "empty stdout and empty stderr",
			hookContent: "#!/bin/sh\nexit 1",
			output:      "",
		},
		{
			desc:        "empty stdout and some stderr",
			hookContent: "#!/bin/sh\necho stderr >&2\nexit 1",
			output:      "stderr\n",
		},
		{
			desc:        "some stdout and empty stderr",
			hookContent: "#!/bin/sh\necho stdout\nexit 1",
			output:      "stdout\n",
		},
		{
			desc:        "some stdout and some stderr",
			hookContent: "#!/bin/sh\necho stdout\necho stderr >&2\nexit 1",
			output:      "stderr\n",
		},
		{
			desc:        "whitespace stdout and some stderr",
			hookContent: "#!/bin/sh\necho '   '\necho stderr >&2\nexit 1",
			output:      "stderr\n",
		},
		{
			desc:        "some stdout and whitespace stderr",
			hookContent: "#!/bin/sh\necho stdout\necho '   ' >&2\nexit 1",
			output:      "stdout\n",
		},
	}

	for _, hookName := range gitlabPreHooks {
		for _, testCase := range testCases {
			t.Run(hookName+"/"+testCase.desc, func(t *testing.T) {
				tagNameInput := "some-tag"
				createRequest := &gitalypb.UserCreateTagRequest{
					Repository:     testRepo,
					TagName:        []byte(tagNameInput),
					TargetRevision: []byte("master"),
					User:           testhelper.TestUser,
				}
				deleteRequest := &gitalypb.UserDeleteTagRequest{
					Repository: testRepo,
					TagName:    []byte(tagNameInput),
					User:       testhelper.TestUser,
				}

				remove, err := testhelper.WriteCustomHook(testRepoPath, hookName, []byte(testCase.hookContent))
				require.NoError(t, err)
				defer remove()

				createResponse, err := client.UserCreateTag(ctx, createRequest)
				require.NoError(t, err)
				require.False(t, createResponse.Exists)
				require.Equal(t, testCase.output, createResponse.PreReceiveError)

				defer exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "tag", "-d", tagNameInput).Run()
				testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag", tagNameInput)

				deleteResponse, err := client.UserDeleteTag(ctx, deleteRequest)
				require.NoError(t, err)

				require.Equal(t, testCase.output, deleteResponse.PreReceiveError)
			})
		}
	}
}
