unless git.modified_files.include?("CHANGELOG.md")
  warn("This MR is missing a CHANGLOG entry")
end

fail("Please provide a MR description") if gitlab.mr_body.empty?

VENDOR_JSON = 'vendor/vendor.json'
fail("Expected #{VENDOR_JSON} to exist") unless File.exist?(VENDOR_JSON)

if git.modified_files.include?(VENDOR_JSON)
  require 'json'
  parsed_json = JSON.parse(File.read(VENDOR_JSON))

  proto = parsed_json["package"]&.find { |h| h["path"].start_with?("gitlab.com/gitlab-org/gitaly-proto") }

  unless proto["version"] && proto["version"] =~ /\Av\d+\./
    fail("gitaly-proto version is incorrect")
  end
end
