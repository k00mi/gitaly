module GitalyServer
  class WikiService < Gitaly::WikiService::Service
    include Utils

    def wiki_write_page(call)
      bridge_exceptions do
        begin
          repo = Gitlab::Git::Repository.from_call(call)
          name = format = commit_details = nil
          content = ""

          call.each_remote_read.with_index do |request, index|
            if index.zero?
              name = request.name
              format = request.format
              commit_details = request.commit_details
            end

            content << request.content
          end

          wiki = Gitlab::Git::Wiki.new(repo)
          commit_details = Gitlab::Git::Wiki::CommitDetails.new(
            commit_details.name,
            commit_details.email,
            commit_details.message
          )

          wiki.write_page(name, format.to_sym, content, commit_details)

          Gitaly::WikiWritePageResponse.new
        rescue Gitlab::Git::Wiki::DuplicatePageError => e
          Gitaly::WikiWritePageResponse.new(duplicate_error: e.message.b)
        end
      end
    end
  end
end
