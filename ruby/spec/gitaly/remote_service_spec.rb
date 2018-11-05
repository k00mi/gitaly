require 'spec_helper'

describe Gitaly::RemoteService do
  include IntegrationClient
  include TestRepo

  subject { gitaly_stub(:RemoteService) }

  describe 'UpdateRemoteMirror' do
    let(:call) { double(metadata: { 'gitaly-storage-path' => '/path/to/storage' }) }
    let(:repo) { gitaly_repo('default', 'foobar.git') }
    let(:remote) { 'my-remote' }

    context 'request does not have ssh_key and known_hosts set' do
      it 'performs the mirroring update with an empty environment' do
        request = Gitaly::UpdateRemoteMirrorRequest.new(repository: repo, ref_name: remote)

        allow(call).to receive(:each_remote_read).and_return(double(next: request, flat_map: []))
        allow(Gitlab::Git::Repository).to receive(:from_gitaly).and_return(repo)
        allow_any_instance_of(Gitlab::Git::RemoteMirror).to receive(:update)
        expect(Gitlab::Git::SshAuth).to receive(:new).with('', '')

        GitalyServer::RemoteService.new.update_remote_mirror(call)
      end
    end

    context 'request has ssh_key and known_hosts set' do
      it 'calls GitlabProjects#fetch_remote with a custom GIT_SSH_COMMAND' do
        request = Gitaly::UpdateRemoteMirrorRequest.new(
          repository: repo,
          ref_name: remote,
          ssh_key: 'SSH KEY',
          known_hosts: 'KNOWN HOSTS'
        )

        allow(call).to receive(:each_remote_read).and_return(double(next: request, flat_map: []))
        allow(Gitlab::Git::Repository).to receive(:from_gitaly).and_return(repo)
        allow_any_instance_of(Gitlab::Git::RemoteMirror).to receive(:update)
        expect(Gitlab::Git::SshAuth).to receive(:new).with('SSH KEY', 'KNOWN HOSTS')

        GitalyServer::RemoteService.new.update_remote_mirror(call)
      end
    end
  end
end
