package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config is a container for everything found in the TOML config file
type Config struct {
	ListenAddr    string          `toml:"listen_addr" split_words:"true"`
	GitalyServers []*GitalyServer `toml:"gitaly_server", split_words:"true"`
}

// GitalyServer allows configuring the servers that RPCs are proxied to
type GitalyServer struct {
	Name       string `toml:"name"`
	ListenAddr string `toml:"listen_addr" split_words:"true"`
}

// FromFile loads the config for the passed file path
func FromFile(filePath string) (*Config, error) {
	cfgFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer cfgFile.Close()

	config := &Config{}
	_, err = toml.DecodeReader(cfgFile, config)
	return config, err
}

// Validate establishes if the config is valid
func (c *Config) Validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("no listen address configured")
	}

	if len(c.GitalyServers) == 0 {
		return fmt.Errorf("no gitaly backends configured")
	}

	listenAddrs := make(map[string]bool, len(c.GitalyServers))
	for _, gitaly := range c.GitalyServers {
		if gitaly.Name == "" {
			return fmt.Errorf("expect %q to have a name", gitaly)
		}

		if _, found := listenAddrs[gitaly.ListenAddr]; found {
			return fmt.Errorf("gitaly listen_addr: %s is not unique", gitaly.ListenAddr)
		}

		listenAddrs[gitaly.ListenAddr] = true
	}

	return nil
}
