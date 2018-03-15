require 'licensee'

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

    def fetch_remote(request, call)
      bridge_exceptions do
        gitlab_projects = Gitlab::Git::GitlabProjects.from_gitaly(request.repository, call)

        success = gitlab_projects.fetch_remote(request.remote, request.timeout,
                                               force: request.force,
                                               tags: !request.no_tags,
                                               ssh_key: request.ssh_key.presence,
                                               known_hosts: request.known_hosts.presence)

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

    def is_squash_in_progress(request, call) # rubocop:disable Naming/PredicateName
      bridge_exceptions do
        repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

        result = repo.squash_in_progress?(request.squash_id)

        Gitaly::IsSquashInProgressResponse.new(in_progress: result)
      end
    end

    def write_ref(request, call)
      bridge_exceptions do
        repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

        # We ignore the output since the shell-version returns the output
        #  while the rugged-version returns true. But both throws expections on errors
        begin
          repo.write_ref(request.ref, request.revision, old_ref: request.old_revision, shell: request.shell)

          Gitaly::WriteRefResponse.new
        rescue Rugged::OSError => ex
          Gitaly::WriteRefResponse.new(error: ex.message.b)
        end
      end
    end

    def write_config(request, call)
      bridge_exceptions do
        begin
          repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

          repo.write_config(full_path: request.full_path)

          Gitaly::WriteConfigResponse.new
        rescue Rugged::Error => ex
          Gitaly::WriteConfigResponse.new(error: ex.message.b)
        end
      end
    end

    def find_license(request, call)
      bridge_exceptions do
        repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

        short_name = begin
                       ::Licensee.license(repo.path).try(:key)
                     rescue Rugged::Error
                     end

        Gitaly::FindLicenseResponse.new(license_short_name: short_name || "")
      end
    end
  end
end
