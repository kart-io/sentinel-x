// Package config provides configuration management and hot reload capabilities.
package config

// Reloadable defines the interface for components that can handle configuration changes.
// Components implementing this interface can be notified when configuration is updated
// and react accordingly without requiring a service restart.
type Reloadable interface {
	// OnConfigChange is called when the configuration changes.
	// The newConfig parameter contains the updated configuration.
	// Implementations should validate the new configuration and apply changes atomically.
	// Returns an error if the configuration update cannot be applied.
	OnConfigChange(newConfig interface{}) error
}
