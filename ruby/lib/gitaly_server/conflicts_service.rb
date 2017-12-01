module GitalyServer
  class ConflictsService < Gitaly::ConflictsService::Service
    include Utils

    def list_conflict_files(request, call)
      bridge_exceptions do
        begin
          resolver = conflict_resolver(request, call)
          conflicts = resolver.conflicts
          files = []
          msg_size = 0

          Enumerator.new do |y|
            conflicts.each do |file|
              files << Gitaly::ConflictFile.new(header: conflict_file_header(file))

              strio = StringIO.new(file.content)
              while chunk = strio.read(Gitlab.config.git.write_buffer_size - msg_size)
                files << Gitaly::ConflictFile.new(content: chunk)
                msg_size += chunk.bytesize

                # We don't send a message for each chunk because the content of
                # a file may be smaller than the size limit, which means we can
                # keep adding data to the message
                next if msg_size < Gitlab.config.git.write_buffer_size

                y.yield(Gitaly::ListConflictFilesResponse.new(files: files))

                files = []
                msg_size = 0
              end
            end

            # Send leftover data, if any
            y.yield(Gitaly::ListConflictFilesResponse.new(files: files)) if files.any?
          end
        rescue Gitlab::Git::Conflict::Resolver::ConflictSideMissing => e
          raise GRPC::FailedPrecondition.new(e.message)
        end
      end
    end

    private

    def conflict_resolver(request, call)
      repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

      Gitlab::Git::Conflict::Resolver.new(repo, request.our_commit_oid, request.their_commit_oid)
    end

    def conflict_file_header(file)
      Gitaly::ConflictFileHeader.new(
        repository: file.repository.gitaly_repository,
        commit_oid: file.commit_oid,
        their_path: file.their_path,
        our_path: file.our_path,
        our_mode: file.our_mode
      )
    end
  end
end
