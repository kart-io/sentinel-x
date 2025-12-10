// Package datasource provides unified management and factory functions for all storage clients.
//
// This package consolidates all storage client lifecycle management into a single entry point,
// including client creation, registration, initialization, health checks, and graceful shutdown.
//
// # Supported Storage Types
//
//   - Redis: In-memory data store for caching and session management
//   - MySQL: Relational database with GORM integration
//   - PostgreSQL: Advanced relational database with GORM integration
//   - MongoDB: Document-oriented NoSQL database
//   - Etcd: Distributed key-value store for configuration and service discovery
//
// # Direct Client Creation
//
// For simple use cases where you need a single client:
//
//	opts := datasource.NewRedisOptions()
//	opts.Host = "localhost"
//	client, err := datasource.NewRedisClient(opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
// # Multi-Instance Management
//
// For applications with multiple database instances, use the Manager:
//
//	mgr := datasource.NewManager()
//	mgr.RegisterMySQL("primary", mysqlOpts)
//	mgr.RegisterRedis("cache", redisOpts)
//	if err := mgr.InitAll(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer mgr.CloseAll()
//
//	db, _ := mgr.GetMySQL("primary")
//	cache, _ := mgr.GetRedis("cache")
package datasource

import (
	"context"

	"github.com/kart-io/sentinel-x/pkg/component/etcd"
	"github.com/kart-io/sentinel-x/pkg/component/mongodb"
	"github.com/kart-io/sentinel-x/pkg/component/mysql"
	"github.com/kart-io/sentinel-x/pkg/component/postgres"
	"github.com/kart-io/sentinel-x/pkg/component/redis"
	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// =============================================================================
// Core Interfaces (re-exported from storage package)
// =============================================================================

// Client is the base interface that all storage clients must implement.
type Client = storage.Client

// Factory is the interface for creating storage clients.
type Factory = storage.Factory

// HealthChecker is a function type for health checks.
type HealthChecker = storage.HealthChecker

// HealthStatus represents the health check result.
type HealthStatus = storage.HealthStatus

// =============================================================================
// Redis Client
// =============================================================================

// RedisClient is the Redis client type.
type RedisClient = redis.Client

// RedisOptions is the Redis configuration options type.
type RedisOptions = redis.Options

// NewRedisClient creates a new Redis client with the provided options.
//
// Example:
//
//	opts := datasource.NewRedisOptions()
//	opts.Host = "localhost"
//	opts.Port = 6379
//	client, err := datasource.NewRedisClient(opts)
func NewRedisClient(opts *RedisOptions) (*RedisClient, error) {
	return redis.New(opts)
}

// NewRedisClientWithContext creates a new Redis client with context support.
func NewRedisClientWithContext(ctx context.Context, opts *RedisOptions) (*RedisClient, error) {
	return redis.NewWithContext(ctx, opts)
}

// NewRedisFactory creates a Redis client factory for dependency injection.
func NewRedisFactory(opts *RedisOptions) Factory {
	return redis.NewFactory(opts)
}

// NewRedisOptions creates default Redis options.
func NewRedisOptions() *RedisOptions {
	return redis.NewOptions()
}

// =============================================================================
// MySQL Client
// =============================================================================

// MySQLClient is the MySQL client type.
type MySQLClient = mysql.Client

// MySQLOptions is the MySQL configuration options type.
type MySQLOptions = mysql.Options

// NewMySQLClient creates a new MySQL client with the provided options.
//
// Example:
//
//	opts := datasource.NewMySQLOptions()
//	opts.Host = "localhost"
//	opts.Database = "mydb"
//	opts.Username = "root"
//	client, err := datasource.NewMySQLClient(opts)
func NewMySQLClient(opts *MySQLOptions) (*MySQLClient, error) {
	return mysql.New(opts)
}

// NewMySQLClientWithContext creates a new MySQL client with context support.
func NewMySQLClientWithContext(ctx context.Context, opts *MySQLOptions) (*MySQLClient, error) {
	return mysql.NewWithContext(ctx, opts)
}

// NewMySQLFactory creates a MySQL client factory for dependency injection.
func NewMySQLFactory(opts *MySQLOptions) Factory {
	return mysql.NewFactory(opts)
}

// NewMySQLOptions creates default MySQL options.
func NewMySQLOptions() *MySQLOptions {
	return mysql.NewOptions()
}

// =============================================================================
// PostgreSQL Client
// =============================================================================

// PostgresClient is the PostgreSQL client type.
type PostgresClient = postgres.Client

// PostgresOptions is the PostgreSQL configuration options type.
type PostgresOptions = postgres.Options

// NewPostgresClient creates a new PostgreSQL client with the provided options.
//
// Example:
//
//	opts := datasource.NewPostgresOptions()
//	opts.Host = "localhost"
//	opts.Database = "mydb"
//	opts.Username = "postgres"
//	client, err := datasource.NewPostgresClient(opts)
func NewPostgresClient(opts *PostgresOptions) (*PostgresClient, error) {
	return postgres.New(opts)
}

// NewPostgresClientWithContext creates a new PostgreSQL client with context support.
func NewPostgresClientWithContext(ctx context.Context, opts *PostgresOptions) (*PostgresClient, error) {
	return postgres.NewWithContext(ctx, opts)
}

// NewPostgresFactory creates a PostgreSQL client factory for dependency injection.
func NewPostgresFactory(opts *PostgresOptions) Factory {
	return postgres.NewSimpleFactory(opts)
}

// NewPostgresOptions creates default PostgreSQL options.
func NewPostgresOptions() *PostgresOptions {
	return postgres.NewOptions()
}

// =============================================================================
// MongoDB Client
// =============================================================================

// MongoDBClient is the MongoDB client type.
type MongoDBClient = mongodb.Client

// MongoDBOptions is the MongoDB configuration options type.
type MongoDBOptions = mongodb.Options

// NewMongoDBClient creates a new MongoDB client with the provided options.
//
// Example:
//
//	opts := datasource.NewMongoDBOptions()
//	opts.Host = "localhost"
//	opts.Database = "mydb"
//	client, err := datasource.NewMongoDBClient(opts)
func NewMongoDBClient(opts *MongoDBOptions) (*MongoDBClient, error) {
	return mongodb.New(opts)
}

// NewMongoDBClientWithContext creates a new MongoDB client with context support.
func NewMongoDBClientWithContext(ctx context.Context, opts *MongoDBOptions) (*MongoDBClient, error) {
	return mongodb.NewWithContext(ctx, opts)
}

// NewMongoDBFactory creates a MongoDB client factory for dependency injection.
func NewMongoDBFactory(opts *MongoDBOptions) Factory {
	return mongodb.NewFactory(opts)
}

// NewMongoDBOptions creates default MongoDB options.
func NewMongoDBOptions() *MongoDBOptions {
	return mongodb.NewOptions()
}

// =============================================================================
// Etcd Client
// =============================================================================

// EtcdClient is the Etcd client type.
type EtcdClient = etcd.Client

// EtcdOptions is the Etcd configuration options type.
type EtcdOptions = etcd.Options

// NewEtcdClient creates a new Etcd client with the provided options.
//
// Example:
//
//	opts := datasource.NewEtcdOptions()
//	opts.Endpoints = []string{"localhost:2379"}
//	client, err := datasource.NewEtcdClient(opts)
func NewEtcdClient(opts *EtcdOptions) (*EtcdClient, error) {
	return etcd.New(opts)
}

// NewEtcdClientWithContext creates a new Etcd client with context support.
func NewEtcdClientWithContext(ctx context.Context, opts *EtcdOptions) (*EtcdClient, error) {
	return etcd.NewWithContext(ctx, opts)
}

// NewEtcdFactory creates an Etcd client factory for dependency injection.
func NewEtcdFactory(opts *EtcdOptions) Factory {
	return etcd.NewFactory(opts)
}

// NewEtcdOptions creates default Etcd options.
func NewEtcdOptions() *EtcdOptions {
	return etcd.NewOptions()
}
