# Gitaly note: JV: no RPC's here.

module Gitlab
  module Git
    class PathHelper
      InvalidPath = Class.new(StandardError)

      class << self
        def normalize_path!(filename)
          return unless filename

          # Strip all leading slashes so that //foo -> foo
          filename = filename.sub(%r{\A/*}, '')

          # Expand relative paths (e.g. foo/../bar)
          filename = Pathname.new(filename)
          filename.relative_path_from(Pathname.new(''))

          filename.each_filename do |segment|
            raise InvalidPath, 'Path cannot include directory traversal' if segment == '..'
          end

          filename.to_s
        end
      end
    end
  end
end
