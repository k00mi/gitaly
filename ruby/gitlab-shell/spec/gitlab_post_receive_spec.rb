# coding: utf-8
require 'spec_helper'
require 'gitlab_post_receive'

describe GitlabPostReceive do
  let(:repository_path) { "/home/git/repositories" }
  let(:repo_name) { 'dzaporozhets/gitlab-ci' }
  let(:actor) { 'key-123' }
  let(:changes) { "123456 789012 refs/heads/tést\n654321 210987 refs/tags/tag" }
  let(:wrongly_encoded_changes) { changes.encode("ISO-8859-1").force_encoding("UTF-8") }
  let(:base64_changes) { Base64.encode64(wrongly_encoded_changes) }
  let(:repo_path) { File.join(repository_path, repo_name) + ".git" }
  let(:gl_repository) { "project-1" }
  let(:push_options) { [] }
  let(:output_stream) { double('output_stream') }
  let(:gitlab_post_receive) { GitlabPostReceive.new(gl_repository, repo_path, actor, wrongly_encoded_changes, push_options, output_stream) }
  let(:broadcast_message) { "test " * 10 + "message " * 10 }
  let(:enqueued_at) { Time.new(2016, 6, 23, 6, 59) }
  let(:new_merge_request_message) do
    "To create a merge request for new_branch, visit:\n  http://localhost/dzaporozhets/gitlab-ci/merge_requests/new?merge_request%5Bsource_branch%5D=new_branch"
  end
  let(:existing_merge_request_message) do
    "View merge request for feature_branch:\n  http://localhost/dzaporozhets/gitlab-ci/merge_requests/1"
  end

  before do
    $logger = double('logger').as_null_object # Global vars are bad
    allow_any_instance_of(GitlabConfig).to receive(:repos_path).and_return(repository_path)
    expect_any_instance_of(GitlabNet).to receive(:post_receive).and_return(response)
  end

  describe "#exec" do
    let(:response) { { 'reference_counter_decreased' => true } }

    subject { gitlab_post_receive.exec }

    it 'calls the api to notify the execution of the hook' do
      expect(subject).to eq(true)
    end

    context 'messages' do
      let(:response) do
        {
          'reference_counter_decreased' => true,
          'messages' => [
            { 'type' => 'alert', 'message' => broadcast_message },
            { 'type' => 'basic', 'message' => new_merge_request_message },
          ]
        }
      end

      it 'prints the merge request message and broadcast message' do
        assert_broadcast_message_printed
        assert_basic_message(new_merge_request_message)

        expect(subject).to eq(true)
      end

      context 'when contains long url string at end' do
        let(:broadcast_message) { "test " * 10 + "message " * 10 + "https://localhost:5000/test/a/really/long/url/that/is/in/the/broadcast/message/do-not-truncate-when-url" }

        it 'doesnt truncate url' do
          assert_broadcast_message_printed_keep_long_url_end
          assert_basic_message(new_merge_request_message)

          expect(subject).to eq(true)
        end
      end

      context 'when contains long url string at start' do
        let(:broadcast_message) { "https://localhost:5000/test/a/really/long/url/that/is/in/the/broadcast/message/do-not-truncate-when-url " + "test " * 10 + "message " * 11}

        it 'doesnt truncate url' do
          assert_broadcast_message_printed_keep_long_url_start
          assert_basic_message(new_merge_request_message)

          expect(subject).to eq(true)
        end
      end

      context 'when contains long url string in middle' do
        let(:broadcast_message) { "test " * 11 + "https://localhost:5000/test/a/really/long/url/that/is/in/the/broadcast/message/do-not-truncate-when-url " + "message " * 11}

        it 'doesnt truncate url' do
          assert_broadcast_message_printed_keep_long_url_middle
          assert_basic_message(new_merge_request_message)

          expect(subject).to eq(true)
        end
      end
    end

    context 'when warnings are present' do
      let(:response) do
        {
          'reference_counter_decreased' => true,
          'messages' => [
            { 'type' => 'alert', 'message' => "WARNINGS:\nMy warning message" }
          ]
        }
      end
      let(:output_stream) { $stdout }

      it 'treats the warning as a broadcast message' do
        assert_warning_looks_like_broadcast_message
        expect(subject).to eq(true)
      end
    end

    context 'when redirected message available' do
      let(:response) do
        {
          'reference_counter_decreased' => true,
          'messages' => [
            { 'type' => 'basic', 'message' => "This is a redirected message" }
          ]
        }
      end

      it 'prints redirected message' do
        assert_first_newline
        assert_basic_message("This is a redirected message")
        expect(subject).to eq(true)
      end
    end

    context 'when project created message is available' do
      let(:response) do
        {
          'reference_counter_decreased' => true,
          'messages' => [
            { 'type' => 'basic', 'message' => "This is a created project message" }
          ]
        }
      end

      it 'prints project created message' do
        assert_first_newline
        assert_basic_message("This is a created project message")

        expect(subject).to be true
      end
    end

    context 'when there are zero messages' do
      let(:response) do
        {
          'reference_counter_decreased' => true,
          'messages' => []
        }
      end

      it 'does not print anything' do
        expect(output_stream).to_not receive(:puts)

        expect(subject).to be true
      end
    end

    context 'when there is no messages parameter' do
      let(:response) do
        {
          'reference_counter_decreased' => true
        }
      end

      it 'does not print anything' do
        expect(output_stream).to_not receive(:puts)

        expect(subject).to be true
      end
    end

    # Deprecated message format for backwards-compatibility
    context 'deprecated message format' do
      let(:new_merge_request_urls) do
        [{
          'branch_name' => 'new_branch',
          'url' => 'http://localhost/dzaporozhets/gitlab-ci/merge_requests/new?merge_request%5Bsource_branch%5D=new_branch',
          'new_merge_request' => true
        }]
      end

      let(:response) do
        {
          'merge_request_urls' => new_merge_request_urls,
          'broadcast_message' => broadcast_message
        }
      end

      def assert_new_mr_printed
        expect_puts "To create a merge request for new_branch, visit:"
        expect_puts "  http://localhost/dzaporozhets/gitlab-ci/merge_requests/new?merge_request%5Bsource_branch%5D=new_branch"
        expect_puts
      end

      it 'prints the merge request urls and broadcast message' do
        assert_broadcast_message_printed
        assert_new_mr_printed

        subject
      end

      context 'when contains long url string at end' do
        let(:broadcast_message) { "test " * 10 + "message " * 10 + "https://localhost:5000/test/a/really/long/url/that/is/in/the/broadcast/message/do-not-truncate-when-url" }

        it 'doesnt truncate url' do
          assert_broadcast_message_printed_keep_long_url_end
          assert_new_mr_printed

          subject
        end
      end

      context 'when contains long url string at start' do
        let(:broadcast_message) { "https://localhost:5000/test/a/really/long/url/that/is/in/the/broadcast/message/do-not-truncate-when-url " + "test " * 10 + "message " * 11}

        it 'doesnt truncate url' do
          assert_broadcast_message_printed_keep_long_url_start
          assert_new_mr_printed

          subject
        end
      end

      context 'when contains long url string in middle' do
        let(:broadcast_message) { "test " * 11 + "https://localhost:5000/test/a/really/long/url/that/is/in/the/broadcast/message/do-not-truncate-when-url " + "message " * 11}

        it 'doesnt truncate url' do
          assert_broadcast_message_printed_keep_long_url_middle
          assert_new_mr_printed

          subject
        end
      end
    end

    context 'when warnings are present' do
      let(:response) do
        {
          'warnings' => 'My warning message'
        }
      end

      it 'treats the warning as a broadcast message' do
        assert_warning_looks_like_broadcast_message

        subject
      end
    end

    context 'when redirected message available' do
      let(:message) { "This is a redirected message" }
      let(:response) do
        {
          'redirected_message' => message
        }
      end

      it 'prints redirected message' do
        expect_puts "This is a redirected message"

        subject
      end

      context 'when project created message is available' do
        let(:message) { "This is a created project message" }
        let(:response) do
          {
            'project_created_message' => message
          }
        end

        it 'prints project created message' do
          expect_puts "This is a created project message"

          subject
        end
      end
    end
  end

  private

  # Red herring: If you see a test failure like this, it is more likely that the
  # content of one of the puts does not match. It probably does not mean the
  # puts are out of order.
  #
  # Failure/Error: puts message
  #   #<GitlabPostReceive:0x00007f97e0843008 ...<snip>... received :puts out of order
  def assert_broadcast_message_printed
    expect_puts
    expect_puts "========================================================================"
    expect_puts
    expect_puts "   test test test test test test test test test test message message"
    expect_puts "    message message message message message message message message"
    expect_puts
    expect_puts "========================================================================"
    expect_puts
  end

  def assert_broadcast_message_printed_keep_long_url_end
    expect_puts
    expect_puts "========================================================================"
    expect_puts
    expect_puts "   test test test test test test test test test test message message"
    expect_puts "    message message message message message message message message"
    expect_puts "https://localhost:5000/test/a/really/long/url/that/is/in/the/broadcast/message/do-not-truncate-when-url"
    expect_puts
    expect_puts "========================================================================"
    expect_puts
  end

  def assert_broadcast_message_printed_keep_long_url_start
    expect_puts
    expect_puts "========================================================================"
    expect_puts
    expect_puts "https://localhost:5000/test/a/really/long/url/that/is/in/the/broadcast/message/do-not-truncate-when-url"
    expect_puts "   test test test test test test test test test test message message"
    expect_puts "    message message message message message message message message"
    expect_puts "                                message"
    expect_puts
    expect_puts "========================================================================"
    expect_puts
  end

  def assert_broadcast_message_printed_keep_long_url_middle
    expect_puts
    expect_puts "========================================================================"
    expect_puts
    expect_puts "         test test test test test test test test test test test"
    expect_puts "https://localhost:5000/test/a/really/long/url/that/is/in/the/broadcast/message/do-not-truncate-when-url"
    expect_puts "    message message message message message message message message"
    expect_puts "                        message message message"
    expect_puts
    expect_puts "========================================================================"
    expect_puts
  end

  def assert_warning_looks_like_broadcast_message
    expect_puts
    expect_puts "========================================================================"
    expect_puts
    expect_puts "                      WARNINGS:\nMy warning message"
    expect_puts
    expect_puts "========================================================================"
    expect_puts
  end

  def assert_basic_message(message)
    expect_puts message
    expect_puts
  end

  def assert_first_newline
    expect_puts
  end

  def expect_puts(*args)
    if (args).none?
      expect(output_stream).to receive(:puts).ordered
    else
      expect(output_stream).to receive(:puts).with(*args).ordered
    end
  end
end
