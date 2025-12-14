package store

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// User is a test struct for GORM
type User struct {
	ID   uint `gorm:"primaryKey"`
	Name string
	Age  int
	Role string
}

func setupTestDB(t *testing.T) *gorm.DB {
	// Use unique DB name per test to ensure isolation
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&User{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestWhere_Pagination(t *testing.T) {
	db := setupTestDB(t)

	// Seed data
	users := []User{
		{Name: "User1", Age: 20},
		{Name: "User2", Age: 20},
		{Name: "User3", Age: 20},
		{Name: "User4", Age: 20},
		{Name: "User5", Age: 20},
	}
	db.Create(&users)

	tests := []struct {
		name     string
		opts     []Option
		wantLen  int
		wantName string // Check first item name
	}{
		{
			name:     "Limit 2",
			opts:     []Option{WithLimit(2)},
			wantLen:  2,
			wantName: "User1",
		},
		{
			name:     "Offset 2, Limit 2",
			opts:     []Option{WithOffset(2), WithLimit(2)},
			wantLen:  2,
			wantName: "User3",
		},
		{
			name:     "Page 2, Size 2",
			opts:     []Option{WithPage(2, 2)},
			wantLen:  2,
			wantName: "User3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var results []User
			whr := NewWhere(tt.opts...)

			// Apply Where
			tx := whr.Where(db)

			if err := tx.Find(&results).Error; err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if len(results) != tt.wantLen {
				t.Errorf("got len %d, want %d", len(results), tt.wantLen)
			}
			if len(results) > 0 && results[0].Name != tt.wantName {
				t.Errorf("got first item %s, want %s", results[0].Name, tt.wantName)
			}
		})
	}
}

func TestWhere_Filtering(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&User{Name: "Alice", Age: 30, Role: "Admin"})
	db.Create(&User{Name: "Bob", Age: 25, Role: "User"})
	db.Create(&User{Name: "Charlie", Age: 30, Role: "User"})

	tests := []struct {
		name    string
		opts    []Option
		wantLen int
	}{
		{
			name:    "Filter by Age 30",
			opts:    []Option{WithFilter(map[interface{}]interface{}{"age": 30})},
			wantLen: 2, // Alice and Charlie
		},
		{
			name:    "Filter by Role Admin",
			opts:    []Option{WithFilter(map[interface{}]interface{}{"role": "Admin"})},
			wantLen: 1, // Alice
		},
		{
			name: "Helper F(k, v)",
			// F returns *Options, so we cannot put it in opts list which expects Option func.
			// We will test WithFilter again but with different syntax, or we should test F separately.
			// Let's use WithFilter again to satisfy the []Option type.
			opts:    []Option{WithFilter(map[interface{}]interface{}{"role": "User"})},
			wantLen: 2, // Bob and Charlie
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var results []User
			whr := NewWhere(tt.opts...)
			tx := whr.Where(db)
			if err := tx.Find(&results).Error; err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if len(results) != tt.wantLen {
				t.Errorf("got len %d, want %d", len(results), tt.wantLen)
			}
		})
	}

	// Separate test for F convenience function
	t.Run("Convenience F", func(t *testing.T) {
		var results []User
		// F creates new options directly
		whr := F("role", "User")
		whr.Where(db).Find(&results)
		if len(results) != 2 {
			t.Errorf("got len %d, want 2", len(results))
		}
	})
}

func TestWhere_Query(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&User{Name: "Alice", Age: 30})
	db.Create(&User{Name: "Bob", Age: 25})

	// Test Q("age > ?", 28)
	var results []User
	whr := NewWhere().Q("age > ?", 28)
	whr.Where(db).Find(&results)

	if len(results) != 1 || results[0].Name != "Alice" {
		t.Errorf("Query failed: got %v", results)
	}
}

func TestWhere_Clauses(t *testing.T) {
	db := setupTestDB(t)
	db.Create(&User{Name: "Zack", Age: 20})
	db.Create(&User{Name: "Adam", Age: 20})

	// Test Order Clause
	var results []User
	whr := NewWhere(WithClauses(clause.OrderBy{
		Columns: []clause.OrderByColumn{{Column: clause.Column{Name: "name"}}},
	}))

	whr.Where(db).Find(&results)

	if len(results) != 2 {
		t.Fatal("Expected 2 results")
	}
	if results[0].Name != "Adam" {
		t.Errorf("Expected Adam first, got %s", results[0].Name)
	}
}

func TestWhere_Tenant(t *testing.T) {
	// Register global tenant for test
	RegisterTenant("tenant_id", func(_ context.Context) string {
		return "tenant-1"
	})

	// Since User struct doesn't have tenant_id, we just check if filter is added to options
	// interacting with DB would fail unless we add the column, but we verify option logic here.

	ctx := context.Background()
	whr := NewWhere()
	whr.T(ctx)

	if val, ok := whr.Filters["tenant_id"]; !ok || val != "tenant-1" {
		t.Errorf("Tenant filter not set correctly: got %v", whr.Filters)
	}
}
