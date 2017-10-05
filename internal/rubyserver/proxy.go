package rubyserver

import (
	"context"
	"io"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"gitlab.com/gitlab-org/gitaly/internal/helper"

	"google.golang.org/grpc/metadata"
)

// SetHeaders adds headers that tell gitaly-ruby the full path to the repository.
func SetHeaders(ctx context.Context, repo *pb.Repository) (context.Context, error) {
	repoPath, err := helper.GetPath(repo)
	if err != nil {
		return nil, err
	}

	md := metadata.Pairs(repoPathHeader, repoPath, glRepositoryHeader, repo.GlRepository)
	newCtx := metadata.NewOutgoingContext(ctx, md)
	return newCtx, nil
}

// Proxy calls recvSend until it receives an error. The error is returned
// to the caller unless it is io.EOF.
func Proxy(recvSend func() error) (err error) {
	for err == nil {
		err = recvSend()
	}

	if err == io.EOF {
		err = nil
	}
	return err
}

// CloseSender captures the CloseSend method from gRPC streams.
type CloseSender interface {
	CloseSend() error
}

// ProxyBidi works like Proxy but runs multiple callbacks simultaneously.
// It returns immediately if proxying one of the callbacks fails. If the
// response stream is done, ProxyBidi returns immediately without waiting
// for the client stream to finish proxying.
func ProxyBidi(requestFunc func() error, requestStream CloseSender, responseFunc func() error) error {
	requestErr := make(chan error, 1)
	go func() {
		requestErr <- Proxy(requestFunc)
	}()

	responseErr := make(chan error, 1)
	go func() {
		responseErr <- Proxy(responseFunc)
	}()

	for {
		select {
		case err := <-requestErr:
			if err != nil {
				return err
			}
			requestStream.CloseSend()
		case err := <-responseErr:
			return err
		}
	}
}
