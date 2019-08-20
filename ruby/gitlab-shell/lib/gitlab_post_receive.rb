require_relative 'gitlab_init'
require_relative 'gitlab_net'
require_relative 'gitlab_metrics'
require_relative 'message'
require 'json'
require 'base64'
require 'securerandom'

class GitlabPostReceive
  attr_reader :config, :gl_repository, :repo_path, :changes, :jid, :output_stream

  def initialize(gl_repository, repo_path, actor, changes, push_options, output_stream = $stdout)
    @config = GitlabConfig.new
    @gl_repository = gl_repository
    @repo_path = repo_path.strip
    @actor = actor
    @changes = changes
    @push_options = push_options
    @jid = SecureRandom.hex(12)
    @output_stream = output_stream
  end

  def exec
    response = GitlabMetrics.measure("post-receive") do
      api.post_receive(gl_repository, @actor, changes, @push_options)
    end

    return false unless response

    # Deprecated message format for backwards-compatibility
    print_gitlab_12_2_messages(response)

    print_messages(response['messages'])

    response['reference_counter_decreased']
  rescue GitlabNet::ApiUnreachableError
    false
  end

  protected

  def api
    @api ||= GitlabNet.new
  end

  # Deprecated message format for backwards-compatibility
  def print_gitlab_12_2_messages(response)
    if response['broadcast_message']
      puts
      print_alert(response['broadcast_message'])
    end

    print_merge_request_links(response['merge_request_urls']) if response['merge_request_urls']
    puts response['redirected_message'] if response['redirected_message']
    puts response['project_created_message'] if response['project_created_message']

    if response['warnings']
      puts
      print_warnings(response['warnings'])
    end
  end

  # Deprecated message format for backwards-compatibility
  def print_merge_request_links(merge_request_urls)
    return if merge_request_urls.empty?
    puts
    merge_request_urls.each { |mr| print_merge_request_link(mr) }
  end

  # Deprecated message format for backwards-compatibility
  def print_merge_request_link(merge_request)
    message =
      if merge_request["new_merge_request"]
        "To create a merge request for #{merge_request['branch_name']}, visit:"
      else
        "View merge request for #{merge_request['branch_name']}:"
      end

    puts message
    puts((" " * 2) + merge_request["url"])
    puts
  end

  # Deprecated message format for backwards-compatibility
  def print_warnings(warnings)
    message = "WARNINGS:\n#{warnings}"
    print_alert(message)
    puts
  end

  def print_alert(message)
    Message.new('alert', message).print(output_stream)
  end

  def print_messages(response_messages)
    return if response_messages.nil? || response_messages.none?

    puts

    response_messages.each do |response_message|
      Message.new(response_message['type'], response_message['message']).print(output_stream)

      puts
    end
  end

  private

  def puts(*args)
    output_stream.puts(*args)
  end
end
