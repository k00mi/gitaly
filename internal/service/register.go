package service

import (
	"gitlab.com/gitlab-org/gitaly/internal/service/smarthttp"
	pb "gitlab.com/gitlab-org/gitaly/protos/go"

	"google.golang.org/grpc"
)

func RegisterAll(grpcServer *grpc.Server) {
	pb.RegisterSmartHTTPServer(grpcServer, smarthttp.NewServer())
}
