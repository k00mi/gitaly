package promtest

import (
	"sync"
)

type MockCounter struct {
	m     sync.RWMutex
	value float64
}

func (m *MockCounter) Value() float64 {
	m.m.RLock()
	defer m.m.RUnlock()
	return m.value
}

func (m *MockCounter) Inc() {
	m.Add(1)
}

func (m *MockCounter) Add(v float64) {
	m.m.Lock()
	defer m.m.Unlock()
	m.value += v
}
