package rubyserver

import (
	"time"
)

type stopwatch struct {
	t1      time.Time
	t2      time.Time
	running bool
}

// mark records the current time and starts the stopwatch if it is not already running
func (st *stopwatch) mark() {
	st.t2 = time.Now()

	if !st.running {
		st.t1 = st.t2
		st.running = true
	}
}

// reset stops the stopwatch and returns it to zero
func (st *stopwatch) reset() {
	st.running = false
}

// elapsed returns the time elapsed between the first and last 'mark'
func (st *stopwatch) elapsed() time.Duration {
	if !st.running {
		return time.Duration(0)
	}

	return st.t2.Sub(st.t1)
}
