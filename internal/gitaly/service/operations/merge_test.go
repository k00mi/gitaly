package operations

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	gitlog "gitlab.com/gitlab-org/gitaly/internal/git/log"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/helper"
	"gitlab.com/gitlab-org/gitaly/internal/helper/text"
	"gitlab.com/gitlab-org/gitaly/internal/metadata/featureflag"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	commitToMerge         = "e63f41fe459e62e1228fcef60d7189127aeba95a"
	mergeBranchName       = "gitaly-merge-test-branch"
	mergeBranchHeadBefore = "281d3a76f31c812dbf48abce82ccf6860adedd81"
)

func testWithFeature(t *testing.T, feature featureflag.FeatureFlag, testcase func(*testing.T, context.Context)) {
	testhelper.NewFeatureSets([]featureflag.FeatureFlag{
		feature,
	}).Run(t, func(t *testing.T, ctx context.Context) {
		testcase(t, ctx)
	})
}

func TestSuccessfulMerge(t *testing.T) {
	testWithFeature(t, featureflag.GoUserMergeBranch, testSuccessfulMerge)
}

func testSuccessfulMerge(t *testing.T, ctx context.Context) {
	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	locator := config.NewLocator(config.Config)

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	mergeBidi, err := client.UserMergeBranch(ctx)
	require.NoError(t, err)

	prepareMergeBranch(t, testRepoPath)

	hooks := GitlabHooks
	hookTempfiles := make([]string, len(hooks))
	for i, hook := range hooks {
		outputFile, err := ioutil.TempFile("", "")
		require.NoError(t, err)
		require.NoError(t, outputFile.Close())
		defer os.Remove(outputFile.Name())

		script := fmt.Sprintf("#!/bin/sh\n(cat && env) >%s \n", outputFile.Name())
		cleanup, err := testhelper.WriteCustomHook(testRepoPath, hook, []byte(script))
		require.NoError(t, err)
		defer cleanup()

		hookTempfiles[i] = outputFile.Name()
	}

	mergeCommitMessage := "Merged by Gitaly"
	firstRequest := &gitalypb.UserMergeBranchRequest{
		Repository: testRepo,
		User:       testhelper.TestUser,
		CommitId:   commitToMerge,
		Branch:     []byte(mergeBranchName),
		Message:    []byte(mergeCommitMessage),
	}

	require.NoError(t, mergeBidi.Send(firstRequest), "send first request")

	firstResponse, err := mergeBidi.Recv()
	require.NoError(t, err, "receive first response")

	_, err = gitlog.GetCommit(ctx, locator, testRepo, firstResponse.CommitId)
	require.NoError(t, err, "look up git commit before merge is applied")

	require.NoError(t, mergeBidi.Send(&gitalypb.UserMergeBranchRequest{Apply: true}), "apply merge")

	secondResponse, err := mergeBidi.Recv()
	require.NoError(t, err, "receive second response")

	err = testhelper.ReceiveEOFWithTimeout(func() error {
		_, err = mergeBidi.Recv()
		return err
	})
	require.NoError(t, err, "consume EOF")

	commit, err := gitlog.GetCommit(ctx, locator, testRepo, mergeBranchName)
	require.NoError(t, err, "look up git commit after call has finished")

	require.Equal(t, gitalypb.OperationBranchUpdate{CommitId: commit.Id}, *(secondResponse.BranchUpdate))

	require.Equal(t, commit.ParentIds, []string{mergeBranchHeadBefore, commitToMerge})

	require.True(t, strings.HasPrefix(string(commit.Body), mergeCommitMessage), "expected %q to start with %q", commit.Body, mergeCommitMessage)

	author := commit.Author
	require.Equal(t, testhelper.TestUser.Name, author.Name)
	require.Equal(t, testhelper.TestUser.Email, author.Email)

	expectedGlID := "GL_ID=" + testhelper.TestUser.GlId
	for i, h := range hooks {
		hookEnv, err := ioutil.ReadFile(hookTempfiles[i])
		require.NoError(t, err)

		lines := strings.Split(string(hookEnv), "\n")
		require.Contains(t, lines, expectedGlID, "expected env of hook %q to contain %q", h, expectedGlID)
		require.Contains(t, lines, "GL_PROTOCOL=web", "expected env of hook %q to contain GL_PROTOCOL")

		if h == "pre-receive" || h == "post-receive" {
			require.Regexp(t, mergeBranchHeadBefore+" .* refs/heads/"+mergeBranchName, lines[0], "expected env of hook %q to contain reference change", h)
		}
	}
}

