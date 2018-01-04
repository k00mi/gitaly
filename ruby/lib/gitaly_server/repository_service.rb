module GitalyServer
  class RepositoryService < Gitaly::RepositoryService::Service
    include Utils

    def create_repository(request, call)
      bridge_exceptions do
        repo_path = GitalyServer.repo_path(call)

        Gitlab::Git::Repository.create(repo_path, bare: true, symlink_hooks_to: Gitlab.config.gitlab_shell.hooks_path)

        Gitaly::CreateRepositoryResponse.new
      end
    end

    def has_local_branches(request, call) # rubocop:disable Naming/PredicateName
      repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

      Gitaly::HasLocalBranchesResponse.new(value: repo.has_local_branches?)
    end

    def fetch_source_branch(request, call)
      bridge_exceptions do
        source_repository = Gitlab::Git::GitalyRemoteRepository.new(request.source_repository, call)
        repository = Gitlab::Git::Repository.from_gitaly(request.repository, call)
        result = repository.fetch_source_branch!(source_repository, request.source_branch, request.target_ref)

        Gitaly::FetchSourceBranchResponse.new(result: result)
      end
    end

    def fsck(request, call)
      repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

      repo.fsck

      Gitaly::FsckResponse.new
    rescue Gitlab::Git::Repository::GitError => ex
      Gitaly::FsckResponse.new(error: ex.message.b)
    rescue Rugged::RepositoryError => ex
      Gitaly::FsckResponse.new(error: ex.message.b)
    end

    def find_merge_base(request, call)
      bridge_exceptions do
        begin
          repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)
          base = repo.merge_base_commit(*request.revisions)

          Gitaly::FindMergeBaseResponse.new(base: base.to_s)
        rescue Rugged::ReferenceError
          Gitaly::FindMergeBaseResponse.new
        end
      end
    end

    def fetch_remote(request, call)
      bridge_exceptions do
        gitlab_projects = Gitlab::Git::GitlabProjects.from_gitaly(request.repository, call)

        success = gitlab_projects.fetch_remote(request.remote, request.timeout,
                                               force: request.force,
                                               tags: !request.no_tags,
                                               ssh_key: request.ssh_key,
                                               known_hosts: request.known_hosts)

        unless success
          raise GRPC::Unknown.new("Fetching remote #{request.remote} failed: #{gitlab_projects.output}")
        end

        Gitaly::FetchRemoteResponse.new
      end
    end

    def is_rebase_in_progress(request, call) # rubocop:disable Naming/PredicateName
      bridge_exceptions do
        repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

        result = repo.rebase_in_progress?(request.rebase_id)

        Gitaly::IsRebaseInProgressResponse.new(in_progress: result)
      end
    end
  end
end
