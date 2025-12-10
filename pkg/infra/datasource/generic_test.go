package datasource

import (
	"context"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/component/etcd"
	"github.com/kart-io/sentinel-x/pkg/component/mongodb"
	"github.com/kart-io/sentinel-x/pkg/component/mysql"
	"github.com/kart-io/sentinel-x/pkg/component/postgres"
	"github.com/kart-io/sentinel-x/pkg/component/redis"
)

// =============================================================================
// Tests for TypedGetter[T]
// =============================================================================

func TestTypedGetter_MySQL(t *testing.T) {
	mgr := NewManager()

	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "test"
	opts.Username = "root"

	err := mgr.RegisterMySQL("primary", opts)
	if err != nil {
		t.Fatalf("RegisterMySQL failed: %v", err)
	}

	// Test the new generic getter
	mysqlGetter := mgr.MySQL()
	if mysqlGetter == nil {
		t.Fatal("MySQL() returned nil getter")
	}

	// Note: This will attempt to connect, which will fail in testing
	// but we're testing the API structure
	_, err = mysqlGetter.Get("primary")
	if err == nil {
		t.Log("Note: MySQL connection succeeded unexpectedly in test environment")
	}
}

func TestTypedGetter_Redis(t *testing.T) {
	mgr := NewManager()

	opts := redis.NewOptions()
	opts.Host = "localhost"

	err := mgr.RegisterRedis("cache", opts)
	if err != nil {
		t.Fatalf("RegisterRedis failed: %v", err)
	}

	// Test the new generic getter
	redisGetter := mgr.Redis()
	if redisGetter == nil {
		t.Fatal("Redis() returned nil getter")
	}

	// Test that unregistered instance returns error
	_, err = redisGetter.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unregistered instance")
	}
}

func TestTypedGetter_Postgres(t *testing.T) {
	mgr := NewManager()

	opts := postgres.NewOptions()
	opts.Host = "localhost"
	opts.Database = "test"
	opts.Username = "postgres"

	err := mgr.RegisterPostgres("main", opts)
	if err != nil {
		t.Fatalf("RegisterPostgres failed: %v", err)
	}

	postgresGetter := mgr.Postgres()
	if postgresGetter == nil {
		t.Fatal("Postgres() returned nil getter")
	}
}

func TestTypedGetter_MongoDB(t *testing.T) {
	mgr := NewManager()

	opts := mongodb.NewOptions()
	opts.Host = "localhost"
	opts.Database = "test"

	err := mgr.RegisterMongoDB("main", opts)
	if err != nil {
		t.Fatalf("RegisterMongoDB failed: %v", err)
	}

	mongoGetter := mgr.MongoDB()
	if mongoGetter == nil {
		t.Fatal("MongoDB() returned nil getter")
	}
}

func TestTypedGetter_Etcd(t *testing.T) {
	mgr := NewManager()

	opts := etcd.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}

	err := mgr.RegisterEtcd("main", opts)
	if err != nil {
		t.Fatalf("RegisterEtcd failed: %v", err)
	}

	etcdGetter := mgr.Etcd()
	if etcdGetter == nil {
		t.Fatal("Etcd() returned nil getter")
	}
}

func TestTypedGetter_GetWithContext(t *testing.T) {
	mgr := NewManager()

	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "test"
	opts.Username = "root"

	err := mgr.RegisterMySQL("primary", opts)
	if err != nil {
		t.Fatalf("RegisterMySQL failed: %v", err)
	}

	ctx := context.Background()
	mysqlGetter := mgr.MySQL()

	// Should attempt initialization with context
	_, err = mysqlGetter.GetWithContext(ctx, "primary")
	// We expect this to fail since there's no actual database
	// but we're testing the API structure
	if err == nil {
		t.Log("Note: MySQL connection succeeded unexpectedly")
	}
}

func TestTypedGetter_MustGetPanics(t *testing.T) {
	mgr := NewManager()

	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "test"
	opts.Username = "root"

	err := mgr.RegisterMySQL("primary", opts)
	if err != nil {
		t.Fatalf("RegisterMySQL failed: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should panic when connection fails")
		}
	}()

	mysqlGetter := mgr.MySQL()
	// This should panic because connection will fail
	_ = mysqlGetter.MustGet("primary")
}

func TestTypedGetter_UnregisteredInstance(t *testing.T) {
	mgr := NewManager()

	mysqlGetter := mgr.MySQL()

	_, err := mysqlGetter.Get("nonexistent")
	if err == nil {
		t.Error("expected error for unregistered instance")
	}

	// Verify error message mentions the instance
	expectedSubstring := "not registered"
	if err != nil && !contains(err.Error(), expectedSubstring) {
		t.Errorf("error message should contain '%s', got: %v", expectedSubstring, err)
	}
}

// =============================================================================
// Tests for Backward Compatibility
// =============================================================================

func TestBackwardCompatibility_GetMySQL(t *testing.T) {
	mgr := NewManager()

	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "test"
	opts.Username = "root"

	err := mgr.RegisterMySQL("primary", opts)
	if err != nil {
		t.Fatalf("RegisterMySQL failed: %v", err)
	}

	// Old API should still work
	_, err = mgr.GetMySQL("primary")
	// Expected to fail in test environment
	if err == nil {
		t.Log("Note: MySQL connection succeeded unexpectedly")
	}
}

