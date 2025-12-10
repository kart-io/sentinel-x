package mongodb

// This file contains health check utilities for MongoDB clients.
// The primary health check interface is implemented in the Client type
// via the Health() method, which returns a storage.HealthChecker function.
//
// Additional health check utilities can be added here as needed.
