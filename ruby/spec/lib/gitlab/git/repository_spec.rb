require 'spec_helper'

describe Gitlab::Git::RevList do
  include TestRepo

  let(:repository) { gitlab_git_from_gitaly(test_repo_read_only) }

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
        expect {diff_data }.to raise_error(ArgumentError)
      end
    end
  end
end
