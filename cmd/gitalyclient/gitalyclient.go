package main

import (
	"flag"

	log "github.com/Sirupsen/logrus"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	serverAddr         = flag.String("server_addr", "127.0.0.1:9999", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
)

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewCommitClient(conn)
	req := &pb.CommitIsAncestorRequest{}
	resp, err := client.CommitIsAncestor(context.Background(), req)
	if err != nil {
		log.WithError(err).Panic("Bad response")
	}

	log.WithField("response", resp).Info("Response")
}
