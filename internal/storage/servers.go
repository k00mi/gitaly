package storage

// GitalyServers hold Gitaly servers info like {"default":{"token":"x","address":"y"}},
// to be passed in `gitaly-servers` metadata.
type GitalyServers map[string]map[string]string
