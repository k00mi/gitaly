module Gitlab
  module Git
    class Index
      IndexError = Class.new(StandardError)

      DEFAULT_MODE = 0o100644
      EXECUTE_MODE = 0o100755

      ACTIONS = %w(create create_dir update move delete chmod).freeze
      ACTION_OPTIONS = %i(
        file_path previous_path content encoding execute_filemode infer_content
      ).freeze

      attr_reader :repository, :raw_index

      def initialize(repository)
        @repository = repository
        @raw_index = repository.rugged.index
      end

      delegate :read_tree, :get, to: :raw_index

      def apply(action, options)
        validate_action!(action)
        public_send(action, options.slice(*ACTION_OPTIONS))
      end

      def write_tree
        raw_index.write_tree(repository.rugged)
      end

      def dir_exists?(path)
        raw_index.find { |entry| entry[:path].start_with?("#{path}/") }
      end

      def create(options)
        options = normalize_options(options)

        raise IndexError, "A file with this name already exists" if get(options[:file_path])

        add_blob(options)
      end

      def create_dir(options)
        options = normalize_options(options)

        raise IndexError, "A file with this name already exists" if get(options[:file_path])

        raise IndexError, "A directory with this name already exists" if dir_exists?(options[:file_path])

        options = options.dup
        options[:file_path] += '/.gitkeep'
        options[:content] = ''

        add_blob(options)
      end

      def update(options)
        options = normalize_options(options)

        file_entry = get(options[:file_path])
        raise IndexError, "A file with this name doesn't exist" unless file_entry

        add_blob(options, mode: file_entry[:mode])
      end

      def move(options)
        options = normalize_options(options)

        old_entry = get(options[:previous_path])
        raise IndexError, "A file with this name doesn't exist" unless old_entry

        new_entry = get(options[:file_path])
        raise IndexError, "A file with this name already exists" if new_entry

        raw_index.remove(options[:previous_path])

        if options[:infer_content]
          raw_index.add(path: options[:file_path], oid: old_entry[:oid], mode: old_entry[:mode])
        else
          add_blob(options, mode: old_entry[:mode])
        end
      end

      def delete(options)
        options = normalize_options(options)

        raise IndexError, "A file with this name doesn't exist" unless get(options[:file_path])

        raw_index.remove(options[:file_path])
      end

      def chmod(options)
        options = normalize_options(options)

        file_entry = get(options[:file_path])
        raise IndexError, "A file with this name doesn't exist" unless file_entry

        mode = options[:execute_filemode] ? EXECUTE_MODE : DEFAULT_MODE

        raw_index.add(path: options[:file_path], oid: file_entry[:oid], mode: mode)
      end

      private

      def normalize_options(options)
        options = options.dup
        options[:file_path] = normalize_path(options[:file_path]) if options[:file_path]
        options[:previous_path] = normalize_path(options[:previous_path]) if options[:previous_path]
        options
      end

      def normalize_path(path)
        raise IndexError, "You must provide a file path" unless path

        Gitlab::Git::PathHelper.normalize_path!(path.dup)
      rescue Gitlab::Git::PathHelper::InvalidPath => e
        raise IndexError, e.message
      end

      def add_blob(options, mode: nil)
        content = options[:content]
        raise IndexError, "You must provide content" unless content

        content = Base64.decode64(content) if options[:encoding] == 'base64'

        detect = CharlockHolmes::EncodingDetector.new.detect(content)
        unless detect && detect[:type] == :binary
          # When writing to the repo directly as we are doing here,
          # the `core.autocrlf` config isn't taken into account.
          content.gsub!("\r\n", "\n") if repository.autocrlf
        end

        oid = repository.rugged.write(content, :blob)

        raw_index.add(path: options[:file_path], oid: oid, mode: mode || DEFAULT_MODE)
      rescue Rugged::IndexError => e
        raise IndexError, e.message
      end

      def validate_action!(action)
        raise ArgumentError, "Unknown action '#{action}'" unless ACTIONS.include?(action.to_s)
      end
    end
  end
end