func TestAbortedMerge(t *testing.T) {
	testWithFeature(t, featureflag.GoUserMergeBranch, testAbortedMerge)
}

func testAbortedMerge(t *testing.T, ctx context.Context) {
	locator := config.NewLocator(config.Config)

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	prepareMergeBranch(t, testRepoPath)

	firstRequest := &gitalypb.UserMergeBranchRequest{
		Repository: testRepo,
		User:       testhelper.TestUser,
		CommitId:   commitToMerge,
		Branch:     []byte(mergeBranchName),
		Message:    []byte("foobar"),
	}

	testCases := []struct {
		req       *gitalypb.UserMergeBranchRequest
		closeSend bool
		desc      string
	}{
		{req: &gitalypb.UserMergeBranchRequest{}, desc: "empty request, don't close"},
		{req: &gitalypb.UserMergeBranchRequest{}, closeSend: true, desc: "empty request and close"},
		{closeSend: true, desc: "no request just close"},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			mergeBidi, err := client.UserMergeBranch(ctx)
			require.NoError(t, err)

			require.NoError(t, mergeBidi.Send(firstRequest), "send first request")

			firstResponse, err := mergeBidi.Recv()
			require.NoError(t, err, "first response")
			require.NotEqual(t, "", firstResponse.CommitId, "commit ID on first response")

			if tc.req != nil {
				require.NoError(t, mergeBidi.Send(tc.req), "send second request")
			}

			if tc.closeSend {
				require.NoError(t, mergeBidi.CloseSend(), "close request stream from client")
			}

			secondResponse, err := recvTimeout(mergeBidi, 1*time.Second)
			if err == errRecvTimeout {
				t.Fatal(err)
			}

			require.Equal(t, "", secondResponse.GetBranchUpdate().GetCommitId(), "merge should not have been applied")
			require.Error(t, err)

			commit, err := gitlog.GetCommit(ctx, locator, testRepo, mergeBranchName)
			require.NoError(t, err, "look up git commit after call has finished")

			require.Equal(t, mergeBranchHeadBefore, commit.Id, "branch should not change when the merge is aborted")
		})
	}
}

func TestFailedMergeConcurrentUpdate(t *testing.T) {
	testWithFeature(t, featureflag.GoUserMergeBranch, testFailedMergeConcurrentUpdate)
}

func testFailedMergeConcurrentUpdate(t *testing.T, ctx context.Context) {
	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	locator := config.NewLocator(config.Config)

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	mergeBidi, err := client.UserMergeBranch(ctx)
	require.NoError(t, err)

	prepareMergeBranch(t, testRepoPath)

	mergeCommitMessage := "Merged by Gitaly"
	firstRequest := &gitalypb.UserMergeBranchRequest{
		Repository: testRepo,
		User:       testhelper.TestUser,
		CommitId:   commitToMerge,
		Branch:     []byte(mergeBranchName),
		Message:    []byte(mergeCommitMessage),
	}

	require.NoError(t, mergeBidi.Send(firstRequest), "send first request")
	firstResponse, err := mergeBidi.Recv()
	require.NoError(t, err, "receive first response")

	// This concurrent update of the branch we are merging into should make the merge fail.
	concurrentCommitID := testhelper.CreateCommit(t, testRepoPath, mergeBranchName, nil)
	require.NotEqual(t, firstResponse.CommitId, concurrentCommitID)

	require.NoError(t, mergeBidi.Send(&gitalypb.UserMergeBranchRequest{Apply: true}), "apply merge")
	require.NoError(t, mergeBidi.CloseSend(), "close send")

	secondResponse, err := mergeBidi.Recv()
	require.NoError(t, err, "receive second response")
	testhelper.ProtoEqual(t, secondResponse, &gitalypb.UserMergeBranchResponse{})

	commit, err := gitlog.GetCommit(ctx, locator, testRepo, mergeBranchName)
	require.NoError(t, err, "get commit after RPC finished")
	require.Equal(t, commit.Id, concurrentCommitID, "RPC should not have trampled concurrent update")
}

