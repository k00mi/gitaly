require 'grpc'
require 'raven/base'

require_relative 'sentry.rb'

# rubocop:disable Lint/RescueWithoutErrorClass
module GitalyServer
  class SentryInterceptor < GRPC::ServerInterceptor
    # Intercept a unary request response call
    def request_response(request: nil, call: nil, method: nil)
      start = Time.now
      yield
    rescue => e
      time_ms = Time.now - start
      handle_exception(e, call, method, time_ms)
    end

    # Intercept a server streaming call
    def server_streamer(request: nil, call: nil, method: nil)
      start = Time.now
      yield
    rescue => e
      time_ms = Time.now - start
      handle_exception(e, call, method, time_ms)
    end

    # Intercept a client streaming call
    def client_streamer(call: nil, method: nil)
      start = Time.now
      yield
    rescue => e
      time_ms = Time.now - start
      handle_exception(e, call, method, time_ms)
    end

    # Intercept a BiDi streaming call
    def bidi_streamer(requests: nil, call: nil, method: nil)
      start = Time.now
      yield
    rescue => e
      time_ms = Time.now - start
      handle_exception(e, call, method, time_ms)
    end

    private

    def handle_exception(exc, call, method, time_ms)
      raise exc unless GitalyServer::Sentry.enabled?

      grpc_method = "#{method.owner.name}##{method.name}"
      tags = {
        'system' => 'gitaly-ruby',
        'gitaly-ruby.method' => grpc_method,
        'gitaly-ruby.time_ms' => format("%.0f", (time_ms * 1000))
      }
      tags.merge!(call.metadata)

      exc_to_report = exc
      exc_to_report = exc.cause if exc.cause && exc.is_a?(GRPC::Unknown) && exc.metadata.key?(:"gitaly-ruby.exception.class")

      Raven.tags_context(tags)
      Raven.capture_exception(exc_to_report, fingerprint: ['gitaly-ruby', grpc_method, exc_to_report.message])

      raise exc
    end
  end
end
