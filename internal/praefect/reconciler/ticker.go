package reconciler

import "time"

// Ticker ticks on the channel returned by C to signal something.
type Ticker interface {
	C() <-chan time.Time
	Stop()
	Reset()
}

// NewTimer returns a Ticker that ticks after the specified interval
// has passed since the previous Reset call.
func NewTimer(interval time.Duration) Ticker {
	// use a long time to initialize to prevent the ticker from
	// firing before first reset call.
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	return &timerTicker{timer: timer, interval: interval}
}

type timerTicker struct {
	timer    *time.Timer
	interval time.Duration
}

func (tt *timerTicker) C() <-chan time.Time { return tt.timer.C }

func (tt *timerTicker) Reset() { tt.timer.Reset(tt.interval) }

func (tt *timerTicker) Stop() { tt.timer.Stop() }
