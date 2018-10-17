package hooks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath(t *testing.T) {
	assert.True(t, strings.HasSuffix(Path(), "hooks"))
}
