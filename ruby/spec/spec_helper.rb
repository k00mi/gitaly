require_relative '../lib/gitaly_server.rb'
require_relative '../lib/gitlab/git.rb'
require_relative 'support/sentry.rb'
require 'test_repo_helper'

ENV['GITALY_RUBY_GIT_BIN_PATH'] ||= 'git'
