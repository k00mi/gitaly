require "spec_helper"

describe Gitlab::Git::Branch do
  include TestRepo

  let(:repository) { gitlab_git_from_gitaly(git_test_repo_read_only) }

  subject { repository.branches }

  it { is_expected.to be_an(Array) }

  describe '#size' do
    subject { super().size }

    it { is_expected.to eq(SeedRepo::Repo::BRANCHES.size) }
  end

  it { expect(repository.branches.size).to eq(SeedRepo::Repo::BRANCHES.size) }
end
