require 'spec_helper'

describe Gitlab::Git::Repository do
  include TestRepo

  let(:repository) { gitlab_git_from_gitaly(test_repo_read_only) }

  describe '#from_gitaly_with_block' do
    let(:call_metadata) do
      {
        'user-agent' => 'grpc-go/1.9.1',
        'gitaly-storage-path' => DEFAULT_STORAGE_DIR,
        'gitaly-repo-path' => TEST_REPO_PATH,
        'gitaly-gl-repository' => 'project-52',
        'gitaly-repo-alt-dirs' => ''
      }
    end
    let(:call) { double(metadata: call_metadata) }
    let(:gitlab_shell_path) { '/foo/bar/gitlab-shell' }

    before do
      ENV['GITALY_RUBY_GITLAB_SHELL_PATH'] = gitlab_shell_path
    end

    after do
      ENV.delete('GITALY_RUBY_GITLAB_SHELL_PATH')
    end

    it 'cleans up the repository' do
      described_class.from_gitaly_with_block(test_repo_read_only, call) do |repository|
        expect(repository.rugged).to receive(:close)
      end
    end
  end

  describe '#cleanup' do
    context 'when Rugged has been called' do
      it 'calls close on Rugged::Repository' do
        rugged = repository.rugged

        expect(rugged).to receive(:close).and_call_original

        repository.cleanup
      end
    end

    context 'when Rugged has not been called' do
      it 'does not call close on Rugged::Repository' do
        expect(repository).not_to receive(:rugged)

        repository.cleanup
      end
    end
  end

  describe '#parse_raw_diff_line' do
    let(:diff_data) { repository.parse_raw_diff_line(diff_line) }

    context 'valid diff line' do
      let(:diff_line) { ":100644 100644 454bade 2b75299 M\tmodified-file.txt" }

      it 'returns the diff data' do
        expect(diff_data).to eq(["100644", "100644", "2b75299", "M\tmodified-file.txt"])
      end

      context 'added file' do
        let(:diff_line) { ":000000 100644 0000000 5579569 A\tnew-file.txt" }

        it 'returns the new blob id' do
          expect(diff_data[2]).to eq('5579569')
        end
      end

      context 'deleted file' do
        let(:diff_line) { ":100644 000000 26b5bd5 0000000 D\tremoved-file.txt" }

        it 'returns the old blob id' do
          expect(diff_data[2]).to eq('26b5bd5')
        end
      end
    end

    context 'invalid diff line' do
      let(:diff_line) { '' }

      it 'raises an ArgumentError' do
        expect { diff_data }.to raise_error(ArgumentError)
      end
    end
  end
end
