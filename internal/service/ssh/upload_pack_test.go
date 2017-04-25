package ssh

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"gitlab.com/gitlab-org/gitaly/internal/testhelper"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"

	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

func TestSuccessfulUploadPackRequest(t *testing.T) {
	server := runSSHServer(t)
	defer server.Stop()

	localRepoPath := path.Join(testRepoRoot, "gitlab-test-local")
	remoteRepoPath := path.Join(testRepoRoot, "gitlab-test-remote")
	// Make a non-bare clone of the test repo to act as a remote one
	testhelper.MustRunCommand(t, nil, "git", "clone", testhelper.GitlabTestRepoPath(), remoteRepoPath)
	// Make a bare clone of the test repo to act as a local one and to leave the original repo intact for other tests
	testhelper.MustRunCommand(t, nil, "git", "clone", "--bare", testhelper.GitlabTestRepoPath(), localRepoPath)
	defer os.RemoveAll(localRepoPath)
	defer os.RemoveAll(remoteRepoPath)

	commitMsg := fmt.Sprintf("Testing UploadPack RPC around %d", time.Now().Unix())
	committerName := "Scrooge McDuck"
	committerEmail := "scrooge@mcduck.com"
	clientCapabilities := "multi_ack_detailed no-done side-band-64k thin-pack include-tag ofs-delta deepen-since deepen-not agent=git/2.12.0"

	// The latest commit ID on the local repo
	oldHead := bytes.TrimSpace(testhelper.MustRunCommand(t, nil, "git", "-C", remoteRepoPath, "rev-parse", "master"))

	testhelper.MustRunCommand(t, nil, "git", "-C", remoteRepoPath,
		"-c", fmt.Sprintf("user.name=%s", committerName),
		"-c", fmt.Sprintf("user.email=%s", committerEmail),
		"commit", "--allow-empty", "-m", commitMsg)

	// The commit ID we want to pull from the remote repo
	newHead := bytes.TrimSpace(testhelper.MustRunCommand(t, nil, "git", "-C", remoteRepoPath, "rev-parse", "master"))

	// UploadPack request is a "want" packet line followed by a packet flush, then many "have" packets followed by a packet flush.
	// This is explained a bit in https://git-scm.com/book/en/v2/Git-Internals-Transfer-Protocols#_downloading_data
	wantPkt := fmt.Sprintf("want %s %s\n", newHead, clientCapabilities)
	havePkt := fmt.Sprintf("have %s\n", oldHead)

	// We don't check for errors because per bytes.Buffer docs, Buffer.Write will always return a nil error.
	requestBuffer := &bytes.Buffer{}
	fmt.Fprintf(requestBuffer, "%04x%s%s", len(wantPkt)+4, wantPkt, pktFlushStr)
	fmt.Fprintf(requestBuffer, "%04x%s%s", len(havePkt)+4, havePkt, pktFlushStr)

	client := newSSHClient(t)
	repo := &pb.Repository{Path: path.Join(remoteRepoPath, ".git")}
	rpcRequest := &pb.SSHUploadPackRequest{Repository: repo}
	stream, err := client.SSHUploadPack(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if err = stream.Send(rpcRequest); err != nil {
		t.Fatal(err)
	}

	data := make([]byte, 16)
	for {
		n, err := requestBuffer.Read(data)
		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal(err)
		}

		rpcRequest = &pb.SSHUploadPackRequest{Stdin: data[:n]}
		if err := stream.Send(rpcRequest); err != nil {
			t.Fatal(err)
		}
	}
	stream.CloseSend()

	responseBuffer := &bytes.Buffer{}
	var chunk int
	for {
		rpcResponse, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				t.Fatal(err)
			}
		}
		chunk++

		if rpcResponse.Stdout != nil {
			responseBuffer.Write(rpcResponse.GetStdout())
		}
		if rpcResponse.Stderr != nil {
			responseBuffer.Write(rpcResponse.GetStderr())
		}
		//		responseBuffer.Write(rpcResponse.GetStderr())
		t.Logf("Read chunk %d", chunk)
	}

	// There's no git command we can pass it this response and do the work for us (extracting pack file, ...),
	// so we have to do it ourselves.
	pack, version, entries := extractPackDataFromResponse(t, responseBuffer)
	if pack == nil {
		t.Errorf("Expected to find a pack file in response, found none")
		return
	}

	err = drainUploadStreamAndVerifyExitStatus(t, 0, stream)
	if err != nil {
		t.Fatal(err)
	}

	testhelper.MustRunCommand(t, bytes.NewReader(pack), "git", "-C", localRepoPath, "unpack-objects", fmt.Sprintf("--pack_header=%d,%d", version, entries))

	// The fact that this command succeeds means that we got the commit correctly, no further checks should be needed.
	testhelper.MustRunCommand(t, nil, "git", "-C", localRepoPath, "show", string(newHead))
}

