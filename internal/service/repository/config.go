package repository

import (
	"regexp"

	"gitlab.com/gitlab-org/gitaly/internal/command"
	"gitlab.com/gitlab-org/gitaly/internal/git"
	"gitlab.com/gitlab-org/gitaly/internal/rubyserver"

	pb "gitlab.com/gitlab-org/gitaly-proto/go"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// validConfigKey is currently not an exhaustive validation, we can
	// improve it over time. It should reject no valid keys. It may fail to
	// reject some invalid keys.
	validConfigKey = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\-.]*$`)
)

func (*server) DeleteConfig(ctx context.Context, req *pb.DeleteConfigRequest) (*pb.DeleteConfigResponse, error) {
	for _, k := range req.Keys {
		if !validConfigKey.MatchString(k) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid config key: %q", k)
		}

		// We assume k does not contain any secrets; it is leaked via 'ps'.
		cmd, err := git.Command(ctx, req.Repository, "config", "--unset-all", k)
		if err != nil {
			return nil, err
		}

		if err := cmd.Wait(); err != nil {
			if code, ok := command.ExitStatus(err); ok && code == 5 {
				// Status code 5 means 'key not in config', see 'git help config'
				continue
			}

			return nil, status.Errorf(codes.Internal, "command failed: %v", err)
		}
	}

	return &pb.DeleteConfigResponse{}, nil
}

func (s *server) SetConfig(ctx context.Context, req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
	for _, entry := range req.Entries {
		if !validConfigKey.MatchString(entry.Key) {
			return nil, status.Errorf(codes.InvalidArgument, "invalid config key: %q", entry.Key)
		}
	}

	// We use gitaly-ruby here because in gitaly-ruby we can use Rugged, and
	// Rugged lets us set config values without leaking secrets via 'ps'. We
	// can't use `git config foo.bar secret` because that leaks secrets.
	client, err := s.RepositoryServiceClient(ctx)
	if err != nil {
		return nil, err
	}

	clientCtx, err := rubyserver.SetHeaders(ctx, req.GetRepository())
	if err != nil {
		return nil, err
	}

	return client.SetConfig(clientCtx, req)
}
