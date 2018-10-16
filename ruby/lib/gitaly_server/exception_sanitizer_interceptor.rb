require 'grpc'

module GitalyServer
  # We do this sanitization because the Go server reports gitaly-ruby exceptions
  # to Sentry as well, hence the sanitization. If we stopped sending gitaly-ruby
  # exceptions from Go then I think we can remove this interceptor.
  class ExceptionSanitizerInterceptor < GRPC::ServerInterceptor
    include GitalyServer::Utils
    %i[request_response server_streamer client_streamer bidi_streamer].each do |meth|
      define_method(meth) do |**, &blk|
        begin
          blk.call
        rescue => exc
          reraise_sanitized_exception(exc)
        end
      end
    end

    private

    def reraise_sanitized_exception(exc)
      sanitized_message = sanitize_url(exc.message)
      exc = exc.exception(sanitized_message)

      if exc.is_a?(GRPC::BadStatus)
        # Although GRPC::BadStatus is_a Exception, calling #exception on it
        # doesn't really change the message as it holds another message in its
        # @details, so we resort to this hacky approach to make sure the exception
        # is totally sanitized.
        exc.details.replace(sanitize_url(exc.details))
      end

      raise exc
    end
  end
end
