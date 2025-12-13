package cache

import "errors"

// ErrIndexNotFound is returned when querying a non-existent index
var ErrIndexNotFound = errors.New("index not found")

// Cache defines the basic interface for a generic cache
type Cache[K comparable, V any] interface {
	// Set adds or updates an item in the cache
	Set(key K, value V)
	// Get retrieves an item from the cache
	Get(key K) (V, bool)
	// Del removes an item from the cache
	Del(key K)
	// Len returns the number of items in the cache
	Len() int
	// Keys returns all keys in the cache
	Keys() []K
	// Values returns all values in the cache
	Values() []V
	// Clear removes all items from the cache
	Clear()
	// Contains checks if a key exists
	Contains(key K) bool
	// Load imports a slice of items into the cache using a key extractor
	Load(items []V, keyFunc func(V) K)
}

// Store extends Cache with querying capabilities
type Store[K comparable, V any] interface {
	Cache[K, V]

	// AddIndex registers a new secondary index
	AddIndex(name string, extractor func(V) any)

	// Find retrieves items matching the index criteria
	Find(indexName string, indexValue any) ([]V, error)

	// Filter scans the cache and returns items matching the predicate
	Filter(predicate func(V) bool) []V
}
