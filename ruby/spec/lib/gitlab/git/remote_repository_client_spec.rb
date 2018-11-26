require 'spec_helper'

describe Gitlab::Git::GitalyRemoteRepository do
  include TestRepo
  include IntegrationClient

  let(:repository) { gitlab_git_from_gitaly_with_gitlab_projects(new_mutable_test_repo) }
  describe 'Connectivity' do
    context 'tcp' do
      let(:client) do
        get_client("tcp://localhost:#{GitalyConfig.dynamic_port}")
      end

      it 'Should connect over tcp' do
        expect(client).not_to be_empty
      end
    end

    context 'unix' do
      let(:client) { get_client("unix:#{File.join(TMP_DIR_NAME, SOCKET_PATH)}") }

      it 'Should connect over unix' do
        expect(client).not_to be_empty
      end
    end
  end
end
