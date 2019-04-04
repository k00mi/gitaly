require_relative 'ref'

module Gitlab
  module Git
    class Branch < Ref
      def initialize(repository, name, target, target_commit)
        super(repository, name, target, target_commit)
      end
    end
  end
end
