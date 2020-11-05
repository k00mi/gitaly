// +build postgres

package datastore

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	promclient "github.com/prometheus/client_golang/prometheus"
	promclientgo "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
)

func TestNewPostgresListener(t *testing.T) {
	for title, tc := range map[string]struct {
		opts      PostgresListenerOpts
		expErrMsg string
	}{
		"all set": {
			opts: PostgresListenerOpts{
				Addr:                 "stub",
				Channel:              "sting",
				MinReconnectInterval: time.Second,
				MaxReconnectInterval: time.Minute,
			},
		},
		"invalid option: address": {
			opts:      PostgresListenerOpts{Addr: ""},
			expErrMsg: "address is invalid",
		},
		"invalid option: channel": {
			opts:      PostgresListenerOpts{Addr: "stub", Channel: "  "},
			expErrMsg: "channel is invalid",
		},
		"invalid option: ping period": {
			opts:      PostgresListenerOpts{Addr: "stub", Channel: "stub", PingPeriod: -1},
			expErrMsg: "invalid ping period",
		},
		"invalid option: min reconnect period": {
			opts:      PostgresListenerOpts{Addr: "stub", Channel: "stub", MinReconnectInterval: 0},
			expErrMsg: "invalid min reconnect period",
		},
		"invalid option: max reconnect period": {
			opts:      PostgresListenerOpts{Addr: "stub", Channel: "stub", MinReconnectInterval: time.Second, MaxReconnectInterval: time.Millisecond},
			expErrMsg: "invalid max reconnect period",
		},
	} {
		t.Run(title, func(t *testing.T) {
			pgl, err := NewPostgresListener(tc.opts)
			if tc.expErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, pgl)
		})
	}
}

type mockListenHandler struct {
	OnNotification func(string)
	OnDisconnect   func()
	OnConnect      func()
}

func (mlh mockListenHandler) Notification(v string) {
	if mlh.OnNotification != nil {
		mlh.OnNotification(v)
	}
}

func (mlh mockListenHandler) Disconnect() {
	if mlh.OnDisconnect != nil {
		mlh.OnDisconnect()
	}
}

func (mlh mockListenHandler) Connected() {
	if mlh.OnConnect != nil {
		mlh.OnConnect()
	}
}