func TestUserMergeBranch_ambiguousReference(t *testing.T) {
	testWithFeature(t, featureflag.GoUserMergeBranch, testUserMergeBranchAmbiguousReference)
}

func testUserMergeBranchAmbiguousReference(t *testing.T, ctx context.Context) {
	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	merge, err := client.UserMergeBranch(ctx)
	require.NoError(t, err)

	prepareMergeBranch(t, testRepoPath)

	repo := git.NewRepository(testRepo)

	master, err := repo.GetReference(ctx, "refs/heads/master")
	require.NoError(t, err)

	// We're now creating all kinds of potentially ambiguous references in
	// the hope that UserMergeBranch won't be confused by it.
	for _, reference := range []string{
		mergeBranchName,
		"heads/" + mergeBranchName,
		"refs/heads/refs/heads/" + mergeBranchName,
		"refs/tags/" + mergeBranchName,
		"refs/tags/heads/" + mergeBranchName,
		"refs/tags/refs/heads/" + mergeBranchName,
	} {
		require.NoError(t, repo.UpdateRef(ctx, reference, master.Target, git.NullSHA))
	}

	mergeCommitMessage := "Merged by Gitaly"
	firstRequest := &gitalypb.UserMergeBranchRequest{
		Repository: testRepo,
		User:       testhelper.TestUser,
		CommitId:   commitToMerge,
		Branch:     []byte(mergeBranchName),
		Message:    []byte(mergeCommitMessage),
	}

	require.NoError(t, merge.Send(firstRequest), "send first request")

	_, err = merge.Recv()
	require.NoError(t, err, "receive first response")
	require.NoError(t, err, "look up git commit before merge is applied")
	require.NoError(t, merge.Send(&gitalypb.UserMergeBranchRequest{Apply: true}), "apply merge")

	response, err := merge.Recv()
	require.NoError(t, err, "receive second response")
	require.NoError(t, testhelper.ReceiveEOFWithTimeout(func() error {
		_, err = merge.Recv()
		return err
	}))

	locator := config.NewLocator(config.Config)

	commit, err := gitlog.GetCommit(ctx, locator, testRepo, "refs/heads/"+mergeBranchName)
	require.NoError(t, err, "look up git commit after call has finished")

	tag, err := gitlog.GetCommit(ctx, locator, testRepo, "refs/tags/"+mergeBranchName)
	require.NoError(t, err, "look up git tag after call has finished")

	require.Equal(t, gitalypb.OperationBranchUpdate{CommitId: commit.Id}, *(response.BranchUpdate))
	require.Equal(t, mergeCommitMessage, string(commit.Body))
	require.Equal(t, testhelper.TestUser.Name, commit.Author.Name)
	require.Equal(t, testhelper.TestUser.Email, commit.Author.Email)

	if featureflag.IsEnabled(helper.OutgoingToIncoming(ctx), featureflag.GoUserMergeBranch) {
		require.Equal(t, []string{mergeBranchHeadBefore, commitToMerge}, commit.ParentIds)
	} else {
		// The Ruby implementation performs a mismerge with the
		// ambiguous tag instead of with the branch name.
		require.Equal(t, []string{tag.Id, commitToMerge}, commit.ParentIds)
	}
}

func TestFailedMergeDueToHooks(t *testing.T) {
	testWithFeature(t, featureflag.GoUserMergeBranch, testFailedMergeDueToHooks)
}

func testFailedMergeDueToHooks(t *testing.T, ctx context.Context) {
	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	prepareMergeBranch(t, testRepoPath)

	hookContent := []byte("#!/bin/sh\necho 'failure'\nexit 1")

	for _, hookName := range gitlabPreHooks {
		t.Run(hookName, func(t *testing.T) {
			remove, err := testhelper.WriteCustomHook(testRepoPath, hookName, hookContent)
			require.NoError(t, err)
			defer remove()

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			mergeBidi, err := client.UserMergeBranch(ctx)
			require.NoError(t, err)

			mergeCommitMessage := "Merged by Gitaly"
			firstRequest := &gitalypb.UserMergeBranchRequest{
				Repository: testRepo,
				User:       testhelper.TestUser,
				CommitId:   commitToMerge,
				Branch:     []byte(mergeBranchName),
				Message:    []byte(mergeCommitMessage),
			}

			require.NoError(t, mergeBidi.Send(firstRequest), "send first request")

			_, err = mergeBidi.Recv()
			require.NoError(t, err, "receive first response")

			require.NoError(t, mergeBidi.Send(&gitalypb.UserMergeBranchRequest{Apply: true}), "apply merge")
			require.NoError(t, mergeBidi.CloseSend(), "close send")

			secondResponse, err := mergeBidi.Recv()
			require.NoError(t, err, "receive second response")
			require.Contains(t, secondResponse.PreReceiveError, "failure")

			err = testhelper.ReceiveEOFWithTimeout(func() error {
				_, err = mergeBidi.Recv()
				return err
			})
			require.NoError(t, err, "consume EOF")

			currentBranchHead := testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "rev-parse", mergeBranchName)
			require.Equal(t, mergeBranchHeadBefore, text.ChompBytes(currentBranchHead), "branch head updated")
		})
	}
}

