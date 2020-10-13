package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

// CheckInfo represents the response of GitLabs `check` API endpoint
type CheckInfo struct {
	// GitLab Server version
	Version string `json:"gitlab_version"`
	// Revision of the Git object of the running GitLab
	Revision string `json:"gitlab_revision"`
	// The version of the API, expected to be v4
	APIVersion string `json:"api_version"`
	// GitLab needs a working Redis, even if the check result is successful
	// GitLab might still not be able to handle hook API calls without it
	RedisReachable bool `json:"redis"`
}

// Check performs an HTTP request to the internal/check API endpoint to verify
// the connection and tokens. It returns basic information of the installed
// GitLab
func (a *gitlabAPI) Check(ctx context.Context) (*CheckInfo, error) {
	resp, err := a.client.Get(ctx, "/check")
	if err != nil {
		return nil, fmt.Errorf("HTTP GET to GitLab endpoint /check failed: %w", err)
	}

	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Check HTTP request failed with status: %d", resp.StatusCode)
	}

	var info CheckInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response from /check endpoint: %w", err)
	}

	return &info, nil
}

func (m *GitLabHookManager) Check(ctx context.Context) (*CheckInfo, error) {
	return m.gitlabAPI.Check(ctx)
}
