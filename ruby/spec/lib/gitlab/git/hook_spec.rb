require 'spec_helper'

describe Gitlab::Git::Hook do
  include TestRepo

  describe '.directory' do
    it 'does not raise an KeyError' do
      expect { described_class.directory }.not_to raise_error
    end
  end

  describe '#trigger' do
    let!(:old_env) { ENV['GITALY_GIT_HOOKS_DIR'] }
    let(:tmp_dir) { Dir.mktmpdir }
    let(:hook_names) { %w[pre-receive post-receive update] }
    let(:repo) { gitlab_git_from_gitaly(test_repo_read_only) }

    before do
      hook_names.each do |f|
        path = File.join(tmp_dir, f)
        File.write(path, script)
        FileUtils.chmod("u+x", path)
      end

      ENV['GITALY_GIT_HOOKS_DIR'] = tmp_dir
    end

    after do
      FileUtils.remove_entry(tmp_dir)
      ENV['GITALY_GIT_HOOKS_DIR'] = old_env
    end

    context 'when the hooks are successful' do
      let(:script) { "#!/bin/sh\nexit 0\n" }

      it 'returns true' do
        hook_names.each do |hook|
          trigger_result = described_class.new(hook, repo)
                                          .trigger('1', 'admin', '0' * 40, 'a' * 40, 'master')

          expect(trigger_result.first).to be(true)
        end
      end
    end

    context 'when the hooks fail' do
      let(:script) { "#!/bin/sh\nexit 1\n" }

      it 'returns false' do
        hook_names.each do |hook|
          trigger_result = described_class.new(hook, repo)
                                          .trigger('1', 'admin', '0' * 40, 'a' * 40, 'master')

          expect(trigger_result.first).to be(false)
        end
      end
    end
  end
end
