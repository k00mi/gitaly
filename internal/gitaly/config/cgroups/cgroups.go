package cgroups

// Config is a struct for cgroups config
type Config struct {
	// Count is the number of cgroups to be created at startup
	Count uint `toml:"count"`
	// Mountpoint is where the cgroup filesystem is mounted, usually under /sys/fs/cgroup/
	Mountpoint string `toml:"mountpoint"`
	// HierarchyRoot is the parent cgroup under which Gitaly creates <Count> of cgroups.
	// A system administrator is expected to create such cgroup/directory under <Mountpoint>/memory
	// and/or <Mountpoint>/cpu depending on which resource is enabled. HierarchyRoot is expected to
	// be owned by the user and group Gitaly runs as.
	HierarchyRoot string `toml:"hierarchy_root"`
	// CPU holds CPU resource configurations
	CPU CPU `toml:"cpu"`
	// Memory holds memory resource configurations
	Memory Memory `toml:"memory"`
}

// Memory is a struct storing cgroups memory config
type Memory struct {
	Enabled bool `toml:"enabled"`
	// Limit is the memory limit in bytes. Could be -1 to indicate unlimited memory.
	Limit int64 `toml:"limit"`
}

// CPU is a struct storing cgroups CPU config
type CPU struct {
	Enabled bool `toml:"enabled"`
	// Shares is the number of CPU shares (relative weight (ratio) vs. other cgroups with CPU shares).
	Shares uint64 `toml:"shares"`
}