func TestSuccessfulUserFFBranchRequest(t *testing.T) {
	testhelper.NewFeatureSets([]featureflag.FeatureFlag{
		featureflag.GoUserFFBranch,
	}).Run(t, func(t *testing.T, ctx context.Context) {
		serverSocketPath, stop := runOperationServiceServer(t)
		defer stop()

		client, conn := newOperationClient(t, serverSocketPath)
		defer conn.Close()

		testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
		defer cleanupFn()

		commitID := "cfe32cf61b73a0d5e9f13e774abde7ff789b1660"
		branchName := "test-ff-target-branch"
		request := &gitalypb.UserFFBranchRequest{
			Repository: testRepo,
			CommitId:   commitID,
			Branch:     []byte(branchName),
			User:       testhelper.TestUser,
		}
		expectedResponse := &gitalypb.UserFFBranchResponse{
			BranchUpdate: &gitalypb.OperationBranchUpdate{
				RepoCreated:   false,
				BranchCreated: false,
				CommitId:      commitID,
			},
		}

		testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "branch", "-f", branchName, "6d394385cf567f80a8fd85055db1ab4c5295806f")
		defer exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "branch", "-d", branchName).Run()

		resp, err := client.UserFFBranch(ctx, request)
		require.NoError(t, err)
		testhelper.ProtoEqual(t, expectedResponse, resp)
		newBranchHead := testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "rev-parse", branchName)
		require.Equal(t, commitID, text.ChompBytes(newBranchHead), "branch head not updated")
	})
}

func TestFailedUserFFBranchRequest(t *testing.T) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	commitID := "cfe32cf61b73a0d5e9f13e774abde7ff789b1660"
	branchName := "test-ff-target-branch"

	testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "branch", "-f", branchName, "6d394385cf567f80a8fd85055db1ab4c5295806f")
	defer exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "branch", "-d", branchName).Run()

	testCases := []struct {
		desc     string
		user     *gitalypb.User
		branch   []byte
		commitID string
		repo     *gitalypb.Repository
		code     codes.Code
	}{
		{
			desc:     "empty repository",
			user:     testhelper.TestUser,
			branch:   []byte(branchName),
			commitID: commitID,
			code:     codes.InvalidArgument,
		},
		{
			desc:     "empty user",
			repo:     testRepo,
			branch:   []byte(branchName),
			commitID: commitID,
			code:     codes.InvalidArgument,
		},
		{
			desc:   "empty commit",
			repo:   testRepo,
			user:   testhelper.TestUser,
			branch: []byte(branchName),
			code:   codes.InvalidArgument,
		},
		{
			desc:     "non-existing commit",
			repo:     testRepo,
			user:     testhelper.TestUser,
			branch:   []byte(branchName),
			commitID: "f001",
			code:     codes.InvalidArgument,
		},
		{
			desc:     "empty branch",
			repo:     testRepo,
			user:     testhelper.TestUser,
			commitID: commitID,
			code:     codes.InvalidArgument,
		},
		{
			desc:     "non-existing branch",
			repo:     testRepo,
			user:     testhelper.TestUser,
			branch:   []byte("this-isnt-real"),
			commitID: commitID,
			code:     codes.InvalidArgument,
		},
		{
			desc:     "commit is not a descendant of branch head",
			repo:     testRepo,
			user:     testhelper.TestUser,
			branch:   []byte(branchName),
			commitID: "1a0b36b3cdad1d2ee32457c102a8c0b7056fa863",
			code:     codes.FailedPrecondition,
		},
	}

	testhelper.NewFeatureSets([]featureflag.FeatureFlag{
		featureflag.GoUserFFBranch,
	}).Run(t, func(t *testing.T, ctx context.Context) {
		for _, testCase := range testCases {
			t.Run(testCase.desc, func(t *testing.T) {
				request := &gitalypb.UserFFBranchRequest{
					Repository: testCase.repo,
					User:       testCase.user,
					Branch:     testCase.branch,
					CommitId:   testCase.commitID,
				}
				_, err := client.UserFFBranch(ctx, request)
				testhelper.RequireGrpcError(t, err, testCase.code)
			})
		}
	})
}