func TestPostgresListener_Listen(t *testing.T) {
	db := getDB(t)

	newOpts := func() PostgresListenerOpts {
		opts := DefaultPostgresListenerOpts
		opts.Addr = getDBConfig(t).ToPQString(true)
		opts.Channel = fmt.Sprintf("channel_%d", time.Now().UnixNano())
		opts.MinReconnectInterval = time.Nanosecond
		opts.MaxReconnectInterval = time.Second
		return opts
	}

	notifyListener := func(t *testing.T, channelName, payload string) {
		t.Helper()

		_, err := db.Exec(fmt.Sprintf(`NOTIFY %s, '%s'`, channelName, payload))
		assert.NoError(t, err)
	}

	listenNotify := func(t *testing.T, opts PostgresListenerOpts, numNotifiers int, payloads []string) (*PostgresListener, []string) {
		t.Helper()

		pgl, err := NewPostgresListener(opts)

		require.NoError(t, err)

		var wg sync.WaitGroup
		ctx, cancel := testhelper.Context()
		defer func() {
			cancel()
			wg.Wait()
		}()

		numResults := len(payloads) * numNotifiers
		allReceivedChan := make(chan struct{})

		wg.Add(1)
		go func() {
			defer func() {
				cancel()
				wg.Done()
			}()

			time.Sleep(100 * time.Millisecond)

			var notifyWG sync.WaitGroup
			notifyWG.Add(numNotifiers)
			for i := 0; i < numNotifiers; i++ {
				go func() {
					defer notifyWG.Done()

					for _, payload := range payloads {
						notifyListener(t, opts.Channel, payload)
					}
				}()
			}
			notifyWG.Wait()

			select {
			case <-time.After(time.Second):
				assert.FailNow(t, "notification propagation takes too long")
			case <-allReceivedChan:
			}
		}()

		result := make([]string, numResults)
		idx := int32(-1)
		callback := func(payload string) {
			i := int(atomic.AddInt32(&idx, 1))
			result[i] = payload
			if i+1 == numResults {
				close(allReceivedChan)
			}
		}

		require.NoError(t, pgl.Listen(ctx, mockListenHandler{OnNotification: callback}))

		return pgl, result
	}

	disconnectListener := func(t *testing.T, channelName string) {
		t.Helper()

		q := `SELECT PG_TERMINATE_BACKEND(pid) FROM PG_STAT_ACTIVITY WHERE datname = $1 AND query = $2`
		res, err := db.Exec(q, databaseName, fmt.Sprintf("LISTEN %q", channelName))
		if assert.NoError(t, err) {
			affected, err := res.RowsAffected()
			assert.NoError(t, err)
			assert.EqualValues(t, 1, affected)
		}
	}

	readMetrics := func(t *testing.T, col promclient.Collector) []promclientgo.Metric {
		t.Helper()

		metricsChan := make(chan promclient.Metric, 16)
		col.Collect(metricsChan)
		close(metricsChan)
		var metric []promclientgo.Metric
		for m := range metricsChan {
			var mtc promclientgo.Metric
			assert.NoError(t, m.Write(&mtc))
			metric = append(metric, mtc)
		}
		return metric
	}

	t.Run("single processor and single notifier", func(t *testing.T) {
		opts := newOpts()

		payloads := []string{"this", "is", "a", "payload"}

		listener, result := listenNotify(t, opts, 1, payloads)
		require.Equal(t, payloads, result)

		metrics := readMetrics(t, listener.reconnectTotal)
		require.Len(t, metrics, 1)
		require.Len(t, metrics[0].Label, 1)
		require.Equal(t, "state", *metrics[0].Label[0].Name)
		require.Equal(t, "connected", *metrics[0].Label[0].Value)
		require.GreaterOrEqual(t, *metrics[0].Counter.Value, 1.0)
	})

	t.Run("single processor and multiple notifiers", func(t *testing.T) {
		opts := newOpts()

		numNotifiers := 10

		payloads := []string{"this", "is", "a", "payload"}
		var expResult []string
		for i := 0; i < numNotifiers; i++ {
			expResult = append(expResult, payloads...)
		}

		_, result := listenNotify(t, opts, numNotifiers, payloads)
		assert.ElementsMatch(t, expResult, result, "there must be no additional data, only expected")
	})

	t.Run("re-listen", func(t *testing.T) {
		opts := newOpts()
		listener, result := listenNotify(t, opts, 1, []string{"1"})
		require.Equal(t, []string{"1"}, result)

		ctx, cancel := testhelper.Context()

		var connected int32

		errCh := make(chan error, 1)
		go func() {
			errCh <- listener.Listen(ctx, mockListenHandler{OnNotification: func(payload string) {
				atomic.StoreInt32(&connected, 1)
				assert.Equal(t, "2", payload)
			}})
		}()

		for atomic.LoadInt32(&connected) == 0 {
			notifyListener(t, opts.Channel, "2")
		}

		cancel()
		err := <-errCh
		require.NoError(t, err)
	})

	t.Run("already listening", func(t *testing.T) {
		opts := newOpts()

		listener, err := NewPostgresListener(opts)
		require.NoError(t, err)

		ctx, cancel := testhelper.Context()

		var connected int32

		errCh := make(chan error, 1)
		go func() {
			errCh <- listener.Listen(ctx, mockListenHandler{OnNotification: func(payload string) {
				atomic.StoreInt32(&connected, 1)
				assert.Equal(t, "2", payload)
			}})
		}()

		for atomic.LoadInt32(&connected) == 0 {
			notifyListener(t, opts.Channel, "2")
		}

		err = listener.Listen(ctx, mockListenHandler{})
		require.Error(t, err)
		require.Equal(t, fmt.Sprintf(`already listening channel %q of %q`, opts.Channel, opts.Addr), err.Error())

		cancel()
		require.NoError(t, <-errCh)
	})

	t.Run("invalid connection", func(t *testing.T) {
		opts := newOpts()
		opts.Addr = "invalid-address"

		listener, err := NewPostgresListener(opts)
		require.NoError(t, err)

		ctx, cancel := testhelper.Context()
		defer cancel()

		err = listener.Listen(ctx, mockListenHandler{OnNotification: func(string) {
			assert.FailNow(t, "no notifications expected to be received")
		}})
		require.Error(t, err, "it should not be possible to start listening on invalid connection")
	})

	t.Run("connection interruption", func(t *testing.T) {
		opts := newOpts()
		listener, err := NewPostgresListener(opts)
		require.NoError(t, err)

		ctx, cancel := testhelper.Context(testhelper.ContextWithTimeout(time.Second))
		defer cancel()

		var connected int32

		errChan := make(chan error, 1)
		go func() {
			errChan <- listener.Listen(ctx, mockListenHandler{OnNotification: func(string) {
				atomic.StoreInt32(&connected, 1)
			}})
		}()

		for atomic.LoadInt32(&connected) == 0 {
			notifyListener(t, opts.Channel, "")
		}

		disconnectListener(t, opts.Channel)
		atomic.StoreInt32(&connected, 0)

		for atomic.LoadInt32(&connected) == 0 {
			notifyListener(t, opts.Channel, "")
		}

		cancel()
		require.NoError(t, <-errChan)
	})

	t.Run("persisted connection interruption", func(t *testing.T) {
		opts := newOpts()
		opts.DisconnectThreshold = 2
		opts.DisconnectTimeWindow = time.Hour

		listener, err := NewPostgresListener(opts)
		require.NoError(t, err)

		ctx, cancel := testhelper.Context(testhelper.ContextWithTimeout(time.Second))
		defer cancel()

		var connected int32
		var disconnected int32

		errChan := make(chan error, 1)
		go func() {
			errChan <- listener.Listen(
				ctx,
				mockListenHandler{
					OnNotification: func(string) { atomic.StoreInt32(&connected, 1) },
					OnDisconnect:   func() { atomic.AddInt32(&disconnected, 1) },
				})
		}()

		for i := 0; i < opts.DisconnectThreshold; i++ {
			for atomic.LoadInt32(&connected) == 0 {
				time.Sleep(100 * time.Millisecond)
				notifyListener(t, opts.Channel, "")
			}

			disconnectListener(t, opts.Channel)
			atomic.StoreInt32(&connected, 0)
		}

		err = <-errChan
		require.Error(t, err)
		require.False(t, errors.Is(err, context.DeadlineExceeded), "listener was blocked for too long")

		metrics := readMetrics(t, listener.reconnectTotal)
		for _, metric := range metrics {
			switch *metric.Label[0].Value {
			case "connected":
				require.GreaterOrEqual(t, *metric.Counter.Value, 1.0)
			case "disconnected":
				require.EqualValues(t, disconnected, *metric.Counter.Value)
			case "reconnected":
				require.GreaterOrEqual(t, *metric.Counter.Value, 1.0)
			}
		}
	})
}

