// +build postgres

package reconciler

import "time"

type manualTicker struct {
	c     chan time.Time
	stop  func()
	reset func()
}

func (mt *manualTicker) C() <-chan time.Time { return mt.c }

func (mt *manualTicker) Stop() { mt.stop() }

func (mt *manualTicker) Reset() { mt.reset() }

func (mt *manualTicker) Tick() { mt.c <- time.Now() }

func newManualTicker() *manualTicker {
	return &manualTicker{
		c:     make(chan time.Time, 1),
		stop:  func() {},
		reset: func() {},
	}
}
