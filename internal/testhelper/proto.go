package testhelper

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
)

func ProtoEqual(t testing.TB, expected proto.Message, actual proto.Message) {
	require.True(t, proto.Equal(expected, actual), "proto messages not equal\nexpected: %v\ngot:      %v", expected, actual)
}
