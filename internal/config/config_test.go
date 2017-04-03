package config

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func configFileReader(content string) io.Reader {
	return bytes.NewReader([]byte(content))
}

func TestLoadClearPrevConfig(t *testing.T) {
	Config = config{SocketPath: "/tmp"}
	err := Load(nil)
	assert.NoError(t, err)

	assert.Empty(t, Config.SocketPath)
}

func TestLoadBrokenConfig(t *testing.T) {
	tmpFile := configFileReader(`path = "/tmp"\nname="foo"`)
	err := Load(tmpFile)
	assert.Error(t, err)

	assert.Empty(t, Config.PrometheusListenAddr)
	assert.Empty(t, Config.SocketPath)
	assert.Empty(t, Config.ListenAddr)
	assert.Empty(t, Config.Storages)
}

func TestLoadEmptyConfig(t *testing.T) {
	tmpFile := configFileReader(``)

	err := Load(tmpFile)
	assert.NoError(t, err)

	assert.Empty(t, Config.PrometheusListenAddr)
	assert.Empty(t, Config.SocketPath)
	assert.Empty(t, Config.ListenAddr)
	assert.Empty(t, Config.Storages)
}

func TestLoadStorage(t *testing.T) {
	tmpFile := configFileReader(`[[storage]]
name = "default"
path = "/tmp"`)

	err := Load(tmpFile)
	assert.NoError(t, err)

	assert.Empty(t, Config.PrometheusListenAddr)
	assert.Empty(t, Config.SocketPath)
	assert.Empty(t, Config.ListenAddr)
	if assert.Equal(t, 1, len(Config.Storages), "Expected one (1) storage") {
		assert.Equal(t, "/tmp", Config.Storages[0].Path)
		assert.Equal(t, "default", Config.Storages[0].Name)
	}
}

func TestLoadMultiStorage(t *testing.T) {
	tmpFile := configFileReader(`[[storage]]
name="default"
path="/tmp/repos1"

[[storage]]
name="other"
path="/tmp/repos2"`)

	err := Load(tmpFile)
	assert.NoError(t, err)

	assert.Empty(t, Config.PrometheusListenAddr)
	assert.Empty(t, Config.SocketPath)
	assert.Empty(t, Config.ListenAddr)
	if assert.Equal(t, 2, len(Config.Storages), "Expected three storages") {
		assert.Equal(t, "/tmp/repos1", Config.Storages[0].Path)
		assert.Equal(t, "default", Config.Storages[0].Name)

		assert.Equal(t, "/tmp/repos2", Config.Storages[1].Path)
		assert.Equal(t, "other", Config.Storages[1].Name)
	}
}

func TestLoadPrometheus(t *testing.T) {
	tmpFile := configFileReader(`prometheus_listen_addr=":9236"`)

	err := Load(tmpFile)
	assert.NoError(t, err)

	assert.Equal(t, ":9236", Config.PrometheusListenAddr)
	assert.Empty(t, Config.ListenAddr)
	assert.Empty(t, Config.SocketPath)
	assert.Equal(t, 0, len(Config.Storages), "Expected zero (0) storage")
}

func TestLoadSocketPath(t *testing.T) {
	tmpFile := configFileReader(`socket_path="/tmp/gitaly.sock"`)

	err := Load(tmpFile)
	assert.NoError(t, err)

	assert.Empty(t, Config.PrometheusListenAddr)
	assert.Empty(t, Config.ListenAddr)
	assert.Equal(t, "/tmp/gitaly.sock", Config.SocketPath)
	assert.Equal(t, 0, len(Config.Storages), "Expected zero (0) storage")
}

func TestLoadListenAddr(t *testing.T) {
	tmpFile := configFileReader(`listen_addr=":8080"`)

	err := Load(tmpFile)
	assert.NoError(t, err)

	assert.Empty(t, Config.PrometheusListenAddr)
	assert.Empty(t, Config.SocketPath)
	assert.Equal(t, ":8080", Config.ListenAddr)
	assert.Equal(t, 0, len(Config.Storages), "Expected zero (0) storage")
}

func tempEnv(key, value string) func() {
	temp := os.Getenv(key)
	os.Setenv(key, value)

	return func() {
		os.Setenv(key, temp)
	}
}

func TestLoadOverrideEnvironment(t *testing.T) {
	// Test that this works since we still want this to work
	tempEnv1 := tempEnv("GITALY_SOCKET_PATH", "/tmp/gitaly2.sock")
	defer tempEnv1()
	tempEnv2 := tempEnv("GITALY_LISTEN_ADDR", ":8081")
	defer tempEnv2()
	tempEnv3 := tempEnv("GITALY_PROMETHEUS_LISTEN_ADDR", ":9237")
	defer tempEnv3()

	tmpFile := configFileReader(`socket_path = "/tmp/gitaly.sock"
listen_addr = ":8080"
prometheus_listen_addr = ":9236"`)

	err := Load(tmpFile)
	assert.NoError(t, err)

	assert.Equal(t, ":9237", Config.PrometheusListenAddr)
	assert.Equal(t, "/tmp/gitaly2.sock", Config.SocketPath)
	assert.Equal(t, ":8081", Config.ListenAddr)
}

func TestLoadOnlyEnvironment(t *testing.T) {
	// Test that this works since we still want this to work
	os.Setenv("GITALY_SOCKET_PATH", "/tmp/gitaly2.sock")
	os.Setenv("GITALY_LISTEN_ADDR", ":8081")
	os.Setenv("GITALY_PROMETHEUS_LISTEN_ADDR", ":9237")

	err := Load(nil)
	assert.NoError(t, err)

	assert.Equal(t, ":9237", Config.PrometheusListenAddr)
	assert.Equal(t, "/tmp/gitaly2.sock", Config.SocketPath)
	assert.Equal(t, ":8081", Config.ListenAddr)
}
