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

    def wiki_find_page(request, call)
      bridge_exceptions do
        repo = Gitlab::Git::Repository.from_call(call)
        wiki = Gitlab::Git::Wiki.new(repo)

        page = wiki.page(
          title: request.title,
          version: request.revision.presence,
          dir: request.directory.presence
        )

        unless page
          return Enumerator.new do |y|
            y.yield Gitaly::WikiFindPageResponse.new
          end
        end

        version = Gitaly::WikiPageVersion.new(
          commit: gitaly_commit_from_rugged(page.version.commit.raw_commit),
          format: page.version.format.to_s
        )
        gitaly_wiki_page = Gitaly::WikiPage.new(
          version: version,
          format: page.format.to_s,
          title: page.title.b,
          url_path: page.url_path.to_s,
          path: page.path.b,
          name: page.name.b,
          historical: page.historical?
        )

        Enumerator.new do |y|
          io = StringIO.new(page.text_data)
          while chunk = io.read(Gitlab.config.git.write_buffer_size)
            gitaly_wiki_page.raw_data = chunk

            y.yield Gitaly::WikiFindPageResponse.new(page: gitaly_wiki_page)

            gitaly_wiki_page = Gitaly::WikiPage.new
          end
        end
      end
    end
  end
end
