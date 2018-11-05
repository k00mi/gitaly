require 'spec_helper'

describe Gitlab::Git::User do
  let(:username) { 'janedoe' }
  let(:name) { 'Jane Doé' }
  let(:email) { 'janedoé@example.com' }
  let(:gl_id) { 'user-123' }
  let(:user) do
    described_class.new(username, name, email, gl_id)
  end

  subject { described_class.new(username, name, email, gl_id) }

  describe '.from_gitaly' do
    let(:gitaly_user) do
      Gitaly::User.new(gl_username: username, name: name.b, email: email.b, gl_id: gl_id)
    end

    subject { described_class.from_gitaly(gitaly_user) }

    it { expect(subject).to eq(user) }
  end

  describe '#==' do
    def eq_other(username, name, email, gl_id)
      eq(described_class.new(username, name, email, gl_id))
    end

    it { expect(subject).to eq_other(username, name, email, gl_id) }

    it { expect(subject).not_to eq_other(nil, nil, nil, nil) }
    it { expect(subject).not_to eq_other(username + 'x', name, email, gl_id) }
    it { expect(subject).not_to eq_other(username, name + 'x', email, gl_id) }
    it { expect(subject).not_to eq_other(username, name, email + 'x', gl_id) }
    it { expect(subject).not_to eq_other(username, name, email, gl_id + 'x') }
  end

  describe '#git_env' do
    let(:git_env) { subject.git_env }

    it 'returns the user environment variables' do
      expect(git_env['GIT_COMMITTER_NAME']).to eq(name)
      expect(git_env['GIT_COMMITTER_EMAIL']).to eq(email)
      expect(git_env['GL_ID']).to eq(gl_id)
    end
  end
end
