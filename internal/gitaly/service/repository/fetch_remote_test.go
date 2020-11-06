package repository

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/gitaly/config"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"gitlab.com/gitlab-org/gitaly/proto/go/gitalypb"
	"google.golang.org/grpc/codes"
)

func copyRepoWithNewRemote(t *testing.T, repo *gitalypb.Repository, locator storage.Locator, remote string) *gitalypb.Repository {
	repoPath, err := locator.GetRepoPath(repo)
	require.NoError(t, err)

	cloneRepo := &gitalypb.Repository{StorageName: repo.GetStorageName(), RelativePath: "fetch-remote-clone.git"}

	clonePath := filepath.Join(testhelper.GitlabTestStoragePath(), "fetch-remote-clone.git")
	t.Logf("clonePath: %q", clonePath)
	os.RemoveAll(clonePath)

	testhelper.MustRunCommand(t, nil, "git", "clone", "--bare", repoPath, clonePath)

	testhelper.MustRunCommand(t, nil, "git", "-C", clonePath, "remote", "add", remote, repoPath)

	return cloneRepo
}

func TestFetchRemoteSuccess(t *testing.T) {
	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	locator := config.NewLocator(config.Config)
	serverSocketPath, stop := runRepoServer(t, locator)
	defer stop()

	client, _ := newRepositoryClient(t, serverSocketPath)

	cloneRepo := copyRepoWithNewRemote(t, testRepo, locator, "my-remote")
	defer func(r *gitalypb.Repository) {
		path, err := locator.GetRepoPath(r)
		if err != nil {
			panic(err)
		}
		os.RemoveAll(path)
	}(cloneRepo)

	resp, err := client.FetchRemote(ctx, &gitalypb.FetchRemoteRequest{
		Repository: cloneRepo,
		Remote:     "my-remote",
		Timeout:    120,
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestFetchRemoteFailure(t *testing.T) {
	server := NewServer(config.Config, RubyServer, config.NewLocator(config.Config), config.GitalyInternalSocketPath())

	tests := []struct {
		desc string
		req  *gitalypb.FetchRemoteRequest
		code codes.Code
		err  string
	}{
		{
			desc: "invalid storage",
			req:  &gitalypb.FetchRemoteRequest{Repository: &gitalypb.Repository{StorageName: "invalid", RelativePath: "foobar.git"}},
			code: codes.InvalidArgument,
			err:  `GetStorageByName: no such storage: "invalid"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			resp, err := server.FetchRemote(ctx, tc.req)
			testhelper.RequireGrpcError(t, err, tc.code)
			require.Contains(t, err.Error(), tc.err)
			assert.Nil(t, resp)
		})
	}
}

const (
	httpToken = "ABCefg0999182"
)

func remoteHTTPServer(t *testing.T, repoName, httpToken string) (*httptest.Server, string) {
	s := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.String() == fmt.Sprintf("/%s.git/info/refs?service=git-upload-pack", repoName) {
				if httpToken != "" && r.Header.Get("Authorization") != httpToken {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
				w.WriteHeader(http.StatusOK)

				b, err := ioutil.ReadFile("testdata/advertise.txt")
				require.NoError(t, err)
				w.Write(b)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}),
	)

	return s, fmt.Sprintf("%s/%s.git", s.URL, repoName)
}

func getRefnames(t *testing.T, repoPath string) []string {
	result := testhelper.MustRunCommand(t, nil, "git", "-C", repoPath, "for-each-ref", "--format", "%(refname:lstrip=2)")
	return strings.Split(string(bytes.TrimRight(result, "\n")), "\n")
}

func TestFetchRemoteOverHTTP(t *testing.T) {
	locator := config.NewLocator(config.Config)
	serverSocketPath, stop := runRepoServer(t, locator)
	defer stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	ctx, cancel := testhelper.Context()
	defer cancel()

	testCases := []struct {
		description string
		httpToken   string
	}{
		{
			description: "with http token",
			httpToken:   httpToken,
		},
		{
			description: "without http token",
			httpToken:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			forkedRepo, forkedRepoPath, forkedRepoCleanup := testhelper.NewTestRepo(t)
			defer forkedRepoCleanup()

			_, remoteURL := remoteHTTPServer(t, "my-repo", tc.httpToken)

			req := &gitalypb.FetchRemoteRequest{
				Repository: forkedRepo,
				RemoteParams: &gitalypb.Remote{
					Url:                     remoteURL,
					Name:                    "geo",
					HttpAuthorizationHeader: tc.httpToken,
					MirrorRefmaps:           []string{"all_refs"},
				},
				Timeout: 1000,
			}

			refs := getRefnames(t, forkedRepoPath)
			require.True(t, len(refs) > 1, "the advertisement.txt should have deleted all refs except for master")

			_, err := client.FetchRemote(ctx, req)
			require.NoError(t, err)

			refs = getRefnames(t, forkedRepoPath)

			require.Len(t, refs, 1)
			assert.Equal(t, "master", refs[0])
		})
	}
}

func TestFetchRemoteOverHTTPWithRedirect(t *testing.T) {
	locator := config.NewLocator(config.Config)
	serverSocketPath, stop := runRepoServer(t, locator)
	defer stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	ctx, cancel := testhelper.Context()
	defer cancel()

	testRepo, _, cleanup := testhelper.NewTestRepo(t)
	defer cleanup()

	s := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/info/refs?service=git-upload-pack", r.URL.String())
			http.Redirect(w, r, "/redirect_url", http.StatusSeeOther)
		}),
	)

	req := &gitalypb.FetchRemoteRequest{
		Repository:   testRepo,
		RemoteParams: &gitalypb.Remote{Url: s.URL, Name: "geo"},
		Timeout:      1000,
	}

	_, err := client.FetchRemote(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "The requested URL returned error: 303")
}

func TestFetchRemoteOverHTTPError(t *testing.T) {
	locator := config.NewLocator(config.Config)
	serverSocketPath, stop := runRepoServer(t, locator)
	defer stop()

	client, conn := newRepositoryClient(t, serverSocketPath)
	defer conn.Close()

	testRepo, _, cleanupFn := testhelper.NewTestRepo(t)
	defer cleanupFn()

	testCases := []struct {
		desc    string
		request gitalypb.FetchRemoteRequest
		code    codes.Code
		err     string
	}{
		{
			desc: "bad remote url",
			request: gitalypb.FetchRemoteRequest{
				Repository: testRepo,
				RemoteParams: &gitalypb.Remote{
					Url:                     "not a url",
					Name:                    "remote",
					HttpAuthorizationHeader: httpToken,
				},
				Timeout: 1000,
			},
			code: codes.InvalidArgument,
			err:  "invalid remote url",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			resp, err := client.FetchRemote(ctx, &tc.request)
			testhelper.RequireGrpcError(t, err, tc.code)
			require.Contains(t, err.Error(), tc.err)
			assert.Nil(t, resp)
		})
	}
}
