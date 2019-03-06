package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoConfigFlag(t *testing.T) {
	_, err := configure()

	assert.Equal(t, err, errNoConfigFile)
}