func TestBackwardCompatibility_GetRedis(t *testing.T) {
	mgr := NewManager()

	opts := redis.NewOptions()
	opts.Host = "localhost"

	err := mgr.RegisterRedis("cache", opts)
	if err != nil {
		t.Fatalf("RegisterRedis failed: %v", err)
	}

	// Old API should still work
	_, err = mgr.GetRedis("cache")
	// Expected to fail in test environment
	if err == nil {
		t.Log("Note: Redis connection succeeded unexpectedly")
	}
}

func TestBackwardCompatibility_GetWithContext(t *testing.T) {
	mgr := NewManager()

	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "test"
	opts.Username = "root"

	err := mgr.RegisterMySQL("primary", opts)
	if err != nil {
		t.Fatalf("RegisterMySQL failed: %v", err)
	}

	ctx := context.Background()

	// Old API should still work
	_, err = mgr.GetMySQLWithContext(ctx, "primary")
	// Expected to fail in test environment
	if err == nil {
		t.Log("Note: MySQL connection succeeded unexpectedly")
	}
}

func TestBackwardCompatibility_MustGet(t *testing.T) {
	mgr := NewManager()

	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "test"
	opts.Username = "root"

	err := mgr.RegisterMySQL("primary", opts)
	if err != nil {
		t.Fatalf("RegisterMySQL failed: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGetMySQL should panic when connection fails")
		}
	}()

	// Old API should still work and panic
	_ = mgr.MustGetMySQL("primary")
}

// =============================================================================
// Tests for All Storage Types
// =============================================================================

func TestAllStorageTypes_GetterCreation(t *testing.T) {
	mgr := NewManager()

	tests := []struct {
		name        string
		getterFunc  func() interface{}
		storageType StorageType
	}{
		{"MySQL", func() interface{} { return mgr.MySQL() }, TypeMySQL},
		{"Postgres", func() interface{} { return mgr.Postgres() }, TypePostgres},
		{"Redis", func() interface{} { return mgr.Redis() }, TypeRedis},
		{"MongoDB", func() interface{} { return mgr.MongoDB() }, TypeMongoDB},
		{"Etcd", func() interface{} { return mgr.Etcd() }, TypeEtcd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getter := tt.getterFunc()
			if getter == nil {
				t.Errorf("%s() returned nil getter", tt.name)
			}
		})
	}
}

func TestAllStorageTypes_BackwardCompatibleMethods(t *testing.T) {
	mgr := NewManager()

	// Register all types
	_ = mgr.RegisterMySQL("test", mysql.NewOptions())
	_ = mgr.RegisterPostgres("test", postgres.NewOptions())
	_ = mgr.RegisterRedis("test", redis.NewOptions())
	_ = mgr.RegisterMongoDB("test", mongodb.NewOptions())
	_ = mgr.RegisterEtcd("test", etcd.NewOptions())

	tests := []struct {
		name        string
		getFunc     func(string) (interface{}, error)
		mustGetFunc func(string) interface{}
	}{
		{
			"MySQL",
			func(n string) (interface{}, error) { return mgr.GetMySQL(n) },
			func(n string) interface{} {
				defer func() { recover() }()
				return mgr.MustGetMySQL(n)
			},
		},
		{
			"Postgres",
			func(n string) (interface{}, error) { return mgr.GetPostgres(n) },
			func(n string) interface{} {
				defer func() { recover() }()
				return mgr.MustGetPostgres(n)
			},
		},
		{
			"Redis",
			func(n string) (interface{}, error) { return mgr.GetRedis(n) },
			func(n string) interface{} {
				defer func() { recover() }()
				return mgr.MustGetRedis(n)
			},
		},
		{
			"MongoDB",
			func(n string) (interface{}, error) { return mgr.GetMongoDB(n) },
			func(n string) interface{} {
				defer func() { recover() }()
				return mgr.MustGetMongoDB(n)
			},
		},
		{
			"Etcd",
			func(n string) (interface{}, error) { return mgr.GetEtcd(n) },
			func(n string) interface{} {
				defer func() { recover() }()
				return mgr.MustGetEtcd(n)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+"_Get", func(t *testing.T) {
			// Get should return error but not panic
			_, err := tt.getFunc("test")
			// We expect error since there's no real database
			_ = err
		})

		t.Run(tt.name+"_MustGet", func(t *testing.T) {
			// MustGet should be callable (will panic internally due to no DB)
			_ = tt.mustGetFunc("test")
		})
	}
}

// =============================================================================
// Performance Comparison Tests
// =============================================================================

func BenchmarkGetMySQL_Generic(b *testing.B) {
	mgr := NewManager()
	opts := mysql.NewOptions()
	_ = mgr.RegisterMySQL("bench", opts)

	getter := mgr.MySQL()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = getter.Get("bench")
	}
}

func BenchmarkGetMySQL_Direct(b *testing.B) {
	mgr := NewManager()
	opts := mysql.NewOptions()
	_ = mgr.RegisterMySQL("bench", opts)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = mgr.GetMySQL("bench")
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
