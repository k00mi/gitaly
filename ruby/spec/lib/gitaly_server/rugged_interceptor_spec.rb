require 'spec_helper'

require_relative '../../../lib/gitaly_server/rugged_interceptor.rb'

describe GitalyServer::RuggedInterceptor do
  include TestRepo

  let(:meth) { GitalyServer::RefService.instance_method(:create_branch) }
  let(:call) { double(metadata: {}) }

  subject do
    described_class.new.server_streamer(call: call, method: meth) { }
  end

  context 'no Rugged repositories initialized' do
    it 'does not clean up any repositories' do
      expect(Rugged::Repository).not_to receive(:new)

      subject
    end
  end

  context 'Rugged repository initialized' do
    let(:rugged) { rugged_from_gitaly(test_repo_read_only) }

    let(:streamer) do
      described_class.new.server_streamer(call: call, method: meth) do
        Thread.current[GitalyServer::RuggedInterceptor::RUGGED_KEY] = [rugged]
      end
    end

    it 'cleans up repositories' do
      expect(rugged).to receive(:close).and_call_original

      streamer
    end
  end
end
