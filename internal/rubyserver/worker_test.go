package rubyserver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitaly/internal/config"
	"gitlab.com/gitlab-org/gitaly/internal/supervisor"
)

func TestWorker(t *testing.T) {
	restartDelay := 100 * time.Millisecond
	defer func(old time.Duration) {
		config.Config.Ruby.RestartDelay = old
	}(config.Config.Ruby.RestartDelay)
	config.Config.Ruby.RestartDelay = restartDelay

	events := make(chan supervisor.Event)
	addr := "the address"
	w := newWorker(&supervisor.Process{Name: "testing"}, addr, events)

	mustIgnore := func(e supervisor.Event) {
		nothing := &nothingBalancer{t}
		w.balancerUpdate <- nothing
		t.Logf("sending Event %+v, expect nothing to happen", e)
		events <- e
		// This second balancer update is used to synchronize with the monitor
		// goroutine. When the channel send finishes, we know the event we sent
		// before must have been processed.
		w.balancerUpdate <- nothing
	}

	mustAdd := func(e supervisor.Event) {
		add := newAdd(t, addr)
		w.balancerUpdate <- add
		t.Logf("sending Event %+v, expect balancer add", e)
		events <- e
		add.wait()
	}

	mustRemove := func(e supervisor.Event) {
		remove := newRemove(t, addr)
		w.balancerUpdate <- remove
		t.Logf("sending Event %+v, expect balancer remove", e)
		events <- e
		remove.wait()
	}

	firstPid := 123

	mustAdd(upEvent(firstPid))

	t.Log("ignore repeated up event")
	mustIgnore(upEvent(firstPid))

	t.Log("send mem high events but too fast to trigger restart")
	for i := 0; i < 5; i++ {
		mustIgnore(memHighEvent(firstPid))
	}

	t.Log("mem low resets mem high counter")
	mustIgnore(memLowEvent(firstPid))

	t.Log("send mem high events but too fast to trigger restart")
	for i := 0; i < 5; i++ {
		mustIgnore(memHighEvent(firstPid))
	}

	time.Sleep(2 * restartDelay)
	t.Log("this mem high should push us over the threshold")
	mustRemove(memHighEvent(firstPid))

	secondPid := 456
	t.Log("time for a new PID")
	mustAdd(upEvent(secondPid))

	t.Log("ignore mem high events for the previous pid")
	mustIgnore(memHighEvent(firstPid))
	time.Sleep(2 * restartDelay)
	mustIgnore(memHighEvent(firstPid))

	t.Log("start high memory timer")
	mustIgnore(memHighEvent(secondPid))

	t.Log("ignore mem low event for wrong pid")
	mustIgnore(memLowEvent(firstPid))

	t.Log("send mem high count over the threshold")
	time.Sleep(2 * restartDelay)
	mustRemove(memHighEvent(secondPid))
}

func waitFail(t *testing.T, done chan struct{}) {
	select {
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for balancer method call")
	case <-done:
	}
}

func upEvent(pid int) supervisor.Event {
	return supervisor.Event{Type: supervisor.Up, Pid: pid}
}

func memHighEvent(pid int) supervisor.Event {
	return supervisor.Event{Type: supervisor.MemoryHigh, Pid: pid}
}

func memLowEvent(pid int) supervisor.Event {
	return supervisor.Event{Type: supervisor.MemoryLow, Pid: pid}
}

func newAdd(t *testing.T, addr string) *addBalancer {
	return &addBalancer{
		t:    t,
		addr: addr,
		done: make(chan struct{}),
	}
}

type addBalancer struct {
	addr string
	t    *testing.T
	done chan struct{}
}

func (ab *addBalancer) RemoveAddress(string) bool {
	ab.t.Fatal("unexpected RemoveAddress call")
	return false
}

func (ab *addBalancer) AddAddress(s string) {
	require.Equal(ab.t, ab.addr, s, "addBalancer expected AddAddress argument")
	close(ab.done)
}

func (ab *addBalancer) wait() {
	waitFail(ab.t, ab.done)
}

func newRemove(t *testing.T, addr string) *removeBalancer {
	return &removeBalancer{
		t:    t,
		addr: addr,
		done: make(chan struct{}),
	}
}

type removeBalancer struct {
	addr string
	t    *testing.T
	done chan struct{}
}

func (rb *removeBalancer) RemoveAddress(s string) bool {
	require.Equal(rb.t, rb.addr, s, "removeBalancer expected RemoveAddress argument")
	close(rb.done)
	return false
}

func (rb *removeBalancer) AddAddress(s string) {
	rb.t.Fatal("unexpected AddAddress call")
}

func (rb *removeBalancer) wait() {
	waitFail(rb.t, rb.done)
}

type nothingBalancer struct {
	t *testing.T
}

func (nb *nothingBalancer) RemoveAddress(s string) bool {
	nb.t.Fatal("unexpected RemoveAddress call")
	return false
}

func (nb *nothingBalancer) AddAddress(s string) {
	nb.t.Fatal("unexpected AddAddress call")
}
