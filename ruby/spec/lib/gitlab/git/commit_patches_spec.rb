require 'spec_helper'

describe Gitlab::Git::CommitPatches do
  include TestRepo

  describe "#commit" do
    let(:repository) { gitlab_git_from_gitaly(new_mutable_test_repo) }
    let(:testdata_dir) { File.join(File.dirname(__FILE__), '../../../../../internal/service/operations/testdata') }
    let(:patches) { File.foreach(File.join(testdata_dir, patch_file_name)) }
    let(:user) { Gitlab::Git::User.new('jane', 'Jane Doe', 'jane@doe.org', '123') }

    def apply_patches(branch_name)
      described_class.new(user, repository, branch_name, patches).commit
    end

    context 'when the patch applies' do
      let(:patch_file_name) { '0001-A-commit-from-a-patch.patch' }

      it 'creates the branch and applies the patch' do
        branch_update = apply_patches('patched_branch')
        commit = repository.commit(branch_update.newrev)

        expect(commit.message).to eq("A commit from a patch\n")
        expect(branch_update).to be_branch_created
      end

      it 'updates the branch if it already existed' do
        branch_update = apply_patches('feature')
        commit = repository.commit(branch_update.newrev)

        expect(commit.message).to eq("A commit from a patch\n")
        expect(branch_update).not_to be_branch_created
      end
    end

    context 'when the patch does not apply' do
      let(:patch_file_name) { '0001-This-does-not-apply-to-the-feature-branch.patch' }

      it 'raises a PatchError' do
        expect { apply_patches('feature') }.to raise_error Gitlab::Git::PatchError
      end

      it 'does not update the branch' do
        expect do
          begin
            apply_patches('feature')
          rescue Gitlab::Git::PatchError => e
            e
          end
        end.not_to(change { repository.find_branch('feature').target })
      end

      it 'does not leave branches dangling' do
        expect do
          begin
            apply_patches('feature')
          rescue Gitlab::Git::PatchError => e
            e
          end
        end.not_to(change { repository.branches.size })
      end
    end
  end
end