func drainUploadStreamAndVerifyExitStatus(t *testing.T, status int32, stream pb.SSH_SSHUploadPackClient) error {
	var (
		err   error
		chunk *pb.SSHUploadPackResponse
	)
	for err == nil {
		chunk, err = stream.Recv()
	}
	if chunk.GetExitStatus().GetValue() != status {
		t.Fatalf("Expected ExitStatus to be %d, got %d", status, chunk.GetExitStatus().GetValue())
	}
	if err != io.EOF {
		return err
	}
	return nil
}

func TestFailedUploadPackRequestDueToValidationError(t *testing.T) {
	server := runSSHServer(t)
	defer server.Stop()

	client := newSSHClient(t)

	rpcRequests := []pb.SSHUploadPackRequest{
		{Repository: &pb.Repository{Path: ""}}, // Repository.Path is empty
		{Repository: nil},                      // Repository is nil
		{Repository: &pb.Repository{Path: "/path/to/repo"}, Stdin: []byte("Fail")}, // Data exists on first request
	}

	for _, rpcRequest := range rpcRequests {
		t.Logf("test case: %v", rpcRequest)
		stream, err := client.SSHUploadPack(context.Background())
		if err != nil {
			t.Fatal(err)
		}

		if err = stream.Send(&rpcRequest); err != nil {
			t.Fatal(err)
		}
		stream.CloseSend()

		err = drainPostUploadPackResponse(stream)
		testhelper.AssertGrpcError(t, err, codes.InvalidArgument, "")
	}
}

func drainPostUploadPackResponse(stream pb.SSH_SSHUploadPackClient) error {
	var err error
	for err == nil {
		_, err = stream.Recv()
	}
	return err
}

// The response contains bunch of things; metadata, progress messages, and a pack file. We're only
// interested in the pack file and its header values.
func extractPackDataFromResponse(t *testing.T, buf *bytes.Buffer) ([]byte, int, int) {
	var pack []byte
	t.Logf("complete buf: %q", buf.String())

	// Since this is Smart Protocol we need to do this twice...
	//  The first pass is listing all the refs that the server has
	//  The second pass is listing the commonly known last ref
	for i := 0; i < 2; i++ {
		// The response should have the following format, where <length> is always four hexadecimal digits.
		// <length><data>
		// <length><data>
		// ...
		// 0000
		for {
			pktLenStr := buf.Next(4)
			if len(pktLenStr) != 4 {
				return nil, 0, 0
			}
			if string(pktLenStr) == pktFlushStr {
				break
			}

			pktLen, err := strconv.ParseUint(string(pktLenStr), 16, 16)
			if err != nil {
				t.Fatal(err)
			}

			restPktLen := int(pktLen) - 4
			pkt := buf.Next(restPktLen)
			if len(pkt) != restPktLen {
				t.Fatalf("Incomplete packet read")
			}

			// The first byte of the packet is the band designator. We only care about data in band 1.
			if pkt[0] == 1 {
				pack = append(pack, pkt[1:]...)
			}
		}
	}

	t.Logf("resulting buf: %s", buf.String())

	// The packet is structured as follows:
	// 4 bytes for signature, here it's "PACK"
	// 4 bytes for header version
	// 4 bytes for header entries
	// The rest is the pack file
	if len(pack) < 4 || string(pack[:4]) != "PACK" {
		t.Fatalf("Invalid packet signature %q", pack)
	}
	version := int(binary.BigEndian.Uint32(pack[4:8]))
	entries := int(binary.BigEndian.Uint32(pack[8:12]))
	pack = pack[12:]

	return pack, version, entries
}
