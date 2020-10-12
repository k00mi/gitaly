package git2go

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSerialization_SerializeTo(t *testing.T) {
	type testStruct struct {
		Contents string `json:"contents"`
	}

	var buf bytes.Buffer

	input := testStruct{
		Contents: "foobar",
	}
	err := serializeTo(&buf, &input)
	require.NoError(t, err)
	require.NotZero(t, buf.Len())

	var output testStruct
	err = deserialize(buf.String(), &output)
	require.NoError(t, err)
	require.Equal(t, input, output)
}
