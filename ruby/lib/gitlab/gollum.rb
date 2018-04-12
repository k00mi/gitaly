module Gollum
  GIT_ADAPTER = "rugged".freeze
end
require "gollum-lib"

Gollum::Page.per_page = 20 # Magic number from Kaminari.config.default_per_page
