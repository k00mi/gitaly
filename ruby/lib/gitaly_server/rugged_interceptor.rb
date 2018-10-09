# frozen_string_literal: true

require 'grpc'

module GitalyServer
  class RuggedInterceptor < GRPC::ServerInterceptor
    RUGGED_KEY = :rugged_list

    # Intercept a unary request response call
    %i[request_response server_streamer client_streamer bidi_streamer].each do |meth|
      define_method(meth) do |**, &blk|
        init_rugged_reference_list

        blk.call

        cleanup_rugged_references
      end
    end

    def init_rugged_reference_list
      Thread.current[RUGGED_KEY] = []
    end

    def cleanup_rugged_references
      repos = Thread.current[RUGGED_KEY]
      repos.compact.map(&:close)
    end
  end
end
