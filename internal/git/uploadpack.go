package git

import "gitlab.com/gitlab-org/gitaly/internal/gitalyssh"

// UploadPackFilterConfig confines config options that are required to allow
// partial-clone filters.
func UploadPackFilterConfig() []Option {
	return []Option{
		ValueFlag{"-c", "uploadpack.allowFilter=true"},
		ValueFlag{"-c", gitalyssh.EnvVarUploadPackAllowAnySHA1InWant},
	}
}
