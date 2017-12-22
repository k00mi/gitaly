module GitalyServer
  class RemoteService < Gitaly::RemoteService::Service
    include Utils

    def add_remote(request, call)
      bridge_exceptions do
        repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

        mirror_refmap = parse_refmap(request.mirror_refmap)

        repo.add_remote(request.name, request.url, mirror_refmap: mirror_refmap)

        Gitaly::AddRemoteResponse.new
      end
    end

    def remove_remote(request, call)
      bridge_exceptions do
        repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

        result = repo.remove_remote(request.name)

        Gitaly::RemoveRemoteResponse.new(result: result)
      end
    end

    def fetch_internal_remote(request, call)
      bridge_exceptions do
        repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)
        remote_repo = Gitlab::Git::GitalyRemoteRepository.new(request.remote_repository, call)

        result = repo.fetch_repository_as_mirror(remote_repo)

        Gitaly::FetchInternalRemoteResponse.new(result: result)
      end
    end

    private

    def parse_refmap(refmap)
      return unless refmap && refmap.rstrip != ""

      refmap_spec = refmap.to_sym

      if Gitlab::Git::RepositoryMirroring::REFMAPS.has_key?(refmap_spec)
        refmap_spec
      else
        refmap
      end
    end
  end
end
