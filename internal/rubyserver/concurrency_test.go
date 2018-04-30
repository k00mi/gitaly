package rubyserver

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/config"
	"gitlab.com/gitlab-org/gitaly/internal/testhelper"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

func TestConcurrency(t *testing.T) {
	testhelper.ConfigureRuby()
	config.Config.Ruby.Concurrency = 1

	s, err := Start()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Stop()

	require.NoError(t, waitPing(s), "wait for gitaly-ruby to boot")

	start := make(chan struct{})
	wg := &sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			<-start
			for j := 0; j < 10; j++ {
				// Before we had gitaly-ruby rate limiting, this could result in
				// ResourceExhausted errors. This test is meant to guard against that
				// problem coming back.
				require.NoError(t, makeRequest(s))
			}
		}()
	}

	close(start) // increase chances that goroutines all run at once

	wg.Wait()
}

func waitPing(s *Server) error {
	var err error
	for start := time.Now(); time.Since(start) < ConnectTimeout; time.Sleep(100 * time.Millisecond) {
		err = makeRequest(s)
		if err == nil {
			return nil
		}
	}
	return err
}

// This benchmark lets you see what happens when you throw a lot of
// concurrent traffic at gitaly-ruby.
func BenchmarkConcurrency(b *testing.B) {
	testhelper.ConfigureRuby()
	config.Config.Ruby.NumWorkers = 2

	s, err := Start()
	if err != nil {
		b.Fatal(err)
	}
	defer s.Stop()

	// Warm-up: wait for gitaly-ruby to boot
	if err := waitPing(s); err != nil {
		b.Fatal(err)
	}

	concurrency := 100
	b.Run(fmt.Sprintf("concurrency %d", concurrency), func(b *testing.B) {
		errCh := make(chan error)
		errCount := make(chan int)
		go func() {
			count := 0
			for err := range errCh {
				b.Log(err)
				count++
			}
			errCount <- count
		}()

		wg := &sync.WaitGroup{}
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for j := 0; j < 1000; j++ {
					err := makeRequest(s)
					if err != nil {
						errCh <- err
					}

					switch status.Code(err) {
					case codes.Unavailable:
						return
					case codes.DeadlineExceeded:
						return
					}
				}
			}()
		}

		wg.Wait()
		close(errCh)

		if count := <-errCount; count != 0 {
			b.Fatalf("received %d errors", count)
		}
	})
}

func makeRequest(s *Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := s.getConnection(ctx)
	if err != nil {
		return err
	}

	client := healthpb.NewHealthClient(conn)
	_, err = client.Check(ctx, &healthpb.HealthCheckRequest{})
	return err
}
