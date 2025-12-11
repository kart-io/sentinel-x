// Package example demonstrates the refactored datasource manager API.
// This file shows both the old API (backward compatible) and the new generic API.
package example

import (
	"context"
	"fmt"

	"github.com/kart-io/sentinel-x/pkg/component/mysql"
	"github.com/kart-io/sentinel-x/pkg/component/redis"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
)

// OldAPIExample shows the traditional way to use the datasource manager.
// This API is still fully supported for backward compatibility.
func OldAPIExample() error {
	mgr := datasource.NewManager()

	// Register MySQL instance
	mysqlOpts := mysql.NewOptions()
	mysqlOpts.Host = "localhost"
	mysqlOpts.Database = "myapp"
	mysqlOpts.Username = "root"
	err := mgr.RegisterMySQL("primary", mysqlOpts)
	if err != nil {
		return err
	}

	// Register Redis instance
	redisOpts := redis.NewOptions()
	redisOpts.Host = "localhost"
	err = mgr.RegisterRedis("cache", redisOpts)
	if err != nil {
		return err
	}

	// Initialize all
	ctx := context.Background()
	if err := mgr.InitAll(ctx); err != nil {
		return err
	}
	defer func() { _ = mgr.CloseAll() }()

	// OLD API: Get clients using traditional methods
	db, err := mgr.GetMySQL("primary")
	if err != nil {
		return fmt.Errorf("failed to get MySQL client: %w", err)
	}

	cache, err := mgr.GetRedis("cache")
	if err != nil {
		return fmt.Errorf("failed to get Redis client: %w", err)
	}

	// Use clients
	_ = db
	_ = cache

	return nil
}

// NewGenericAPIExample shows the new generic API.
// This provides better type safety and is more maintainable.
func NewGenericAPIExample() error {
	mgr := datasource.NewManager()

	// Register instances (same as before)
	mysqlOpts := mysql.NewOptions()
	mysqlOpts.Host = "localhost"
	mysqlOpts.Database = "myapp"
	mysqlOpts.Username = "root"
	_ = mgr.RegisterMySQL("primary", mysqlOpts)
	_ = mgr.RegisterMySQL("replica", mysqlOpts)

	redisOpts := redis.NewOptions()
	redisOpts.Host = "localhost"
	_ = mgr.RegisterRedis("cache", redisOpts)

	ctx := context.Background()
	if err := mgr.InitAll(ctx); err != nil {
		return err
	}
	defer func() { _ = mgr.CloseAll() }()

	// NEW API: Get typed getters
	mysqlGetter := mgr.MySQL()
	redisGetter := mgr.Redis()

	// Get clients using the typed getters
	primaryDB, err := mysqlGetter.Get("primary")
	if err != nil {
		return fmt.Errorf("failed to get primary MySQL: %w", err)
	}

	replicaDB, err := mysqlGetter.Get("replica")
	if err != nil {
		return fmt.Errorf("failed to get replica MySQL: %w", err)
	}

	cache, err := redisGetter.Get("cache")
	if err != nil {
		return fmt.Errorf("failed to get Redis: %w", err)
	}

	// Use clients
	_ = primaryDB
	_ = replicaDB
	_ = cache

	return nil
}

// AdvancedGenericAPIExample shows advanced usage patterns.
func AdvancedGenericAPIExample() error {
	mgr := datasource.NewManager()

	// Setup (omitted for brevity)
	mysqlOpts := mysql.NewOptions()
	_ = mgr.RegisterMySQL("primary", mysqlOpts)
	_ = mgr.RegisterMySQL("replica", mysqlOpts)

	ctx := context.Background()

	// 1. Store getters for reuse
	mysqlGetter := mgr.MySQL()

	// 2. Use with context for timeout control
	primaryDB, err := mysqlGetter.GetWithContext(ctx, "primary")
	if err != nil {
		return err
	}

	// 3. Use MustGet in initialization code
	replicaDB := mysqlGetter.MustGet("replica") // Panics on error

	// 4. Pass getters to functions
	if err := processWithDB(mysqlGetter, "primary"); err != nil {
		return err
	}

	_ = primaryDB
	_ = replicaDB

	return nil
}

// processWithDB demonstrates passing a typed getter to a function.
// This enables better testability and dependency injection.
func processWithDB(getter *datasource.TypedGetter[*mysql.Client], name string) error {
	db, err := getter.Get(name)
	if err != nil {
		return err
	}

	// Use the database
	_ = db
	return nil
}

// MixedAPIExample shows that both APIs can be used together.
// This is useful during migration periods.
func MixedAPIExample() error {
	mgr := datasource.NewManager()

	mysqlOpts := mysql.NewOptions()
	_ = mgr.RegisterMySQL("primary", mysqlOpts)

	redisOpts := redis.NewOptions()
	_ = mgr.RegisterRedis("cache", redisOpts)

	ctx := context.Background()
	if err := mgr.InitAll(ctx); err != nil {
		return err
	}
	defer func() { _ = mgr.CloseAll() }()

	// Mix old and new APIs freely
	db, err := mgr.GetMySQL("primary")     // Old API
	cache, err := mgr.Redis().Get("cache") // New API
	if err != nil {
		return err
	}

	_ = db
	_ = cache

	return nil
}

// TestableDesignExample shows how the new API improves testability.
type Service struct {
	mysqlGetter *datasource.TypedGetter[*mysql.Client]
	redisGetter *datasource.TypedGetter[*redis.Client]
}

func NewService(mgr *datasource.Manager) *Service {
	return &Service{
		mysqlGetter: mgr.MySQL(),
		redisGetter: mgr.Redis(),
	}
}

func (s *Service) DoWork(ctx context.Context) error {
	// Get clients on-demand
	db, err := s.mysqlGetter.GetWithContext(ctx, "primary")
	if err != nil {
		return err
	}

	cache, err := s.redisGetter.GetWithContext(ctx, "cache")
	if err != nil {
		return err
	}

	// Use clients
	_ = db
	_ = cache

	return nil
}

// ComparisonExample shows side-by-side comparison of old vs new API.
func ComparisonExample() {
	mgr := datasource.NewManager()

	// OLD WAY: Multiple similar calls
	db1, _ := mgr.GetMySQL("primary")
	db2, _ := mgr.GetMySQL("replica")
	db3, _ := mgr.GetMySQL("analytics")
	_, _, _ = db1, db2, db3

	// NEW WAY: Get once, use multiple times
	mysqlGetter := mgr.MySQL()
	db1, _ = mysqlGetter.Get("primary")
	db2, _ = mysqlGetter.Get("replica")
	db3, _ = mysqlGetter.Get("analytics")

	// OLD WAY: With context (verbose)
	ctx := context.Background()
	db4, _ := mgr.GetMySQLWithContext(ctx, "primary")
	db5, _ := mgr.GetMySQLWithContext(ctx, "replica")
	_, _ = db4, db5

	// NEW WAY: With context (cleaner)
	db4, _ = mysqlGetter.GetWithContext(ctx, "primary")
	db5, _ = mysqlGetter.GetWithContext(ctx, "replica")

	// OLD WAY: MustGet (explicit for each storage type)
	primaryDB := mgr.MustGetMySQL("primary")
	cacheDB := mgr.MustGetRedis("cache")
	_, _ = primaryDB, cacheDB

	// NEW WAY: MustGet (unified interface)
	primaryDB = mgr.MySQL().MustGet("primary")
	cacheDB = mgr.Redis().MustGet("cache")
}
