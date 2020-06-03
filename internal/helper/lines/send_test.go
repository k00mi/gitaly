package lines

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLinesSend(t *testing.T) {
	reader := bytes.NewBufferString("mepmep foo bar")

	var out [][]byte
	sender := func(in [][]byte) error { out = in; return nil }
	err := Send(reader, sender, SenderOpts{Delimiter: []byte(" ")})
	require.NoError(t, err)

	expected := [][]byte{
		[]byte("mepmep"),
		[]byte("foo"),
		[]byte("bar"),
	}

	require.Equal(t, expected, out)
}
