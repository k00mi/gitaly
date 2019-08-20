package testhelper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"google.golang.org/grpc"
)

func TestSetCtxGrpcMethod(t *testing.T) {
	expectFullMethodName := "/pinkypb/TakeOverTheWorld.SNARF"
	ctx := testhelper.SetCtxGrpcMethod(context.Background(), expectFullMethodName)

	actualFullMethodName, ok := grpc.Method(ctx)
	require.True(t, ok, "expected context to contain server transport stream")
	require.Equal(t, expectFullMethodName, actualFullMethodName)
}
