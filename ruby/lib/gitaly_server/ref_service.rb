module GitalyServer
  class RefService < Gitaly::RefService::Service
    include Utils

    TAGS_PER_MESSAGE = 100

    def create_branch(request, call)
      start_point = request.start_point
      start_point = 'HEAD' if start_point.empty?
      branch_name = request.name

      repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)
      rugged_ref = repo.rugged.branches.create(branch_name, start_point)

      Gitaly::CreateBranchResponse.new(
        status: :OK,
        branch: Gitaly::Branch.new(
          name: rugged_ref.name.b,
          target_commit: gitaly_commit_from_rugged(rugged_ref.target)
        )
      )
    rescue Rugged::ReferenceError => e
      status = case e.to_s
               when /'refs\/heads\/#{branch_name}' is not valid/
                 :ERR_INVALID
               when /a reference with that name already exists/
                 :ERR_EXISTS
               else
                 :ERR_INVALID_START_POINT
               end

      Gitaly::CreateBranchResponse.new(status: status)
    end

    def delete_branch(request, call)
      branch_name = request.name
      raise GRPC::InvalidArgument.new("empty Name") if branch_name.empty?

      repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)
      repo.delete_branch(branch_name)

      Gitaly::DeleteBranchResponse.new
    rescue Gitlab::Git::Repository::DeleteBranchError => e
      raise GRPC::Internal.new(e.to_s)
    end

    # Post 11.10 this method can be removed
    def delete_refs(request, call)
      repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

      begin
        if request.refs.any?
          repo.delete_refs(*request.refs)
        else
          repo.delete_all_refs_except(request.except_with_prefix)
        end

        Gitaly::DeleteRefsResponse.new
      rescue Gitlab::Git::Repository::GitError => e
        Gitaly::DeleteRefsResponse.new(git_error: e.message)
      end
    end

    def get_tag_messages(request, call)
      repository = Gitlab::Git::Repository.from_gitaly(request.repository, call)

      Enumerator.new do |y|
        request.tag_ids.each do |tag_id|
          annotation = repository.rugged.rev_parse(tag_id)
          next unless annotation

          response = Gitaly::GetTagMessagesResponse.new(tag_id: tag_id)
          io = StringIO.new(annotation.message)

          while chunk = io.read(Gitlab.config.git.max_commit_or_tag_message_size)
            response.message = chunk

            y.yield response

            response = Gitaly::GetTagMessagesResponse.new
          end
        end
      end
    end
  end
end
