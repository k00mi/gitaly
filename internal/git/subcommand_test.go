package git

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSubcommand_mayGeneratePackfiles(t *testing.T) {
	require.True(t, mayGeneratePackfiles("gc"))
	require.False(t, mayGeneratePackfiles("apply"))
	require.False(t, mayGeneratePackfiles("nonexistent"))
}
