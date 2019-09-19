package models

// Node describes an address that serves a storage
type Node struct {
	ID             int
	Storage        string `toml:"storage"`
	Address        string `toml:"address"`
	Token          string `toml:"token"`
	DefaultPrimary bool   `toml:"primary"`
}

// Repository describes a repository's relative path and its primary and list of secondaries
type Repository struct {
	ID           int
	RelativePath string
	Primary      Node
	Replicas     []Node
}
