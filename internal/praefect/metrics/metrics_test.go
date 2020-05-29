package metrics

import "testing"

func TestStorageGauge(t *testing.T) {
	sg := newStorageGauge("test")
	// the following will panic if the number of labels is wrong:
	sg.Inc("storage-1", "gitaly-1")
	sg.Dec("storage-1", "gitaly-1")
}