func TestThreshold(t *testing.T) {
	t.Run("reaches as there are no pauses between the calls", func(t *testing.T) {
		thresholdReached := threshold(100, time.Hour)

		for i := 0; i < 99; i++ {
			require.False(t, thresholdReached())
		}
		require.True(t, thresholdReached())
	})

	t.Run("doesn't reach because of pauses between the calls", func(t *testing.T) {
		thresholdReached := threshold(2, time.Microsecond)

		require.False(t, thresholdReached())
		time.Sleep(time.Millisecond)
		require.False(t, thresholdReached())
	})

	t.Run("reaches only on 6-th call because of the pause after first check", func(t *testing.T) {
		thresholdReached := threshold(5, time.Millisecond)

		require.False(t, thresholdReached())
		time.Sleep(time.Millisecond)
		require.False(t, thresholdReached())
		require.False(t, thresholdReached())
		require.False(t, thresholdReached())
		require.False(t, thresholdReached())
		require.True(t, thresholdReached())
	})

	t.Run("always reached for zero values", func(t *testing.T) {
		thresholdReached := threshold(0, 0)

		require.True(t, thresholdReached())
		time.Sleep(time.Millisecond)
		require.True(t, thresholdReached())
	})
}

func TestPostgresListener_Listen_repositories_delete(t *testing.T) {
	db := getDB(t)

	testListener(
		t,
		"repositories_updates",
		func(t *testing.T) {
			_, err := db.DB.Exec(`
				INSERT INTO repositories
				VALUES ('praefect-1', '/path/to/repo/1', 1),
					('praefect-1', '/path/to/repo/2', 1),
					('praefect-1', '/path/to/repo/3', 0)`)
			require.NoError(t, err)
		},
		func(t *testing.T) {
			_, err := db.DB.Exec(`DELETE FROM repositories WHERE generation > 0`)
			require.NoError(t, err)
		},
		func(t *testing.T, payload string) {
			require.JSONEq(t, `
				{
					"old": [
						{"virtual_storage":"praefect-1","relative_path":"/path/to/repo/1","generation":1,"primary":null},
						{"virtual_storage":"praefect-1","relative_path":"/path/to/repo/2","generation":1,"primary":null}
					],
					"new" : null
				}`,
				payload,
			)
		},
	)
}

