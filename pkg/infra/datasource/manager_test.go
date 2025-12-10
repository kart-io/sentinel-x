package datasource

import (
	"testing"

	mysqlOpts "github.com/kart-io/sentinel-x/pkg/component/mysql"
	redisOpts "github.com/kart-io/sentinel-x/pkg/component/redis"
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

func TestGetGlobal(t *testing.T) {
	// Reset global state before test
	ResetGlobal()

	mgr := GetGlobal()
	if mgr == nil {
		t.Fatal("GetGlobal returned nil")
	}

	// Should return same instance
	mgr2 := GetGlobal()
	if mgr != mgr2 {
		t.Error("GetGlobal should return same instance")
	}
}

func TestSetGlobal(t *testing.T) {
	// Reset global state before test
	ResetGlobal()

	customMgr := NewManager()
	err := SetGlobal(customMgr)
	if err != nil {
		t.Fatalf("SetGlobal failed: %v", err)
	}

	// Retrieved manager should be the same
	retrieved := GetGlobal()
	if retrieved != customMgr {
		t.Error("GetGlobal should return the custom manager set via SetGlobal")
	}

	// Second SetGlobal should fail
	err = SetGlobal(NewManager())
	if err == nil {
		t.Error("SetGlobal should return error when called twice")
	}
}

func TestSetGlobalWithNil(t *testing.T) {
	// Reset global state before test
	ResetGlobal()

	err := SetGlobal(nil)
	if err == nil {
		t.Error("SetGlobal should return error for nil manager")
	}
}

func TestMustSetGlobal(t *testing.T) {
	// Reset global state before test
	ResetGlobal()

	customMgr := NewManager()
	MustSetGlobal(customMgr)

	retrieved := GetGlobal()
	if retrieved != customMgr {
		t.Error("GetGlobal should return the custom manager set via MustSetGlobal")
	}
}

func TestMustSetGlobalPanicsOnDuplicate(t *testing.T) {
	// Reset global state before test
	ResetGlobal()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustSetGlobal should panic when called twice")
		}
	}()

	MustSetGlobal(NewManager())
	MustSetGlobal(NewManager()) // Should panic
}

func TestResetGlobal(t *testing.T) {
	// Set a global manager
	customMgr := NewManager()
	ResetGlobal()
	err := SetGlobal(customMgr)
	if err != nil {
		t.Fatalf("SetGlobal failed: %v", err)
	}

	// Reset should return the previous manager
	prev := ResetGlobal()
	if prev != customMgr {
		t.Error("ResetGlobal should return the previous manager")
	}

	// After reset, should be able to set a new manager
	newMgr := NewManager()
	err = SetGlobal(newMgr)
	if err != nil {
		t.Errorf("SetGlobal should succeed after ResetGlobal: %v", err)
	}

	retrieved := GetGlobal()
	if retrieved != newMgr {
		t.Error("GetGlobal should return the new manager after reset")
	}
}

func TestConcurrentGetGlobal(t *testing.T) {
	// Reset global state before test
	ResetGlobal()

	const numGoroutines = 100
	done := make(chan *Manager, numGoroutines)

	// Launch multiple goroutines trying to get the global manager
	for i := 0; i < numGoroutines; i++ {
		go func() {
			mgr := GetGlobal()
			done <- mgr
		}()
	}

	// Collect all results
	var managers []*Manager
	for i := 0; i < numGoroutines; i++ {
		mgr := <-done
		managers = append(managers, mgr)
	}

	// All goroutines should get the same instance
	first := managers[0]
	for i, mgr := range managers {
		if mgr != first {
			t.Errorf("goroutine %d got different manager instance", i)
		}
	}
}

func TestConcurrentSetGlobalFails(t *testing.T) {
	// Reset global state before test
	ResetGlobal()

	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	// Launch multiple goroutines trying to set the global manager
	for i := 0; i < numGoroutines; i++ {
		go func() {
			mgr := NewManager()
			err := SetGlobal(mgr)
			done <- err
		}()
	}

	// Collect all results
	var successCount int
	var errorCount int
	for i := 0; i < numGoroutines; i++ {
		err := <-done
		if err == nil {
			successCount++
		} else {
			errorCount++
		}
	}

	// Exactly one goroutine should succeed
	if successCount != 1 {
		t.Errorf("expected exactly 1 successful SetGlobal, got %d", successCount)
	}

	// All other goroutines should fail
	if errorCount != numGoroutines-1 {
		t.Errorf("expected %d failed SetGlobal, got %d", numGoroutines-1, errorCount)
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
