module GitalyServer
  class CommitService < Gitaly::CommitService::Service
    def filter_shas_with_signatures(_session, call)
      Enumerator.new do |y|
        repository = nil

        call.each_remote_read.with_index do |request, index|
          repository = Gitlab::Git::Repository.from_gitaly(request.repository, call) if index.zero?

          y << Gitaly::FilterShasWithSignaturesResponse.new(shas: Gitlab::Git::Commit.shas_with_signatures(repository, request.shas))
        end
      end
    end
  end
end
