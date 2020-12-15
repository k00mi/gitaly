package git

// UploadPackFilterConfig confines config options that are required to allow
// partial-clone filters.
func UploadPackFilterConfig() []GlobalOption {
	return []GlobalOption{
		ValueFlag{"-c", "uploadpack.allowFilter=true"},
		// Enables the capability to request individual SHA1's from the
		// remote repo.
		ValueFlag{"-c", "uploadpack.allowAnySHA1InWant=true"},
	}
}
