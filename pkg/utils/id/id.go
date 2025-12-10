// Package id provides unique ID generation utilities for Sentinel-X.
//
// This package supports multiple ID generation strategies:
//   - UUID: Standard UUID v4 (random)
//   - Snowflake: Twitter's distributed ID algorithm (time-based, sortable)
//   - ULID: Universally Unique Lexicographically Sortable Identifier
//
// Usage:
//
//	// Using default generators
//	uuid := id.NewUUID()           // e.g., "550e8400-e29b-41d4-a716-446655440000"
//	sfid := id.NewSnowflake()      // e.g., "1234567890123456789"
//	ulid := id.NewULID()           // e.g., "01ARZ3NDEKTSV4RRFFQ69G5FAV"
//
//	// Using generator instances for custom configuration
//	gen := id.NewSnowflakeGenerator(id.WithNodeID(1))
//	id := gen.Generate()
package id

import (
	"sync"
)

// Generator defines the interface for ID generators.
type Generator interface {
	// Generate creates a new unique ID.
	Generate() string

	// GenerateN creates n unique IDs.
	GenerateN(n int) []string
}

// Type represents the type of ID generator.
type Type string

const (
	// TypeUUID represents UUID v4 generator.
	TypeUUID Type = "uuid"

	// TypeSnowflake represents Snowflake ID generator.
	TypeSnowflake Type = "snowflake"

	// TypeULID represents ULID generator.
	TypeULID Type = "ulid"
)

var (
	defaultUUID      Generator
	defaultSnowflake Generator
	defaultULID      Generator
	initOnce         sync.Once
)

// initDefaults initializes default generators.
func initDefaults() {
	initOnce.Do(func() {
		defaultUUID = NewUUIDGenerator()
		defaultSnowflake, _ = NewSnowflakeGenerator()
		defaultULID = NewULIDGenerator()
	})
}

// NewUUID generates a new UUID v4 string.
func NewUUID() string {
	initDefaults()
	return defaultUUID.Generate()
}

// NewSnowflake generates a new Snowflake ID string.
func NewSnowflake() string {
	initDefaults()
	return defaultSnowflake.Generate()
}

// NewULID generates a new ULID string.
func NewULID() string {
	initDefaults()
	return defaultULID.Generate()
}

// New generates a new ID using the specified generator type.
func New(t Type) string {
	switch t {
	case TypeUUID:
		return NewUUID()
	case TypeSnowflake:
		return NewSnowflake()
	case TypeULID:
		return NewULID()
	default:
		return NewUUID()
	}
}

// Must panics if err is not nil, otherwise returns the value.
func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
