package cache

import (
	"sync"
)

// MemoryCache implements a thread-safe in-memory cache with indexing support.
type MemoryCache[K comparable, V any] struct {
	mu sync.RWMutex

	// data stores the primary key-value pairs
	data map[K]V

	// extractors stores function to extract index values from items
	extractors map[string]func(V) any

	// indices stores the index data: indexName -> indexValue -> set of keys
	indices map[string]map[any]map[K]struct{}
}

// NewMemoryCache creates a new instance of MemoryCache
func NewMemoryCache[K comparable, V any]() *MemoryCache[K, V] {
	return &MemoryCache[K, V]{
		data:       make(map[K]V),
		extractors: make(map[string]func(V) any),
		indices:    make(map[string]map[any]map[K]struct{}),
	}
}

// Set adds or updates an item in the cache
func (c *MemoryCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key exists, we need to remove old index entries first
	if oldValue, exists := c.data[key]; exists {
		c.removeFromIndexes(key, oldValue)
	}

	c.data[key] = value
	c.addToIndexes(key, value)
}

// Load imports a slice of items into the cache using a key extractor
func (c *MemoryCache[K, V]) Load(items []V, keyFunc func(V) K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, item := range items {
		key := keyFunc(item)
		// If key exists, we need to remove old index entries first
		if oldValue, exists := c.data[key]; exists {
			c.removeFromIndexes(key, oldValue)
		}
		c.data[key] = item
		c.addToIndexes(key, item)
	}
}

// Get retrieves an item from the cache
func (c *MemoryCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, ok := c.data[key]
	return val, ok
}

// Del removes an item from the cache
func (c *MemoryCache[K, V]) Del(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if oldValue, exists := c.data[key]; exists {
		c.removeFromIndexes(key, oldValue)
		delete(c.data, key)
	}
}

// Keys returns all keys in the cache
func (c *MemoryCache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]K, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values in the cache
func (c *MemoryCache[K, V]) Values() []V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	values := make([]V, 0, len(c.data))
	for _, v := range c.data {
		values = append(values, v)
	}
	return values
}

// Len returns the number of items in the cache
func (c *MemoryCache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Clear removes all items from the cache
func (c *MemoryCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[K]V)
	// Re-initialize indices structure but keep extractors
	c.indices = make(map[string]map[any]map[K]struct{})
	for name := range c.extractors {
		c.indices[name] = make(map[any]map[K]struct{})
	}
}

// Contains checks if a key exists
func (c *MemoryCache[K, V]) Contains(key K) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.data[key]
	return exists
}

// AddIndex registers a new secondary index
func (c *MemoryCache[K, V]) AddIndex(name string, extractor func(V) any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.extractors[name] = extractor
	c.indices[name] = make(map[any]map[K]struct{})

	// Re-index existing data
	for k, v := range c.data {
		val := extractor(v)
		c.addIndexEntry(name, val, k)
	}
}

// Find retrieves items matching the index criteria
func (c *MemoryCache[K, V]) Find(indexName string, indexValue any) ([]V, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if index exists
	if _, ok := c.extractors[indexName]; !ok {
		return nil, ErrIndexNotFound
	}

	index, ok := c.indices[indexName]
	if !ok {
		return []V{}, nil
	}

	keySet, ok := index[indexValue]
	if !ok {
		return []V{}, nil
	}

	results := make([]V, 0, len(keySet))
	for k := range keySet {
		if val, exists := c.data[k]; exists {
			results = append(results, val)
		}
	}

	return results, nil
}

// Filter scans the cache and returns items matching the predicate
func (c *MemoryCache[K, V]) Filter(predicate func(V) bool) []V {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var results []V
	for _, v := range c.data {
		if predicate(v) {
			results = append(results, v)
		}
	}
	return results
}

// Internal helper methods (assumes lock is held)

func (c *MemoryCache[K, V]) addToIndexes(key K, value V) {
	for name, extractor := range c.extractors {
		val := extractor(value)
		c.addIndexEntry(name, val, key)
	}
}

func (c *MemoryCache[K, V]) removeFromIndexes(key K, value V) {
	for name, extractor := range c.extractors {
		val := extractor(value)
		c.removeIndexEntry(name, val, key)
	}
}

func (c *MemoryCache[K, V]) addIndexEntry(indexName string, indexValue any, key K) {
	index, ok := c.indices[indexName]
	if !ok {
		index = make(map[any]map[K]struct{})
		c.indices[indexName] = index
	}

	keySet, ok := index[indexValue]
	if !ok {
		keySet = make(map[K]struct{})
		index[indexValue] = keySet
	}
	keySet[key] = struct{}{}
}

func (c *MemoryCache[K, V]) removeIndexEntry(indexName string, indexValue any, key K) {
	if index, ok := c.indices[indexName]; ok {
		if keySet, ok := index[indexValue]; ok {
			delete(keySet, key)
			// Cleanup empty maps to save memory
			if len(keySet) == 0 {
				delete(index, indexValue)
			}
		}
	}
}