func TestFailedUserFFBranchDueToHooks(t *testing.T) {
	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	commitID := "cfe32cf61b73a0d5e9f13e774abde7ff789b1660"
	branchName := "test-ff-target-branch"
	request := &gitalypb.UserFFBranchRequest{
		Repository: testRepo,
		CommitId:   commitID,
		Branch:     []byte(branchName),
		User:       testhelper.TestUser,
	}

	testhelper.MustRunCommand(t, nil, "git", "-C", testRepoPath, "branch", "-f", branchName, "6d394385cf567f80a8fd85055db1ab4c5295806f")
	defer exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "branch", "-d", branchName).Run()

	hookContent := []byte("#!/bin/sh\necho 'failure'\nexit 1")

	featureSets := testhelper.NewFeatureSets([]featureflag.FeatureFlag{
		featureflag.GoUserFFBranch,
	})

	for _, featureSet := range featureSets {
		t.Run(featureSet.Desc(), func(t *testing.T) {
			for _, hookName := range gitlabPreHooks {
				t.Run(hookName, func(t *testing.T) {
					remove, err := testhelper.WriteCustomHook(testRepoPath, hookName, hookContent)
					require.NoError(t, err)
					defer remove()

					ctx, cancel := testhelper.Context()
					defer cancel()

					ctx = featureSet.Disable(ctx)

					resp, err := client.UserFFBranch(ctx, request)
					require.Nil(t, err)
					require.Contains(t, resp.PreReceiveError, "failure")
				})
			}
		})
	}
}

func TestSuccessfulUserMergeToRefRequest(t *testing.T) {
	ctx, cleanup := testhelper.Context()
	defer cleanup()

	locator := config.NewLocator(config.Config)

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	prepareMergeBranch(t, testRepoPath)

	existingTargetRef := []byte("refs/merge-requests/x/written")
	emptyTargetRef := []byte("refs/merge-requests/x/merge")
	mergeCommitMessage := "Merged by Gitaly"

	// Writes in existingTargetRef
	beforeRefreshCommitSha := "a5391128b0ef5d21df5dd23d98557f4ef12fae20"
	out, err := exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "update-ref", string(existingTargetRef), beforeRefreshCommitSha).CombinedOutput()
	require.NoError(t, err, "give an existing state to the target ref: %s", out)

	testCases := []struct {
		desc           string
		user           *gitalypb.User
		branch         []byte
		targetRef      []byte
		emptyRef       bool
		sourceSha      string
		message        string
		firstParentRef []byte
	}{
		{
			desc:           "empty target ref merge",
			user:           testhelper.TestUser,
			targetRef:      emptyTargetRef,
			emptyRef:       true,
			sourceSha:      commitToMerge,
			message:        mergeCommitMessage,
			firstParentRef: []byte("refs/heads/" + mergeBranchName),
		},
		{
			desc:           "existing target ref",
			user:           testhelper.TestUser,
			targetRef:      existingTargetRef,
			emptyRef:       false,
			sourceSha:      commitToMerge,
			message:        mergeCommitMessage,
			firstParentRef: []byte("refs/heads/" + mergeBranchName),
		},
		{
			desc:      "branch is specified and firstParentRef is empty",
			user:      testhelper.TestUser,
			branch:    []byte(mergeBranchName),
			targetRef: existingTargetRef,
			emptyRef:  false,
			sourceSha: "38008cb17ce1466d8fec2dfa6f6ab8dcfe5cf49e",
			message:   mergeCommitMessage,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			request := &gitalypb.UserMergeToRefRequest{
				Repository:     testRepo,
				User:           testCase.user,
				Branch:         testCase.branch,
				TargetRef:      testCase.targetRef,
				SourceSha:      testCase.sourceSha,
				Message:        []byte(testCase.message),
				FirstParentRef: testCase.firstParentRef,
			}

			commitBeforeRefMerge, fetchRefBeforeMergeErr := gitlog.GetCommit(ctx, locator, testRepo, string(testCase.targetRef))
			if testCase.emptyRef {
				require.Error(t, fetchRefBeforeMergeErr, "error when fetching empty ref commit")
			} else {
				require.NoError(t, fetchRefBeforeMergeErr, "no error when fetching existing ref")
			}

			resp, err := client.UserMergeToRef(ctx, request)
			require.NoError(t, err)

			commit, err := gitlog.GetCommit(ctx, locator, testRepo, string(testCase.targetRef))
			require.NoError(t, err, "look up git commit after call has finished")

			// Asserts commit parent SHAs
			require.Equal(t, []string{mergeBranchHeadBefore, testCase.sourceSha}, commit.ParentIds, "merge commit parents must be the sha before HEAD and source sha")

			require.True(t, strings.HasPrefix(string(commit.Body), testCase.message), "expected %q to start with %q", commit.Body, testCase.message)

			// Asserts author
			author := commit.Author
			require.Equal(t, testhelper.TestUser.Name, author.Name)
			require.Equal(t, testhelper.TestUser.Email, author.Email)

			require.Equal(t, resp.CommitId, commit.Id)

			// Calling commitBeforeRefMerge.Id in a non-existent
			// commit will raise a null-pointer error.
			if !testCase.emptyRef {
				require.NotEqual(t, commit.Id, commitBeforeRefMerge.Id)
			}
		})
	}
}

