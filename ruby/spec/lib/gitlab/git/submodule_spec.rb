require 'spec_helper'

describe Gitlab::Git::Submodule do
  include TestRepo

  let(:repository) { gitlab_git_from_gitaly(new_mutable_test_repo) }
  let(:user) { Gitlab::Git::User.new('johndone', 'John Doe', 'johndoe@mail.com', 'user-1') }
  let(:message) { 'Updating submodule' }
  let(:branch) { 'master' }
  let(:new_oid) { 'db97db76ecd478eb361f439807438f82d97b29a5' }
  let(:submodule_path) { 'gitlab-grack' }

  subject do
    described_class.new(user, repository, submodule_path, branch).update(new_oid, message)
  end

  describe '#validate!' do
    context 'with repository' do
      let(:repository) { gitlab_git_from_gitaly(new_empty_test_repo) }

      it 'raises error if empty' do
        expect { subject }.to raise_error(Gitlab::Git::CommitError, 'Repository is empty')
      end
    end

    context 'with user' do
      let(:user) { nil }

      it 'raises error if not present' do
        expect { subject }.to raise_error(ArgumentError, 'User cannot be empty')
      end
    end

    context 'with submodule' do
      context 'when is not present' do
        let(:submodule_path) { '' }

        it 'raises error' do
          expect { subject }.to raise_error(ArgumentError, 'Submodule can not be empty')
        end
      end

      context 'when it has path traversal' do
        let(:submodule_path) { '../gitlab-grack' }

        it 'raises error' do
          expect { subject }.to raise_error(ArgumentError, 'Path cannot include directory traversal')
        end
      end
    end
  end

  describe '#update' do
    it 'updates the submodule oid' do
      update_commit = subject

      blob = repository.blob_at(update_commit.newrev, submodule_path)

      expect(repository.commit.id).to eq update_commit.newrev
      expect(blob.id).to eq new_oid
    end

    context 'when branch is not master' do
      let(:branch) { 'csv' }

      it 'updates submodule oid' do
        update_commit = subject

        blob = repository.blob_at(update_commit.newrev, submodule_path)

        expect(repository.commit(branch).id).to eq update_commit.newrev
        expect(blob.id).to eq new_oid
      end
    end

    context 'when submodule inside folder' do
      let(:branch) { 'submodule_inside_folder' }
      let(:submodule_path) { 'test_inside_folder/another_folder/six' }
      let(:new_oid) { 'e25eda1fece24ac7a03624ed1320f82396f35bd8' }

      it 'updates submodule oid' do
        update_commit = subject

        blob = repository.blob_at(update_commit.newrev, submodule_path)

        expect(repository.commit(branch).id).to eq update_commit.newrev
        expect(blob.id).to eq new_oid
      end
    end
  end
end
