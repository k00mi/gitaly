require 'logger'

module Rails
  LOGGER = Logger.new(STDOUT)
  LOGGER.level = 'info'

  def self.logger
    LOGGER
  end
end
