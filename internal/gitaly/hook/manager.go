package hook

// Manager is a hook manager containing Git hook business logic.
type Manager struct{}

// NewManager returns a new hook manager
func NewManager() *Manager {
	return &Manager{}
}
