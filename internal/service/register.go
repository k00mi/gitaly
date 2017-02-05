package service

import (
	"gitlab.com/gitlab-org/gitaly/internal/service/notifications"
	"gitlab.com/gitlab-org/gitaly/internal/service/smarthttp"
	pb "gitlab.com/gitlab-org/gitaly/protos/go"

	"google.golang.org/grpc"
)

func RegisterAll(grpcServer *grpc.Server) {
	pb.RegisterNotificationsServer(grpcServer, notifications.NewServer())
	pb.RegisterSmartHTTPServer(grpcServer, smarthttp.NewServer())
}
