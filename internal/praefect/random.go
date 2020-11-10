package praefect

import (
	"sync"
)

// Random is the interface of the Go random number generator.
type Random interface {
	// Intn returns a random integer in the range [0,n).
	Intn(n int) int
}

// randomFunc is an adapter to turn conforming functions in to a Random.
type randomFunc func(n int) int

func (fn randomFunc) Intn(n int) int { return fn(n) }

// NewLockedRandom wraps the passed in Random to make it safe for concurrent use.
func NewLockedRandom(r Random) Random {
	var m sync.Mutex
	return randomFunc(func(n int) int {
		m.Lock()
		defer m.Unlock()
		return r.Intn(n)
	})
}