func TestConflictsOnUserMergeToRefRequest(t *testing.T) {
	ctx, cleanup := testhelper.Context()
	defer cleanup()

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	prepareMergeBranchWithHead(t, testRepoPath, "824be604a34828eb682305f0d963056cfac87b2d")

	request := &gitalypb.UserMergeToRefRequest{
		Repository:     testRepo,
		User:           testhelper.TestUser,
		TargetRef:      []byte("refs/merge-requests/x/written"),
		SourceSha:      "1450cd639e0bc6721eb02800169e464f212cde06",
		Message:        []byte("message1"),
		FirstParentRef: []byte("refs/heads/" + mergeBranchName),
	}

	t.Run("allow conflicts to be merged with markers", func(t *testing.T) {
		request.AllowConflicts = true

		resp, err := client.UserMergeToRef(ctx, request)
		require.NoError(t, err)

		var buf bytes.Buffer
		cmd := exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "show", resp.CommitId)
		cmd.Stdout = &buf
		require.NoError(t, cmd.Run())

		bufStr := buf.String()
		require.Contains(t, bufStr, "+<<<<<<< files/ruby/popen.rb")
		require.Contains(t, bufStr, "+>>>>>>> files/ruby/popen.rb")
		require.Contains(t, bufStr, "+<<<<<<< files/ruby/regex.rb")
		require.Contains(t, bufStr, "+>>>>>>> files/ruby/regex.rb")
	})

	t.Run("disallow conflicts to be merged", func(t *testing.T) {
		request.AllowConflicts = false

		_, err := client.UserMergeToRef(ctx, request)
		require.Equal(t, status.Error(codes.FailedPrecondition, "Failed to create merge commit for source_sha 1450cd639e0bc6721eb02800169e464f212cde06 and target_sha 824be604a34828eb682305f0d963056cfac87b2d at refs/merge-requests/x/written"), err)
	})
}

