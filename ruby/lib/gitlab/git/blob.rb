module Gitlab
  module Git
    class Blob
      class << self
        def find(repository, sha, path, limit: MAX_DATA_DISPLAY_SIZE)
          return unless path

          # Strip any leading / characters from the path
          path = path.sub(%r{\A/*}, '')

          rugged_commit = repository.lookup(sha)
          root_tree = rugged_commit.tree

          blob_entry = find_entry_by_path(repository, root_tree.oid, *path.split('/'))

          return nil unless blob_entry

          if blob_entry[:type] == :commit
            submodule_blob(blob_entry, path, sha)
          else
            blob = repository.lookup(blob_entry[:oid])

            if blob
              new(
                id: blob.oid,
                name: blob_entry[:name],
                size: blob.size,
                # Rugged::Blob#content is expensive; don't call it if we don't have to.
                data: limit.zero? ? '' : blob.content(limit),
                mode: blob_entry[:filemode].to_s(8),
                path: path,
                commit_id: sha,
                binary: blob.binary?
              )
            end
          end
        rescue Rugged::ReferenceError
          nil
        end
      end
    end
  end
end
