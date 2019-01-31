package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	gitalySrvs := []*GitalyServer{&GitalyServer{"test", "localhost:23456"}}

	testCases := []struct {
		desc        string
		config      *Config
		expectError bool
	}{
		{
			desc:        "No ListenAddr",
			config:      &Config{"", gitalySrvs},
			expectError: true,
		},
		{
			desc:        "No servers",
			config:      &Config{"localhost:1234", nil},
			expectError: true,
		},
		{
			desc:        "duplicate address",
			config:      &Config{"localhost:1234", []*GitalyServer{gitalySrvs[0], gitalySrvs[0]}},
			expectError: true,
		},
		{
			desc:   "Valid config",
			config: &Config{"localhost:1234", gitalySrvs},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t1 *testing.T) {
			err := tc.config.Validate()
			if tc.expectError {
				require.Error(t1, err)
			} else {
				require.NoError(t1, err)
			}

		})
	}
}
