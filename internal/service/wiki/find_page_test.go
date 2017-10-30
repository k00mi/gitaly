package wiki

import (
	"bytes"
	"io"
	"testing"

	gitlog "gitlab.com/gitlab-org/gitaly/internal/git/log"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

func TestSuccessfulWikiFindPageRequest(t *testing.T) {
	wikiRepo, cleanupFunc := setupWikiRepo()
	defer cleanupFunc()

	ctx, cancel := testhelper.Context()
	defer cancel()

	server, serverSocketPath := runWikiServiceServer(t)
	defer server.Stop()

	client, conn := newWikiClient(t, serverSocketPath)
	defer conn.Close()

	content := bytes.Repeat([]byte("Mock wiki page content"), 10000)

	page1Name := "Home Page"
	page2Name := "Installing/Step 133-b"
	page3Name := "Installing/Step 133-c"

	writeWikiPage(t, client, wikiRepo, page1Name, content)
	head1ID := testhelper.MustRunCommand(t, nil, "git", "-C", wikiRepoPath, "show", "--format=format:%H", "--no-patch", "HEAD")
	page1Commit, err := gitlog.GetCommit(ctx, wikiRepo, string(head1ID), "")
	require.NoError(t, err, "look up git commit after writing a wiki page")

	writeWikiPage(t, client, wikiRepo, page2Name, content)

	writeWikiPage(t, client, wikiRepo, page3Name, content)
	head3ID := testhelper.MustRunCommand(t, nil, "git", "-C", wikiRepoPath, "show", "--format=format:%H", "--no-patch", "HEAD")
	page3Commit, err := gitlog.GetCommit(ctx, wikiRepo, string(head3ID), "")
	require.NoError(t, err, "look up git commit after writing a wiki page")

	testCases := []struct {
		desc         string
		request      *pb.WikiFindPageRequest
		expectedPage *pb.WikiPage
	}{
		{
			desc: "title only",
			request: &pb.WikiFindPageRequest{
				Repository: wikiRepo,
				Title:      []byte(page1Name),
			},
			expectedPage: &pb.WikiPage{
				Version: &pb.WikiPageVersion{
					Commit: page3Commit,
					Format: "markdown",
				},
				Title:      []byte(page1Name),
				Format:     "markdown",
				UrlPath:    "Home-Page",
				Path:       []byte("Home-Page.md"),
				Name:       []byte(page1Name),
				Historical: false,
			},
		},
		{
			desc: "title + revision that includes the page",
			request: &pb.WikiFindPageRequest{
				Repository: wikiRepo,
				Title:      []byte(page1Name),
				Revision:   []byte(page1Commit.Id),
			},
			expectedPage: &pb.WikiPage{
				Version: &pb.WikiPageVersion{
					Commit: page1Commit,
					Format: "markdown",
				},
				Title:      []byte(page1Name),
				Format:     "markdown",
				UrlPath:    "Home-Page",
				Path:       []byte("Home-Page.md"),
				Name:       []byte(page1Name),
				Historical: true,
			},
		},
		{
			desc: "title + revision that does not include the page",
			request: &pb.WikiFindPageRequest{
				Repository: wikiRepo,
				Title:      []byte(page2Name),
				Revision:   []byte(page1Commit.Id),
			},
			expectedPage: nil,
		},
		{
			desc: "title + directory that includes the page",
			request: &pb.WikiFindPageRequest{
				Repository: wikiRepo,
				Title:      []byte("Step 133-b"),
				Directory:  []byte("Installing"),
			},
			expectedPage: &pb.WikiPage{
				Version: &pb.WikiPageVersion{
					Commit: page3Commit,
					Format: "markdown",
				},
				Title:      []byte("Step 133 b"),
				Format:     "markdown",
				UrlPath:    "Installing/Step-133-b",
				Path:       []byte("Installing/Step-133-b.md"),
				Name:       []byte("Step 133 b"),
				Historical: false,
			},
		},
		{
			desc: "title + directory that does not include the page",
			request: &pb.WikiFindPageRequest{
				Repository: wikiRepo,
				Title:      []byte("Step 133-b"),
				Directory:  []byte("Installation"),
			},
			expectedPage: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			c, err := client.WikiFindPage(ctx, testCase.request)
			require.NoError(t, err)

			expectedPage := testCase.expectedPage
			receivedPage := readFullWikiPageFromWikiFindPageClient(t, c)

			// require.Equal doesn't display a proper diff when either expected/actual has a field
			// with large data (RawData in our case), so we compare page attributes and content separately.
			receivedContent := receivedPage.GetRawData()
			if receivedPage != nil {
				receivedPage.RawData = nil
			}

			require.Equal(t, expectedPage, receivedPage, "mismatched page attributes")
			if expectedPage != nil {
				require.Equal(t, content, receivedContent, "mismatched page content")
			}
		})
	}
}

func TestFailedWikiFindPageDueToValidation(t *testing.T) {
	wikiRepo, cleanupFunc := setupWikiRepo()
	defer cleanupFunc()

	server, serverSocketPath := runWikiServiceServer(t)
	defer server.Stop()

	client, conn := newWikiClient(t, serverSocketPath)
	defer conn.Close()

	testCases := []struct {
		desc  string
		title string
		code  codes.Code
	}{
		{
			desc:  "empty page path",
			title: "",
			code:  codes.InvalidArgument,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.desc, func(t *testing.T) {
			ctx, cancel := testhelper.Context()
			defer cancel()

			request := &pb.WikiFindPageRequest{
				Repository: wikiRepo,
				Title:      []byte(testCase.title),
			}

			c, err := client.WikiFindPage(ctx, request)
			require.NoError(t, err)

			err = drainWikiFindPageResponse(c)
			testhelper.AssertGrpcError(t, err, testCase.code, "")
		})
	}
}

func drainWikiFindPageResponse(c pb.WikiService_WikiFindPageClient) error {
	for {
		_, err := c.Recv()
		if err != nil {
			return err
		}
	}
}

func readFullWikiPageFromWikiFindPageClient(t *testing.T, c pb.WikiService_WikiFindPageClient) (wikiPage *pb.WikiPage) {
	for {
		resp, err := c.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}

		if wikiPage == nil {
			wikiPage = resp.GetPage()
		} else {
			wikiPage.RawData = append(wikiPage.RawData, resp.GetPage().GetRawData()...)
		}
	}

	return wikiPage
}
