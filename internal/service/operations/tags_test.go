package operations

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"

	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestSuccessfulUserDeleteTagRequest(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	server := runOperationServiceServer(t)
	defer server.Stop()

	client, conn := newOperationClient(t)
	defer conn.Close()

	tagNameInput := "to-be-deleted-soon-tag"

	defer exec.Command("git", "-C", testRepoPath, "tag", "-d", tagNameInput).Run()

	user := &pb.User{
		Name:  []byte("Ahmad Sherif"),
		Email: []byte("ahmad@gitlab.com"),
		GlId:  "user-123",
	}

	testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag", tagNameInput)

	request := &pb.UserDeleteTagRequest{
		Repository: testRepo,
		TagName:    []byte(tagNameInput),
		User:       user,
	}

	_, err := client.UserDeleteTag(ctx, request)
	require.NoError(t, err)

	tags := testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag")
	require.NotContains(t, string(tags), tagNameInput, "tag name still exists in tags list")
}

func TestSuccessfulGitHooksForUserDeleteTagRequest(t *testing.T) {
	server := runOperationServiceServer(t)
	defer server.Stop()

	client, conn := newOperationClient(t)
	defer conn.Close()

	tagNameInput := "to-be-deleted-soon-tag"
	defer exec.Command("git", "-C", testRepoPath, "tag", "-d", tagNameInput).Run()

	user := &pb.User{
		Name:  []byte("Ahmad Sherif"),
		Email: []byte("ahmad@gitlab.com"),
		GlId:  "user-123",
	}

	request := &pb.UserDeleteTagRequest{
		Repository: testRepo,
		TagName:    []byte(tagNameInput),
		User:       user,
	}

	for _, hookName := range []string{"pre-receive", "update", "post-receive"} {
		t.Run(hookName, func(t *testing.T) {
			testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag", tagNameInput)

			hookPath, hookOutputTempPath := writeEnvToHook(t, hookName)
			defer os.Remove(hookPath)

			ctx, cancel := testhelper.Context()
			defer cancel()

			_, err := client.UserDeleteTag(ctx, request)
			require.NoError(t, err)

			output := testhelper.MustReadFile(t, hookOutputTempPath)
			require.Contains(t, string(output), "GL_ID=user-123")
		})
	}
}

func TestFailedUserDeleteTagDueToValidation(t *testing.T) {
	server := runOperationServiceServer(t)
	defer server.Stop()

	client, conn := newOperationClient(t)
	defer conn.Close()

	user := &pb.User{
		Name:  []byte("Ahmad Sherif"),
		Email: []byte("ahmad@gitlab.com"),
		GlId:  "user-123",
	}

	testCases := []struct {
		desc    string
		request *pb.UserDeleteTagRequest
		code    codes.Code
	}{
		{
			desc: "empty user",
			request: &pb.UserDeleteTagRequest{
				Repository: testRepo,
				TagName:    []byte("does-matter-the-name-if-user-is-empty"),
			},
			code: codes.InvalidArgument,
		},
		{
			desc: "empty tag name",
			request: &pb.UserDeleteTagRequest{
				Repository: testRepo,
				User:       user,
			},
			code: codes.InvalidArgument,
		},
		{
			desc: "non-existent tag name",
			request: &pb.UserDeleteTagRequest{
				Repository: testRepo,
				User:       user,
				TagName:    []byte("i-do-not-exist"),
			},
			code: codes.Unknown,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			_, err := client.UserDeleteTag(ctx, testCase.request)
			testhelper.AssertGrpcError(t, err, testCase.code, "")
		})
	}
}

func TestFailedUserDeleteTagDueToHooks(t *testing.T) {
	server := runOperationServiceServer(t)
	defer server.Stop()

	client, conn := newOperationClient(t)
	defer conn.Close()

	tagNameInput := "to-be-deleted-soon-tag"
	testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag", tagNameInput)
	defer exec.Command("git", "-C", testRepoPath, "tag", "-d", tagNameInput).Run()

	user := &pb.User{
		Name:  []byte("Ahmad Sherif"),
		Email: []byte("ahmad@gitlab.com"),
		GlId:  "user-123",
	}

	request := &pb.UserDeleteTagRequest{
		Repository: testRepo,
		TagName:    []byte(tagNameInput),
		User:       user,
	}

	hookContent := []byte("#!/bin/false")

	for _, hookName := range []string{"pre-receive", "update"} {
		t.Run(hookName, func(t *testing.T) {
			hookPath := path.Join(testRepoPath, "hooks", hookName)
			ioutil.WriteFile(hookPath, hookContent, 0755)
			defer os.Remove(hookPath)

			ctx, cancel := testhelper.Context()
			defer cancel()

			_, err := client.UserDeleteTag(ctx, request)
			testhelper.AssertGrpcError(t, err, codes.FailedPrecondition, "")

			tags := testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "tag")
			require.Contains(t, string(tags), tagNameInput, "tag name does not exist in tags list")
		})
	}
}
