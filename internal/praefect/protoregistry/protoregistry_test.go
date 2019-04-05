package protoregistry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPopulatesProtoRegistry(t *testing.T) {
	r := New()
	require.NoError(t, r.RegisterFiles(GitalyProtoFileDescriptors...))
}
