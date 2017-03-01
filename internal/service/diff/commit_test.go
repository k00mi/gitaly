package diff

import (
	"bytes"
	"io"
	"net"
	"path"
	"strings"
	"testing"
	"time"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/diff"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var serverSocketPath = path.Join(scratchDir, "gitaly.sock")

func TestSuccessfulCommitDiffRequest(t *testing.T) {
	server := runDiffServer(t)
	defer server.Stop()

	client := newDiffClient(t)
	repo := &pb.Repository{Path: path.Join(testRepoRoot, testRepo)}
	rightCommit := "372ab6950519549b14d220271ee2322caa44d4eb"
	leftCommit := rightCommit + "~" // Parent of rightCommit
	rpcRequest := &pb.CommitDiffRequest{Repository: repo, RightCommitId: rightCommit, LeftCommitId: leftCommit}

	c, err := client.CommitDiff(context.Background(), rpcRequest)
	if err != nil {
		t.Fatal(err)
	}

	expectedDiffs := []struct {
		diff.Diff
		ChunksCombined []byte
	}{
		{
			Diff: diff.Diff{
				FromID:   "bdea48ee65c869eb0b86b1283069d76cce0a7254",
				ToID:     "0000000000000000000000000000000000000000",
				OldMode:  0100644,
				NewMode:  0,
				FromPath: []byte("gitaly/deleted-file"),
				ToPath:   []byte("/dev/null"),
				Binary:   false,
			},
			ChunksCombined: testhelper.MustReadFile(t, "testdata/deleted-file-chunks.txt"),
		},
		{
			Diff: diff.Diff{
				FromID:   "aa408b4556e594f7974390ad6b86210617fbda6e",
				ToID:     "1c69c4d2a65ad05c24ac3b6780b5748b97ffd3aa",
				OldMode:  0100644,
				NewMode:  0100644,
				FromPath: []byte("gitaly/file-with-multiple-chunks"),
				ToPath:   []byte("gitaly/file-with-multiple-chunks"),
				Binary:   false,
			},
			ChunksCombined: testhelper.MustReadFile(t, "testdata/file-with-multiple-chunks-chunks.txt"),
		},
		{
			Diff: diff.Diff{
				FromID:   "0000000000000000000000000000000000000000",
				ToID:     "bc2ef601a538d69ef99d5bdafa605e63f902e8e4",
				OldMode:  0,
				NewMode:  0100644,
				FromPath: []byte("gitaly/logo-white.png"),
				ToPath:   []byte("gitaly/logo-white.png"),
				Binary:   true,
			},
		},
		{
			Diff: diff.Diff{
				FromID:   "",
				ToID:     "",
				OldMode:  0100644,
				NewMode:  0100755,
				FromPath: []byte("gitaly/mode-file"),
				ToPath:   []byte("gitaly/mode-file"),
				Binary:   false,
			},
		},
		{
			Diff: diff.Diff{
				FromID:   "357406f3075a57708d0163752905cc1576fceacc",
				ToID:     "8e5177d718c561d36efde08bad36b43687ee6bf0",
				OldMode:  0100644,
				NewMode:  0100755,
				FromPath: []byte("gitaly/mode-file-with-mods"),
				ToPath:   []byte("gitaly/mode-file-with-mods"),
				Binary:   false,
			},
			ChunksCombined: testhelper.MustReadFile(t, "testdata/mode-file-with-mods-chunks.txt"),
		},
		{
			Diff: diff.Diff{
				FromID:   "43d24af4e22580f36b1ca52647c1aff75a766a33",
				ToID:     "0000000000000000000000000000000000000000",
				OldMode:  0100644,
				NewMode:  0,
				FromPath: []byte("gitaly/named-file-with-mods"),
				ToPath:   []byte("/dev/null"),
				Binary:   false,
			},
			ChunksCombined: testhelper.MustReadFile(t, "testdata/named-file-with-mods-chunks.txt"),
		},
		{
			Diff: diff.Diff{
				FromID:   "0000000000000000000000000000000000000000",
				ToID:     "b464dff7a75ccc92fbd920fd9ae66a84b9d2bf94",
				OldMode:  0,
				NewMode:  0100644,
				FromPath: []byte("/dev/null"),
				ToPath:   []byte("gitaly/no-newline-at-the-end"),
				Binary:   false,
			},
			ChunksCombined: testhelper.MustReadFile(t, "testdata/no-newline-at-the-end-chunks.txt"),
		},
		{
			Diff: diff.Diff{
				FromID:   "",
				ToID:     "",
				OldMode:  0,
				NewMode:  0,
				FromPath: []byte("gitaly/named-file"),
				ToPath:   []byte("gitaly/renamed-file"),
				Binary:   false,
			},
		},
		{
			Diff: diff.Diff{
				FromID:   "0000000000000000000000000000000000000000",
				ToID:     "3856c00e9450a51a62096327167fc43d3be62eef",
				OldMode:  0,
				NewMode:  0100644,
				FromPath: []byte("/dev/null"),
				ToPath:   []byte("gitaly/renamed-file-with-mods"),
				Binary:   false,
			},
			ChunksCombined: testhelper.MustReadFile(t, "testdata/renamed-file-with-mods-chunks.txt"),
		},
	}

	i := 0
	for {
		fetchedDiff, err := c.Recv()
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}

		if i >= len(expectedDiffs) {
			t.Errorf("Unexpected diff #%d received: %v", i, fetchedDiff)
			break
		}

		expectedDiff := expectedDiffs[i]

		if expectedDiff.FromID != fetchedDiff.FromId {
			t.Errorf("Expected diff #%d FromID to equal = %q, got %q", i, expectedDiff.FromID, fetchedDiff.FromId)
		}

		if expectedDiff.ToID != fetchedDiff.ToId {
			t.Errorf("Expected diff #%d ToID to equal = %q, got %q", i, expectedDiff.ToID, fetchedDiff.ToId)
		}

		if expectedDiff.OldMode != fetchedDiff.OldMode {
			t.Errorf("Expected diff #%d OldMode to equal = %o, got %o", i, expectedDiff.OldMode, fetchedDiff.OldMode)
		}

		if expectedDiff.NewMode != fetchedDiff.NewMode {
			t.Errorf("Expected diff #%d NewMode to equal = %o, got %o", i, expectedDiff.NewMode, fetchedDiff.NewMode)
		}

		if !bytes.Equal(expectedDiff.FromPath, fetchedDiff.FromPath) {
			t.Errorf("Expected diff #%d FromPath to equal = %s, got %s", i, expectedDiff.FromPath, fetchedDiff.FromPath)
		}

		if !bytes.Equal(expectedDiff.ToPath, fetchedDiff.ToPath) {
			t.Errorf("Expected diff #%d ToPath to equal = %s, got %s", i, expectedDiff.ToPath, fetchedDiff.ToPath)
		}

		if expectedDiff.Binary != fetchedDiff.Binary {
			t.Errorf("Expected diff #%d Binary to be %t, got %t", i, expectedDiff.Binary, fetchedDiff.Binary)
		}

		fetchedChunksCombined := bytes.Join(fetchedDiff.RawChunks, nil)
		if !bytes.Equal(expectedDiff.ChunksCombined, fetchedChunksCombined) {
			t.Errorf("Expected diff #%d Chunks to be %v, got %v", i, expectedDiff.ChunksCombined, fetchedChunksCombined)
		}

		i++
	}

	if len(expectedDiffs) != i {
		t.Errorf("Expected number of diffs to be %d, got %d", len(expectedDiffs), i)
	}
}

