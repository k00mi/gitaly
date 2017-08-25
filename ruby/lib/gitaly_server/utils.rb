module GitalyServer
  module Utils
    def gitaly_commit_from_rugged(rugged_commit)
      Gitaly::GitCommit.new(
        id: rugged_commit.oid,
        subject: rugged_commit.message.split("\n", 2)[0].chomp,
        body: rugged_commit.message,
        parent_ids: rugged_commit.parent_ids,
        author: gitaly_commit_author_from_rugged(rugged_commit.author),
        committer: gitaly_commit_author_from_rugged(rugged_commit.committer),
      )
    end

    def gitaly_commit_author_from_rugged(rugged_author)
      Gitaly::CommitAuthor.new(
        name: bytes!(rugged_author[:name]),
        email: bytes!(rugged_author[:email]),
        date: Google::Protobuf::Timestamp.new(seconds: rugged_author[:time].to_i)
      )
    end

    def bytes!(string)
      string.force_encoding('ASCII-8BIT')
    end
  end
end
