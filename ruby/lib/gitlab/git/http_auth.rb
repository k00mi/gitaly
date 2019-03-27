module Gitlab
  module Git
    class HttpAuth
      def self.from_gitaly(request, call)
        params = request.remote_params

        return yield unless params.present?
        return yield unless params.http_authorization_header.present?

        repo = Gitlab::Git::Repository.from_gitaly(request.repository, call)

        validate_remote_params(params)

        key = "http.#{params.url}.extraHeader"
        repo.rugged.config[key] = "Authorization: #{params.http_authorization_header}"

        begin
          yield
        ensure
          repo.rugged.config.delete(key)
        end
      end

      def self.validate_remote_params(remote_params)
        begin
          URI.parse(remote_params.url)
        rescue URI::Error
          raise GRPC::InvalidArgument, 'invalid remote url'
        end
      end
    end
  end
end