func TestFailedCommitDiffRequestWithEmptyRepository(t *testing.T) {
	server := runDiffServer(t)
	defer server.Stop()

	client := newDiffClient(t)
	repo := &pb.Repository{Path: ""}
	rightCommit := "d42783470dc29fde2cf459eb3199ee1d7e3f3a72"
	leftCommit := rightCommit + "~" // Parent of rightCommit
	// Case: Repository.Path is empty
	rpcRequest := &pb.CommitDiffRequest{Repository: repo, RightCommitId: rightCommit, LeftCommitId: leftCommit}

	c, err := client.CommitDiff(context.Background(), rpcRequest)
	if err != nil {
		t.Fatal(err)
	}

	expectedError := "Repository is empty"

	if err := drainCommitDiffResponse(c); !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to be %q, got %q", expectedError, err.Error())
	}

	// Case: Repository is nil
	rpcRequest = &pb.CommitDiffRequest{Repository: nil, RightCommitId: rightCommit, LeftCommitId: leftCommit}

	c, err = client.CommitDiff(context.Background(), rpcRequest)
	if err != nil {
		t.Fatal(err)
	}

	if err := drainCommitDiffResponse(c); !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to be %q, got %q", expectedError, err.Error())
	}
}

func TestFailedCommitDiffRequestWithEmptyCommit(t *testing.T) {
	server := runDiffServer(t)
	defer server.Stop()

	client := newDiffClient(t)
	repo := &pb.Repository{Path: path.Join(testRepoRoot, testRepo)}
	rightCommit := ""
	leftCommit := rightCommit + "~" // Parent of rightCommit
	rpcRequest := &pb.CommitDiffRequest{Repository: repo, RightCommitId: rightCommit, LeftCommitId: leftCommit}

	c, err := client.CommitDiff(context.Background(), rpcRequest)
	if err != nil {
		t.Fatal(err)
	}

	expectedError := "RightCommitId is empty"
	if err := drainCommitDiffResponse(c); !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to be %q, got %q", expectedError, err.Error())
	}

	rightCommit = "d42783470dc29fde2cf459eb3199ee1d7e3f3a72"
	leftCommit = ""
	rpcRequest = &pb.CommitDiffRequest{Repository: repo, RightCommitId: rightCommit, LeftCommitId: leftCommit}

	c, err = client.CommitDiff(context.Background(), rpcRequest)
	if err != nil {
		t.Fatal(err)
	}

	expectedError = "LeftCommitId is empty"
	if err := drainCommitDiffResponse(c); !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to be %q, got %q", expectedError, err.Error())
	}
}

func runDiffServer(t *testing.T) *grpc.Server {
	server := grpc.NewServer()
	listener, err := net.Listen("unix", serverSocketPath)
	if err != nil {
		t.Fatal(err)
	}

	pb.RegisterDiffServer(server, NewServer())
	reflection.Register(server)

	go server.Serve(listener)

	return server
}

func newDiffClient(t *testing.T) pb.DiffClient {
	connOpts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, _ time.Duration) (net.Conn, error) {
			return net.Dial("unix", addr)
		}),
	}
	conn, err := grpc.Dial(serverSocketPath, connOpts...)
	if err != nil {
		t.Fatal(err)
	}

	return pb.NewDiffClient(conn)
}

func drainCommitDiffResponse(c pb.Diff_CommitDiffClient) error {
	for {
		_, err := c.Recv()
		if err != nil {
			return err
		}
	}

	return nil
}
