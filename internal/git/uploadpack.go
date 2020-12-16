package git

// UploadPackFilterConfig confines config options that are required to allow
// partial-clone filters.
func UploadPackFilterConfig() []GlobalOption {
	return []GlobalOption{
		ConfigPair{Key: "uploadpack.allowFilter", Value: "true"},
		// Enables the capability to request individual SHA1's from the
		// remote repo.
		ConfigPair{Key: "uploadpack.allowAnySHA1InWant", Value: "true"},
	}
}
