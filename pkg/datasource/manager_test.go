package datasource

import (
	"testing"

	mysqlOpts "github.com/kart-io/sentinel-x/pkg/options/mysql"
	redisOpts "github.com/kart-io/sentinel-x/pkg/options/redis"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager()
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

func TestRegisterMySQL(t *testing.T) {
	mgr := NewManager()

	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "test"
	opts.Username = "root"

	err := mgr.RegisterMySQL("primary", opts)
	if err != nil {
		t.Fatalf("RegisterMySQL failed: %v", err)
	}

	// Duplicate registration should fail
	err = mgr.RegisterMySQL("primary", opts)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestRegisterRedis(t *testing.T) {
	mgr := NewManager()

	opts := redisOpts.NewOptions()
	opts.Host = "localhost"

	err := mgr.RegisterRedis("cache", opts)
	if err != nil {
		t.Fatalf("RegisterRedis failed: %v", err)
	}

	// Duplicate registration should fail
	err = mgr.RegisterRedis("cache", opts)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestListRegistered(t *testing.T) {
	mgr := NewManager()

	_ = mgr.RegisterMySQL("primary", mysqlOpts.NewOptions())
	_ = mgr.RegisterMySQL("replica", mysqlOpts.NewOptions())
	_ = mgr.RegisterRedis("cache", redisOpts.NewOptions())

	registered := mgr.ListRegistered()

	if len(registered[TypeMySQL]) != 2 {
		t.Errorf("expected 2 MySQL instances, got %d", len(registered[TypeMySQL]))
	}

	if len(registered[TypeRedis]) != 1 {
		t.Errorf("expected 1 Redis instance, got %d", len(registered[TypeRedis]))
	}
}

func TestMakeKey(t *testing.T) {
	key := makeKey(TypeMySQL, "primary")
	if key != "mysql:primary" {
		t.Errorf("expected 'mysql:primary', got '%s'", key)
	}
}

func TestParseKey(t *testing.T) {
	storageType, name := parseKey("mysql:primary")
	if storageType != TypeMySQL {
		t.Errorf("expected TypeMySQL, got %s", storageType)
	}
	if name != "primary" {
		t.Errorf("expected 'primary', got '%s'", name)
	}
}

func TestGlobal(t *testing.T) {
	mgr := Global()
	if mgr == nil {
		t.Fatal("Global returned nil")
	}

	// Should return same instance
	mgr2 := Global()
	if mgr != mgr2 {
		t.Error("Global should return same instance")
	}
}

func TestGetUnregisteredReturnsNil(t *testing.T) {
	mgr := NewManager()

	client, err := mgr.GetMySQL("nonexistent")
	if err == nil {
		t.Error("GetMySQL should return error for unregistered instance")
	}
	if client != nil {
		t.Error("GetMySQL should return nil client for unregistered instance")
	}

	redisClient, err := mgr.GetRedis("nonexistent")
	if err == nil {
		t.Error("GetRedis should return error for unregistered instance")
	}
	if redisClient != nil {
		t.Error("GetRedis should return nil client for unregistered instance")
	}
}
