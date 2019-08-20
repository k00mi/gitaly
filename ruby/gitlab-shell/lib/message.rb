class Message
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

  def initialize(type, message)
    @type = type
    @message = message
  end

  def print(output_stream = $stdout)
    @output_stream = output_stream

    case @type
    when 'alert'
      print_alert
    when 'basic'
      print_basic
    else
      raise "Unknown message type #{@type}"
    end
  end

  private

  def print_basic
    puts @message
  end

  def print_alert
    # Automatically wrap message at MAX_MESSAGE_TEXT_WIDTH (= 68) characters:
    # Splits the message up into the longest possible chunks matching
    # "<between 0 and MAX_MESSAGE_TEXT_WIDTH characters><space or end-of-line>".

    msg_start_idx = 0
    lines = []
    while msg_start_idx < @message.length
      parsed_line = parse_alert_message(@message[msg_start_idx..-1], MAX_MESSAGE_TEXT_WIDTH)
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

  def puts(*args)
    @output_stream.puts(*args)
  end
end
