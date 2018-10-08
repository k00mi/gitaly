require 'raven/base'
require 'raven/transports/dummy'

Raven.configure do |config|
  config.dsn = "dummy://12345:67890@sentry.localdomain:3000/sentry/42"
  config.encoding = 'json'
  config.logger = ::Logger.new(nil)
end
