package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainsPathTraversal(t *testing.T) {
	testCases := []struct {
		path              string
		containsTraversal bool
	}{
		{"../parent", true},
		{"subdir/../../parent", true},
		{"subdir/..", true},
		{"subdir", false},
		{"./subdir", false},
	}

	for _, tc := range testCases {
		assert.Equal(t, tc.containsTraversal, ContainsPathTraversal(tc.path))
	}
}
