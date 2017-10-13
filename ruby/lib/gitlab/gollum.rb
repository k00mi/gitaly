module Gollum
  GIT_ADAPTER = "rugged".freeze
end
require "gollum-lib"

module Gollum
  class Committer
    # Patch for UTF-8 path
    def method_missing(name, *args) # rubocop:disable Style/MethodMissing
      index.send(name, *args) # rubocop:disable GitlabSecurity/PublicSend
    end
  end
end