func TestFailedUserMergeToRefRequest(t *testing.T) {
	ctx, cleanup := testhelper.Context()
	defer cleanup()

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	prepareMergeBranch(t, testRepoPath)

	validTargetRef := []byte("refs/merge-requests/x/merge")

	testCases := []struct {
		desc      string
		user      *gitalypb.User
		branch    []byte
		targetRef []byte
		sourceSha string
		repo      *gitalypb.Repository
		code      codes.Code
	}{
		{
			desc:      "empty repository",
			user:      testhelper.TestUser,
			branch:    []byte(branchName),
			sourceSha: commitToMerge,
			targetRef: validTargetRef,
			code:      codes.InvalidArgument,
		},
		{
			desc:      "empty user",
			repo:      testRepo,
			branch:    []byte(branchName),
			sourceSha: commitToMerge,
			targetRef: validTargetRef,
			code:      codes.InvalidArgument,
		},
		{
			desc:      "empty source SHA",
			repo:      testRepo,
			user:      testhelper.TestUser,
			branch:    []byte(branchName),
			targetRef: validTargetRef,
			code:      codes.InvalidArgument,
		},
		{
			desc:      "non-existing commit",
			repo:      testRepo,
			user:      testhelper.TestUser,
			branch:    []byte(branchName),
			sourceSha: "f001",
			targetRef: validTargetRef,
			code:      codes.InvalidArgument,
		},
		{
			desc:      "empty branch and first parent ref",
			repo:      testRepo,
			user:      testhelper.TestUser,
			sourceSha: commitToMerge,
			targetRef: validTargetRef,
			code:      codes.InvalidArgument,
		},
		{
			desc:      "invalid target ref",
			repo:      testRepo,
			user:      testhelper.TestUser,
			branch:    []byte(branchName),
			sourceSha: commitToMerge,
			targetRef: []byte("refs/heads/branch"),
			code:      codes.InvalidArgument,
		},
		{
			desc:      "non-existing branch",
			repo:      testRepo,
			user:      testhelper.TestUser,
			branch:    []byte("this-isnt-real"),
			sourceSha: commitToMerge,
			targetRef: validTargetRef,
			code:      codes.InvalidArgument,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			request := &gitalypb.UserMergeToRefRequest{
				Repository: testCase.repo,
				User:       testCase.user,
				Branch:     testCase.branch,
				SourceSha:  testCase.sourceSha,
				TargetRef:  testCase.targetRef,
			}
			_, err := client.UserMergeToRef(ctx, request)
			testhelper.RequireGrpcError(t, err, testCase.code)
		})
	}
}

func TestUserMergeToRefIgnoreHooksRequest(t *testing.T) {
	ctx, cleanup := testhelper.Context()
	defer cleanup()

	serverSocketPath, stop := runOperationServiceServer(t)
	defer stop()

	client, conn := newOperationClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, testRepoPath, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	prepareMergeBranch(t, testRepoPath)

	targetRef := []byte("refs/merge-requests/x/merge")
	mergeCommitMessage := "Merged by Gitaly"

	request := &gitalypb.UserMergeToRefRequest{
		Repository: testRepo,
		SourceSha:  commitToMerge,
		Branch:     []byte(mergeBranchName),
		TargetRef:  targetRef,
		User:       testhelper.TestUser,
		Message:    []byte(mergeCommitMessage),
	}

	hookContent := []byte("#!/bin/sh\necho 'failure'\nexit 1")

	for _, hookName := range gitlabPreHooks {
		t.Run(hookName, func(t *testing.T) {
			remove, err := testhelper.WriteCustomHook(testRepoPath, hookName, hookContent)
			require.NoError(t, err)
			defer remove()

			resp, err := client.UserMergeToRef(ctx, request)
			require.NoError(t, err)
			require.Empty(t, resp.PreReceiveError)
		})
	}
}

func prepareMergeBranch(t *testing.T, testRepoPath string) {
	prepareMergeBranchWithHead(t, testRepoPath, mergeBranchHeadBefore)
}

func prepareMergeBranchWithHead(t *testing.T, testRepoPath, mergeBranchHead string) {
	deleteBranch(testRepoPath, mergeBranchName)
	out, err := exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "branch", mergeBranchName, mergeBranchHead).CombinedOutput()
	require.NoError(t, err, "set up branch to merge into: %s", out)
}

func deleteBranch(testRepoPath, branchName string) {
	exec.Command(config.Config.Git.BinPath, "-C", testRepoPath, "branch", "-D", branchName).Run()
}

// This error is used as a sentinel value
var errRecvTimeout = fmt.Errorf("timeout waiting for response")

func recvTimeout(bidi gitalypb.OperationService_UserMergeBranchClient, timeout time.Duration) (*gitalypb.UserMergeBranchResponse, error) {
	type responseError struct {
		response *gitalypb.UserMergeBranchResponse
		err      error
	}
	responseCh := make(chan responseError, 1)

	go func() {
		resp, err := bidi.Recv()
		responseCh <- responseError{resp, err}
	}()

	select {
	case respErr := <-responseCh:
		return respErr.response, respErr.err
	case <-time.After(timeout):
		return nil, errRecvTimeout
	}
}
