module Gitlab
  module Git
    # These are monkey patches on top of the vendored version of Repository.
    class Repository
      class << self
        def from_gitaly(gitaly_repository, call)
          new(
            gitaly_repository,
            GitalyServer.repo_path(call),
            GitalyServer.gl_repository(call),
            Gitlab::Git::GitlabProjects.from_gitaly(gitaly_repository, call),
            GitalyServer.repo_alt_dirs(call)
          )
        end

        def create(repo_path)
          FileUtils.mkdir_p(repo_path, mode: 0770)

          # Equivalent to `git --git-path=#{repo_path} init [--bare]`
          repo = Rugged::Repository.init_at(repo_path, true)
          repo.close

          symlink_hooks_to = Gitlab.config.gitlab_shell.hooks_path
          create_hooks(repo_path, symlink_hooks_to) if symlink_hooks_to.present?
        end

        def create_hooks(repo_path, global_hooks_path)
          local_hooks_path = File.join(repo_path, 'hooks')
          real_local_hooks_path = :not_found

          begin
            real_local_hooks_path = File.realpath(local_hooks_path)
          rescue Errno::ENOENT
            # real_local_hooks_path == :not_found
          end

          # Do nothing if hooks already exist
          unless real_local_hooks_path == File.realpath(global_hooks_path)
            if File.exist?(local_hooks_path)
              # Move the existing hooks somewhere safe
              FileUtils.mv(
                local_hooks_path,
                "#{local_hooks_path}.old.#{Time.now.to_i}")
            end

            # Create the hooks symlink
            FileUtils.ln_sf(global_hooks_path, local_hooks_path)
          end

          true
        end
      end

      attr_reader :path

      def initialize(gitaly_repository, path, gl_repository, gitlab_projects, combined_alt_dirs="")
        @gitaly_repository = gitaly_repository

        @alternate_object_directories = combined_alt_dirs
          .split(File::PATH_SEPARATOR)
          .map { |d| File.join(path, d) }

        @storage = gitaly_repository.storage_name
        @relative_path = gitaly_repository.relative_path
        @path = path
        @gl_repository = gl_repository
        @gitlab_projects = gitlab_projects
      end

      def add_branch(branch_name, user:, target:)
        target_object = Ref.dereference_object(lookup(target))
        raise InvalidRef.new("target not found: #{target}") unless target_object

        OperationService.new(user, self).add_branch(branch_name, target_object.oid)
        find_branch(branch_name)
      rescue Rugged::ReferenceError => ex
        raise InvalidRef, ex
      end

      # Fake implementation, so we wrap correctly on the client side
      def wrapped_gitaly_errors
        yield
      end

      def circuit_breaker
        FakeCircuitBreaker
      end

      def gitaly_repository
        @gitaly_repository
      end

      def alternate_object_directories
        @alternate_object_directories
      end

      def relative_object_directories
        raise "don't use relative object directories in gitaly-ruby"
      end

      # TODO: Can be removed once https://gitlab.com/gitlab-org/gitaly/merge_requests/738
      #       is well and truly out in the wild.
      def fsck
        msg, status = run_git(%W[--git-dir=#{path} fsck], nice: true)
        raise GitError.new("Could not fsck repository: #{msg}") unless status.zero?
      end

      def exists?
        File.exist?(File.join(path, 'refs'))
      end

      def root_ref
        @root_ref ||= gitaly_migrate(:root_ref) do |is_enabled|
          if is_enabled
            gitaly_ref_client.default_branch_name
          else
            discover_default_branch
          end
        end
      end

      def branch_names
        gitaly_migrate(:branch_names) do |is_enabled|
          if is_enabled
            gitaly_ref_client.branch_names
          else
            branches.map(&:name)
          end
        end
      end

      def branches
        gitaly_migrate(:branches) do |is_enabled|
          if is_enabled
            gitaly_ref_client.branches
          else
            branches_filter
          end
        end
      end

      def local_branches(sort_by: nil)
        branches_filter(filter: :local, sort_by: sort_by)
      end

      def has_local_branches?
        local_branches.any?
      end

       def has_local_branches_rugged?
        rugged.branches.each(:local).any? do |ref|
          begin
            ref.name && ref.target # ensures the branch is valid
 
            true
          rescue Rugged::ReferenceError
            false
          end
        end
      end

      def tag_names
        gitaly_migrate(:tag_names) do |is_enabled|
          if is_enabled
            gitaly_ref_client.tag_names
          else
            rugged.tags.map { |t| t.name }
          end
        end
      end

      def tags
        gitaly_migrate(:tags) do |is_enabled|
          if is_enabled
            tags_from_gitaly
          else

            rugged.references.each("refs/tags/*").map do |ref|
              message = nil

              if ref.target.is_a?(Rugged::Tag::Annotation)
                tag_message = ref.target.message

                if tag_message.respond_to?(:chomp)
                  message = tag_message.chomp
                end
              end

              target_commit = Gitlab::Git::Commit.find(self, ref.target)
              Gitlab::Git::Tag.new(self, {
                name: ref.name,
                target: ref.target,
                target_commit: target_commit,
                message: message
              })
            end.sort_by(&:name)
          end
        end
      end

      # Discovers the default branch based on the repository's available branches
      #
      # - If no branches are present, returns nil
      # - If one branch is present, returns its name
      # - If two or more branches are present, returns current HEAD or master or first branch
      def discover_default_branch
        names = branch_names

        return if names.empty?

        return names[0] if names.length == 1

        if rugged_head
          extracted_name = Ref.extract_branch_name(rugged_head.name)

          return extracted_name if names.include?(extracted_name)
        end

        if names.include?('master')
          'master'
        else
          names[0]
        end
      end

      private

      def uncached_has_local_branches?
        gitaly_migrate(:has_local_branches, status: Gitlab::GitalyClient::MigrationStatus::OPT_OUT) do |is_enabled|
          if is_enabled
            gitaly_repository_client.has_local_branches?
          else
            has_local_branches_rugged?
          end
        end
      end

      # Gitaly note: JV: Trying to get rid of the 'filter' option so we can implement this with 'git'.
      def branches_filter(filter: nil, sort_by: nil)
        branches = rugged.branches.each(filter).map do |rugged_ref|
          begin
            target_commit = Gitlab::Git::Commit.find(self, rugged_ref.target)
            Gitlab::Git::Branch.new(self, rugged_ref.name, rugged_ref.target, target_commit)
          rescue Rugged::ReferenceError
            # Omit invalid branch
          end
        end.compact

        sort_branches(branches, sort_by)
      end
    end
  end
end
