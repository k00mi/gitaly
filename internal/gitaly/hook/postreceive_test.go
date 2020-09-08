package hook

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintAlert(t *testing.T) {
	testCases := []struct {
		message  string
		expected string
	}{
		{
			message: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Curabitur nec mi lectus. Fusce eu ligula in odio hendrerit posuere. Ut semper neque vitae maximus accumsan. In malesuada justo nec leo congue egestas. Vivamus interdum nec libero ac convallis. Praesent euismod et nunc vitae vulputate. Mauris tincidunt ligula urna, bibendum vestibulum sapien luctus eu. Donec sed justo in erat dictum semper. Ut porttitor augue in felis gravida scelerisque. Morbi dolor justo, accumsan et nulla vitae, luctus consectetur est. Donec aliquet erat pellentesque suscipit elementum. Cras posuere eros ipsum, a tincidunt tortor laoreet quis. Mauris varius nulla vitae placerat imperdiet. Vivamus ut ligula odio. Cras nec euismod ligula.",
			expected: `========================================================================

   Lorem ipsum dolor sit amet, consectetur adipiscing elit. Curabitur
  nec mi lectus. Fusce eu ligula in odio hendrerit posuere. Ut semper
    neque vitae maximus accumsan. In malesuada justo nec leo congue
  egestas. Vivamus interdum nec libero ac convallis. Praesent euismod
    et nunc vitae vulputate. Mauris tincidunt ligula urna, bibendum
  vestibulum sapien luctus eu. Donec sed justo in erat dictum semper.
  Ut porttitor augue in felis gravida scelerisque. Morbi dolor justo,
  accumsan et nulla vitae, luctus consectetur est. Donec aliquet erat
      pellentesque suscipit elementum. Cras posuere eros ipsum, a
   tincidunt tortor laoreet quis. Mauris varius nulla vitae placerat
      imperdiet. Vivamus ut ligula odio. Cras nec euismod ligula.

========================================================================`,
		},
		{
			message: "Lorem ipsum dolor sit amet, consectetur",
			expected: `========================================================================

                Lorem ipsum dolor sit amet, consectetur

========================================================================`,
		},
	}

	for _, tc := range testCases {
		var result bytes.Buffer

		require.NoError(t, printAlert(PostReceiveMessage{Message: tc.message}, &result))
		assert.Equal(t, tc.expected, result.String())
	}
}
