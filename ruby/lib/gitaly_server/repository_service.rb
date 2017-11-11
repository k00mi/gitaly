module GitalyServer
  class RepositoryService < Gitaly::RepositoryService::Service
    include Utils

    def create_repository(request, _call)
      bridge_exceptions do
        repo_path = GitalyServer.repo_path(_call)

        Gitlab::Git::Repository.create(repo_path, bare: true, symlink_hooks_to: Gitlab.config.gitlab_shell.hooks_path)

        Gitaly::CreateRepositoryResponse.new
      end
    end

    def has_local_branches(request, call)
      repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

      Gitaly::HasLocalBranchesResponse.new(value: repo.has_local_branches?)
    end
  end
end