func TestPostgresListener_Listen_storage_repositories_insert(t *testing.T) {
	db := getDB(t)

	testListener(
		t,
		"storage_repositories_updates",
		func(t *testing.T) {},
		func(t *testing.T) {
			_, err := db.DB.Exec(`
				INSERT INTO storage_repositories
				VALUES ('praefect-1', '/path/to/repo', 'gitaly-1', 0),
					('praefect-1', '/path/to/repo', 'gitaly-2', 0)`,
			)
			require.NoError(t, err)
		},
		func(t *testing.T, payload string) {
			require.JSONEq(t, `
				{
					"old":null,
					"new":[
						{"virtual_storage":"praefect-1","relative_path":"/path/to/repo","storage":"gitaly-1","generation":0,"assigned":true},
						{"virtual_storage":"praefect-1","relative_path":"/path/to/repo","storage":"gitaly-2","generation":0,"assigned":true}
					]
				}`,
				payload,
			)
		},
	)
}

func TestPostgresListener_Listen_storage_repositories_update(t *testing.T) {
	db := getDB(t)

	testListener(
		t,
		"storage_repositories_updates",
		func(t *testing.T) {
			_, err := db.DB.Exec(`INSERT INTO storage_repositories VALUES ('praefect-1', '/path/to/repo', 'gitaly-1', 0)`)
			require.NoError(t, err)
		},
		func(t *testing.T) {
			_, err := db.DB.Exec(`UPDATE storage_repositories SET generation = generation + 1`)
			require.NoError(t, err)
		},
		func(t *testing.T, payload string) {
			require.JSONEq(t, `
				{
					"old" : [{"virtual_storage":"praefect-1","relative_path":"/path/to/repo","storage":"gitaly-1","generation":0,"assigned":true}],
					"new" : [{"virtual_storage":"praefect-1","relative_path":"/path/to/repo","storage":"gitaly-1","generation":1,"assigned":true}]
				}`,
				payload,
			)
		},
	)
}

func TestPostgresListener_Listen_storage_repositories_delete(t *testing.T) {
	db := getDB(t)

	testListener(
		t,
		"storage_repositories_updates",
		func(t *testing.T) {
			_, err := db.DB.Exec(`
				INSERT INTO storage_repositories (virtual_storage, relative_path, storage, generation)
				VALUES ('praefect-1', '/path/to/repo', 'gitaly-1', 0)`,
			)
			require.NoError(t, err)
		},
		func(t *testing.T) {
			_, err := db.DB.Exec(`DELETE FROM storage_repositories`)
			require.NoError(t, err)
		},
		func(t *testing.T, payload string) {
			require.JSONEq(t, `
				{
					"old" : [{"virtual_storage":"praefect-1","relative_path":"/path/to/repo","storage":"gitaly-1","generation":0,"assigned":true}],
					"new" : null
				}`,
				payload,
			)
		},
	)
}

func testListener(t *testing.T, channel string, setup func(t *testing.T), trigger func(t *testing.T), verifier func(t *testing.T, payload string)) {
	setup(t)

	opts := DefaultPostgresListenerOpts
	opts.Addr = getDBConfig(t).ToPQString(true)
	opts.Channel = channel

	pgl, err := NewPostgresListener(opts)
	require.NoError(t, err)

	ctx, cancel := testhelper.Context()
	defer cancel()

	readyChan := make(chan struct{})
	receivedChan := make(chan struct{})
	var payload string

	callback := func(pld string) {
		select {
		case <-receivedChan:
			return
		default:
			payload = pld
			close(receivedChan)
		}
	}

	go func() {
		require.NoError(t, pgl.Listen(ctx, mockListenHandler{OnNotification: callback, OnConnect: func() { close(readyChan) }}))
	}()

	select {
	case <-time.After(time.Second):
		require.FailNow(t, "no connection for too long period")
	case <-readyChan:
	}

	trigger(t)

	select {
	case <-time.After(time.Second):
		require.FailNow(t, "no notifications for too long period")
	case <-receivedChan:
	}

	verifier(t, payload)
}
