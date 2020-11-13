package datastore

import (
	"context"
	"strings"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestDirectStorageProvider_GetSyncedNodes(t *testing.T) {
	getCtx := func() (context.Context, context.CancelFunc) {
		ctx, cancel := testhelper.Context()

		logger := testhelper.DiscardTestEntry(t)
		ctx = ctxlogrus.ToContext(ctx, logger)
		return ctx, cancel
	}

	t.Run("ok", func(t *testing.T) {
		ctx, cancel := getCtx()
		defer cancel()

		for _, tc := range []struct {
			desc string
			ret  map[string]struct{}
			exp  []string
		}{
			{
				desc: "primary included",
				ret:  map[string]struct{}{"g2": {}, "g3": {}},
				exp:  []string{"g1", "g2", "g3"},
			},
			{
				desc: "distinct values",
				ret:  map[string]struct{}{"g1": {}, "g2": {}, "g3": {}},
				exp:  []string{"g1", "g2", "g3"},
			},
			{
				desc: "none",
				ret:  nil,
				exp:  []string{"g1"},
			},
		} {
			t.Run(tc.desc, func(t *testing.T) {
				rs := &mockConsistentSecondariesProvider{}
				rs.On("GetConsistentSecondaries", ctx, "vs", "/repo/path", "g1").Return(tc.ret, nil)

				sp := NewDirectStorageProvider(rs)
				storages := sp.GetSyncedNodes(ctx, "vs", "/repo/path", "g1")
				require.ElementsMatch(t, tc.exp, storages)
			})
		}
	})

	t.Run("repository store returns an error", func(t *testing.T) {
		ctx, cancel := getCtx()
		defer cancel()

		logger := ctxlogrus.Extract(ctx)
		logHook := test.NewLocal(logger.Logger)

		rs := &mockConsistentSecondariesProvider{}
		rs.On("GetConsistentSecondaries", ctx, "vs", "/repo/path", "g1").
			Return(nil, assert.AnError).
			Once()

		sp := NewDirectStorageProvider(rs)

		storages := sp.GetSyncedNodes(ctx, "vs", "/repo/path", "g1")
		require.ElementsMatch(t, []string{"g1"}, storages)

		require.Len(t, logHook.AllEntries(), 1)
		require.Equal(t, "get consistent secondaries", logHook.LastEntry().Message)
		require.Equal(t, logrus.Fields{"error": assert.AnError}, logHook.LastEntry().Data)
		require.Equal(t, logrus.WarnLevel, logHook.LastEntry().Level)

		// "populate" metric is not set as there was an error and we don't want this result to be cached
		err := testutil.CollectAndCompare(sp, strings.NewReader(`
			# HELP gitaly_praefect_uptodate_storages_errors_total Total number of errors raised during defining up to date storages for reads distribution
			# TYPE gitaly_praefect_uptodate_storages_errors_total counter
			gitaly_praefect_uptodate_storages_errors_total{type="retrieve"} 1
		`))
		require.NoError(t, err)
	})
}

type mockConsistentSecondariesProvider struct {
	mock.Mock
}

func (m *mockConsistentSecondariesProvider) GetConsistentSecondaries(ctx context.Context, virtualStorage, relativePath, primary string) (map[string]struct{}, error) {
	args := m.Called(ctx, virtualStorage, relativePath, primary)
	val := args.Get(0)
	var res map[string]struct{}
	if val != nil {
		res = val.(map[string]struct{})
	}
	return res, args.Error(1)
}
