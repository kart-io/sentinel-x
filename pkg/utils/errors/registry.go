package errors

import (
	"fmt"
	"sync"
)

// errnoRegistry stores all registered error codes for uniqueness validation.
var (
	errnoRegistry = make(map[int]*Errno)
	registryMu    sync.RWMutex
)

// Register registers an Errno and validates uniqueness.
// Panics if the code is already registered.
func Register(e *Errno) *Errno {
	registryMu.Lock()
	defer registryMu.Unlock()

	if existing, ok := errnoRegistry[e.Code]; ok {
		panic(fmt.Sprintf("errno code %d already registered: %s", e.Code, existing.MessageEN))
	}
	errnoRegistry[e.Code] = e
	return e
}

// MustRegister is an alias for Register for consistency.
func MustRegister(e *Errno) *Errno {
	return Register(e)
}

// Lookup returns the registered Errno for the given code.
func Lookup(code int) (*Errno, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	e, ok := errnoRegistry[code]
	return e, ok
}

// GetAllRegistered returns all registered error codes.
// This is useful for documentation and debugging.
func GetAllRegistered() map[int]*Errno {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make(map[int]*Errno, len(errnoRegistry))
	for k, v := range errnoRegistry {
		result[k] = v
	}
	return result
}

// RegistrySize returns the number of registered error codes.
func RegistrySize() int {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return len(errnoRegistry)
}
