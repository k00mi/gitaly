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

    it 'returns the passed result of the block passed' do
      result = described_class.from_gitaly_with_block(test_repo_read_only, call) { 'Hello world' }

      expect(result).to eq('Hello world')
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

  describe '#rugged' do
    after do
      Thread.current[described_class::RUGGED_KEY] = nil
    end

    it 'stores reference in Thread.current' do
      Thread.current[described_class::RUGGED_KEY] = []

      2.times do
        rugged = repository.rugged

        expect(rugged).to be_a(Rugged::Repository)
        expect(Thread.current[described_class::RUGGED_KEY]).to eq([rugged])
      end
    end

    it 'does not store reference if Thread.current is not set up' do
      rugged = repository.rugged

      expect(rugged).to be_a(Rugged::Repository)
      expect(Thread.current[described_class::RUGGED_KEY]).to be_nil
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

  describe "#commit_patches" do
    let(:repository) { gitlab_git_from_gitaly(new_mutable_test_repo) }
    let(:testdata_dir) { File.join(File.dirname(__FILE__), '../../../../../internal/service/operations/testdata') }
    let(:patches) { File.foreach(File.join(testdata_dir, patch_file_name)) }

    def apply_patches(branch_name)
      repository.commit_patches(branch_name, patches)
    end

    context 'when the patch applies' do
      let(:patch_file_name) { '0001-A-commit-from-a-patch.patch' }

      it 'creates a new rev with the patch' do
        new_rev = apply_patches(repository.root_ref)
        commit = repository.commit(new_rev)

        expect(new_rev).not_to be_nil
        expect(commit.message).to eq("A commit from a patch\n")
      end
    end

    context 'when the patch does not apply' do
      let(:patch_file_name) { '0001-This-does-not-apply-to-the-feature-branch.patch' }

      it 'raises a PatchError' do
        expect { apply_patches('feature') }.to raise_error Gitlab::Git::PatchError
      end
    end
  end
end
