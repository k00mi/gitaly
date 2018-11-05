require 'spec_helper'

describe Gitaly::RepositoryService do
  include IntegrationClient
  include TestRepo

  subject { gitaly_stub(:RepositoryService) }

  describe 'RepositoryExists' do
    it 'returns false if the repository does not exist' do
      request = Gitaly::RepositoryExistsRequest.new(repository: gitaly_repo('default', 'foobar.git'))
      response = subject.repository_exists(request)
      expect(response.exists).to eq(false)
    end

    it 'returns true if the repository exists' do
      request = Gitaly::RepositoryExistsRequest.new(repository: test_repo_read_only)
      response = subject.repository_exists(request)
      expect(response.exists).to eq(true)
    end
  end

  describe 'FetchRemote' do
    let(:call) { double(metadata: { 'gitaly-storage-path' => '/path/to/storage' }) }
    let(:repo) { gitaly_repo('default', 'foobar.git') }
    let(:remote) { 'my-remote' }

    let(:gl_projects) { double('Gitlab::Git::GitlabProjects') }

    before do
      allow(Gitlab::Git::GitlabProjects).to receive(:from_gitaly).and_return(gl_projects)
    end

    context 'request does not have ssh_key and known_hosts set' do
      it 'calls GitlabProjects#fetch_remote with an empty environment' do
        request = Gitaly::FetchRemoteRequest.new(repository: repo, remote: remote)

        expect(gl_projects).to receive(:fetch_remote)
          .with(remote, 0, force: false, tags: true, env: {})
          .and_return(true)

        GitalyServer::RepositoryService.new.fetch_remote(request, call)
      end
    end

    context 'request has ssh_key and known_hosts set' do
      it 'calls GitlabProjects#fetch_remote with a custom GIT_SSH_COMMAND' do
        request = Gitaly::FetchRemoteRequest.new(
          repository: repo,
          remote: remote,
          ssh_key: 'SSH KEY',
          known_hosts: 'KNOWN HOSTS'
        )

        expect(gl_projects).to receive(:fetch_remote)
          .with(remote, 0, force: false, tags: true, env: { 'GIT_SSH_COMMAND' => /ssh/ })
          .and_return(true)

        GitalyServer::RepositoryService.new.fetch_remote(request, call)
      end
    end
  end
end
