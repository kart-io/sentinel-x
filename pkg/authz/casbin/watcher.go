package casbin

// Watcher defines the interface for distributed policy synchronization
type Watcher interface {
	// SetUpdateCallback sets the callback function to be called when an update is received
	SetUpdateCallback(callback func(string))

	// Update notifies other instances that the policy has changed
	Update() error

	// Close stops the watcher
	Close()
}
