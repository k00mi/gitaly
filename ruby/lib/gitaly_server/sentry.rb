require 'raven/base'

module GitalyServer
  module Sentry
    def self.enabled?
      ENV['SENTRY_DSN'].present?
    end
  end
end

Raven.configure do |config|
  config.release = ENV['GITALY_VERSION'].presence
  config.sanitize_fields = %w[gitaly-servers authorization]
end
