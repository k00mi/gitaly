require_relative 'gitlab_init'
require_relative 'gitlab_net'
require_relative 'gitlab_metrics'
require 'json'
require 'base64'
require 'securerandom'

class GitlabPostReceive
  # A standard terminal window is (at least) 80 characters wide.
  TERMINAL_WIDTH = 80
  GIT_REMOTE_MESSAGE_PREFIX_LENGTH = "remote: ".length
  TERMINAL_MESSAGE_PADDING = 2

  # Git prefixes remote messages with "remote: ", so this width is subtracted
  # from the width available to us.
  MAX_MESSAGE_WIDTH = TERMINAL_WIDTH - GIT_REMOTE_MESSAGE_PREFIX_LENGTH

  # Our centered text shouldn't start or end right at the edge of the window,
  # so we add some horizontal padding: 2 chars on either side.
  MAX_MESSAGE_TEXT_WIDTH = MAX_MESSAGE_WIDTH - 2 * TERMINAL_MESSAGE_PADDING

  attr_reader :config, :gl_repository, :repo_path, :changes, :jid

  def initialize(gl_repository, repo_path, actor, changes, push_options)
    @config = GitlabConfig.new
    @gl_repository = gl_repository
    @repo_path = repo_path.strip
    @actor = actor
    @changes = changes
    @push_options = push_options
    @jid = SecureRandom.hex(12)
  end

  def exec
    response = GitlabMetrics.measure("post-receive") do
      api.post_receive(gl_repository, @actor, changes, @push_options)
    end

    return false unless response

    print_messages(response['messages'])

    response['reference_counter_decreased']
  rescue GitlabNet::ApiUnreachableError
    false
  end

  protected

  def api
    @api ||= GitlabNet.new
  end

  def print_messages(response_messages)
    return if response_messages.nil? || response_messages.none?

    puts

    response_messages.each do |message|
      print_message(message)
      puts
    end
  end

  def print_message(message)
    case message['type']
    when 'alert'
      print_alert(message['message'])
    when 'basic'
      print_basic(message['message'])
    end
  end

  def print_basic(str)
    puts str
  end

  def print_alert(message)
    # Automatically wrap message at MAX_MESSAGE_TEXT_WIDTH (= 68) characters:
    # Splits the message up into the longest possible chunks matching
    # "<between 0 and MAX_MESSAGE_TEXT_WIDTH characters><space or end-of-line>".

    msg_start_idx = 0
    lines = []
    while msg_start_idx < message.length
      parsed_line = parse_alert_message(message[msg_start_idx..-1], MAX_MESSAGE_TEXT_WIDTH)
      msg_start_idx += parsed_line.length
      lines.push(parsed_line.strip)
    end

    puts "=" * MAX_MESSAGE_WIDTH
    puts

    lines.each do |line|
      line.strip!

      # Center the line by calculating the left padding measured in characters.
      line_padding = [(MAX_MESSAGE_WIDTH - line.length) / 2, 0].max
      puts((" " * line_padding) + line)
    end

    puts
    puts "=" * MAX_MESSAGE_WIDTH
  end

  private

  def parse_alert_message(msg, text_length)
    msg ||= ""
    # just return msg if shorter than or equal to text length
    return msg if msg.length <= text_length

    # search for word break shorter than text length
    truncate_to_space = msg.match(/\A(.{,#{text_length}})(?=\s|$)(\s*)/).to_s

    if truncate_to_space.empty?
      # search for word break longer than text length
      truncate_to_space = msg.match(/\A\S+/).to_s
    end

    truncate_to_space
  end
end
