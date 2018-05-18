require 'yaml'
require 'json'

fail("Please provide a MR description") if gitlab.mr_body.empty?

def check_changelog(path)
  if git.modified_files.include?("CHANGELOG.md")
    fail("CHANGELOG.md was edited. Please remove the additions and create an entry with _support/changelog")
    return
  end

  if !git.added_files.include?(path)
    warn("No changelog entry was generated, please do so by executing _support/changelog")
  else
    yaml = YAML.safe_load(yaml)

    unless yaml['merge_request'] == gitlab.mr["iid"]
      fail("Merge request ID was set to #{yaml['merge_request']}, expected #{gitlab.mr['iid']}")
    end

    unless yaml['title'] == gitlab.mr_title
      fail('Changelog entry should match the MR title')
    end
  end
end

check_changelog(File.join('changelogs', 'unreleased', "#{gitlab.branch_for_head}.yml}"))

VENDOR_JSON = 'vendor/vendor.json'
fail("Expected #{VENDOR_JSON} to exist") unless File.exist?(VENDOR_JSON)

if git.modified_files.include?(VENDOR_JSON)
  parsed_json = JSON.parse(File.read(VENDOR_JSON))

  proto = parsed_json["package"]&.find { |h| h["path"].start_with?("gitlab.com/gitlab-org/gitaly-proto") }

  unless proto["version"] && proto["version"] =~ /\Av\d+\./
    fail("gitaly-proto version is incorrect")
  end
end
