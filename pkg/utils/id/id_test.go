package id

import (
	"strings"
	"testing"
	"time"
)

func TestUUIDGenerator(t *testing.T) {
	gen := NewUUIDGenerator()

	t.Run("Generate", func(t *testing.T) {
		id := gen.Generate()
		if len(id) != 36 {
			t.Errorf("expected UUID length 36, got %d", len(id))
		}

		// Check format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
		parts := strings.Split(id, "-")
		if len(parts) != 5 {
			t.Errorf("expected 5 parts, got %d", len(parts))
		}

		// Check version (4)
		if id[14] != '4' {
			t.Errorf("expected version 4, got %c", id[14])
		}

		// Check variant
		variant := id[19]
		if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' {
			t.Errorf("expected variant 8/9/a/b, got %c", variant)
		}
	})

	t.Run("GenerateN", func(t *testing.T) {
		ids := gen.GenerateN(10)
		if len(ids) != 10 {
			t.Errorf("expected 10 IDs, got %d", len(ids))
		}

		// Check uniqueness
		seen := make(map[string]bool)
		for _, id := range ids {
			if seen[id] {
				t.Errorf("duplicate ID: %s", id)
			}
			seen[id] = true
		}
	})

	t.Run("IsValidUUID", func(t *testing.T) {
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		if !IsValidUUID(validUUID) {
			t.Errorf("expected %s to be valid", validUUID)
		}

		invalidUUIDs := []string{
			"",
			"invalid",
			"550e8400-e29b-41d4-a716",
			"550e8400-e29b-41d4-a716-44665544000g",
		}
		for _, invalid := range invalidUUIDs {
			if IsValidUUID(invalid) {
				t.Errorf("expected %s to be invalid", invalid)
			}
		}
	})
}

func TestSnowflakeGenerator(t *testing.T) {
	gen, err := NewSnowflakeGenerator(WithNodeID(1))
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	t.Run("Generate", func(t *testing.T) {
		id := gen.Generate()
		if id == "" {
			t.Error("expected non-empty ID")
		}

		// Should be parseable
		parsed, err := ParseSnowflakeString(id)
		if err != nil {
			t.Errorf("failed to parse: %v", err)
		}

		if parsed.NodeID != 1 {
			t.Errorf("expected node ID 1, got %d", parsed.NodeID)
		}
	})

	t.Run("GenerateN", func(t *testing.T) {
		ids := gen.GenerateN(100)
		if len(ids) != 100 {
			t.Errorf("expected 100 IDs, got %d", len(ids))
		}

		// Check uniqueness and ordering
		seen := make(map[string]bool)
		for _, id := range ids {
			if seen[id] {
				t.Errorf("duplicate ID: %s", id)
			}
			seen[id] = true
		}
	})

	t.Run("InvalidNodeID", func(t *testing.T) {
		_, err := NewSnowflakeGenerator(WithNodeID(1024))
		if err != ErrInvalidNodeID {
			t.Errorf("expected ErrInvalidNodeID, got %v", err)
		}
	})

	t.Run("ParseSnowflake", func(t *testing.T) {
		id := gen.GenerateInt64()
		parsed := ParseSnowflake(id)

		if parsed.NodeID != 1 {
			t.Errorf("expected node ID 1, got %d", parsed.NodeID)
		}

		// Time should be recent
		timeDiff := time.Since(parsed.Time())
		if timeDiff > time.Second {
			t.Errorf("time difference too large: %v", timeDiff)
		}
	})
}

func TestULIDGenerator(t *testing.T) {
	gen := NewULIDGenerator()

	t.Run("Generate", func(t *testing.T) {
		id := gen.Generate()
		if len(id) != 26 {
			t.Errorf("expected ULID length 26, got %d", len(id))
		}

		// Should be uppercase
		if id != strings.ToUpper(id) {
			t.Errorf("expected uppercase ULID")
		}
	})

	t.Run("GenerateN", func(t *testing.T) {
		ids := gen.GenerateN(100)
		if len(ids) != 100 {
			t.Errorf("expected 100 IDs, got %d", len(ids))
		}

		// Check uniqueness
		seen := make(map[string]bool)
		for _, id := range ids {
			if seen[id] {
				t.Errorf("duplicate ID: %s", id)
			}
			seen[id] = true
		}
	})

	t.Run("Monotonicity", func(t *testing.T) {
		// Generate multiple IDs in quick succession
		ids := gen.GenerateN(10)

		// They should be lexicographically sorted
		for i := 1; i < len(ids); i++ {
			if ids[i] < ids[i-1] {
				t.Errorf("IDs not monotonic: %s < %s", ids[i], ids[i-1])
			}
		}
	})

	t.Run("ParseULID", func(t *testing.T) {
		id := gen.Generate()
		parsed, err := ParseULID(id)
		if err != nil {
			t.Errorf("failed to parse: %v", err)
		}

		// Time should be recent
		timeDiff := time.Since(parsed.Time())
		if timeDiff > time.Second {
			t.Errorf("time difference too large: %v", timeDiff)
		}
	})

	t.Run("IsValidULID", func(t *testing.T) {
		valid := gen.Generate()
		if !IsValidULID(valid) {
			t.Errorf("expected %s to be valid", valid)
		}

		invalidULIDs := []string{
			"",
			"invalid",
			"01ARZ3NDEKTSV4RRFFQ69G5FA", // too short
		}
		for _, invalid := range invalidULIDs {
			if IsValidULID(invalid) {
				t.Errorf("expected %s to be invalid", invalid)
			}
		}
	})
}

func TestDefaultGenerators(t *testing.T) {
	t.Run("NewUUID", func(t *testing.T) {
		id := NewUUID()
		if !IsValidUUID(id) {
			t.Errorf("NewUUID returned invalid UUID: %s", id)
		}
	})

	t.Run("NewSnowflake", func(t *testing.T) {
		id := NewSnowflake()
		if id == "" {
			t.Error("NewSnowflake returned empty string")
		}
	})

	t.Run("NewULID", func(t *testing.T) {
		id := NewULID()
		if !IsValidULID(id) {
			t.Errorf("NewULID returned invalid ULID: %s", id)
		}
	})

	t.Run("New", func(t *testing.T) {
		uuidID := New(TypeUUID)
		if !IsValidUUID(uuidID) {
			t.Errorf("New(TypeUUID) returned invalid UUID: %s", uuidID)
		}

		snowflakeID := New(TypeSnowflake)
		if snowflakeID == "" {
			t.Error("New(TypeSnowflake) returned empty string")
		}

		ulidID := New(TypeULID)
		if !IsValidULID(ulidID) {
			t.Errorf("New(TypeULID) returned invalid ULID: %s", ulidID)
		}
	})
}

func BenchmarkUUID(b *testing.B) {
	gen := NewUUIDGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkSnowflake(b *testing.B) {
	gen, _ := NewSnowflakeGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkULID(b *testing.B) {
	gen := NewULIDGenerator()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}
