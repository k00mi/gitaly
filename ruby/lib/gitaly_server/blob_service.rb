module GitalyServer
  class BlobService < Gitaly::BlobService::Service
    include Utils

    def get_lfs_pointers(request, call)
      bridge_exceptions do
        repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)
        blobs = Gitlab::Git::Blob.batch_lfs_pointers(repo, request.blob_ids)

        Enumerator.new do |y|
          # LFS pointers are maximum 200 bytes in size, and for a default message size of 4 MB
          # we got more than enough room for 100 blobs.
          blobs.each_slice(100) do |blobs_slice|
            lfs_pointers = blobs_slice.map do |blob|
              Gitaly::LFSPointer.new(
                size: blob.size,
                data: blob.data,
                oid: blob.id
              )
            end

            y.yield Gitaly::GetLFSPointersResponse.new(lfs_pointers: lfs_pointers)
          end
        end
      end
    end
  end
end
