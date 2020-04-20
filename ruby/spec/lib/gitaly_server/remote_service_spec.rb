# frozen_string_literal: true

require 'spec_helper'

describe GitalyServer::RemoteService do
  describe '#update_remote_mirror' do
    it 'assigns a limited number of divergent refs' do
      stub_const("#{described_class}::DIVERGENT_REF_LIMIT", 2)

      mirror = double(
        divergent_refs: %w[refs/heads/master refs/heads/develop refs/heads/stable]
      ).as_null_object
      stub_const('Gitlab::Git::RemoteMirror', mirror)

      call = double(to_a: [], to_ary: []).as_null_object
      response = described_class.new.update_remote_mirror(call)

      expect(response.divergent_refs).to eq(%w[refs/heads/master refs/heads/develop])
    end
  end
end
