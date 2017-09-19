module GitalyServer
  class OperationsService < Gitaly::OperationService::Service
    include Utils

    def user_delete_tag(request, call)
      repo = Gitlab::Git::Repository.from_call(call)

      gitaly_user = request.user
      raise GRPC::InvalidArgument.new('empty user') unless gitaly_user
      user = Gitlab::Git::User.from_gitaly(gitaly_user)

      tag_name = request.tag_name
      raise GRPC::InvalidArgument.new('empty tag name') if tag_name.blank?

      repo.rm_tag(tag_name, user: user)

      Gitaly::UserDeleteTagResponse.new
    rescue Gitlab::Git::HooksService::PreReceiveError => e
      raise GRPC::FailedPrecondition.new(e.to_s)
    end
  end
end
