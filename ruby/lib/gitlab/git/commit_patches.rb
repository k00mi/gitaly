module Gitlab
  module Git
    class CommitPatches
      attr_reader :user, :repository, :branch_name, :patches

      def initialize(user, repository, branch_name, patches)
        @user = user
        @branch_name = branch_name
        @patches = patches
        @repository = repository
      end

      def commit
        start_point = repository.find_branch(branch_name)&.target || repository.root_ref

        OperationService.new(user, repository).with_branch(branch_name) do
          repository.commit_patches(start_point, patches, extra_env: user.git_env)
        end
      end
    end
  end
end
