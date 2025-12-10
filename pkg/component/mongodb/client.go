package mongodb

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	mongoopts "go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kart-io/sentinel-x/pkg/component/storage"
)

// Client wraps mongo.Client with storage.Client interface implementation.
// It provides a unified MongoDB client that implements the standard storage
// interface while exposing the underlying MongoDB driver for advanced usage.
//
// Example usage:
//
//	opts := NewOptions()
//	opts.Host = "localhost"
//	opts.Database = "mydb"
//
//	client, err := New(opts)
//	if err != nil {
//	    log.Fatalf("failed to create MongoDB client: %v", err)
//	}
//	defer client.Close()
//
//	// Use MongoDB driver directly
//	collection := client.Collection("users")
//	collection.InsertOne(ctx, bson.M{"name": "John"})
type Client struct {
	client   *mongo.Client
	database *mongo.Database
	opts     *Options
}

// New creates a new MongoDB client from the provided options.
// It validates the options, builds the connection URI, and establishes a connection
// with the configured pool settings and timeouts.
//
// Returns an error if:
// - Options validation fails
// - Connection to MongoDB server fails
// - Connection pool configuration fails
func New(opts *Options) (*Client, error) {
	return NewWithContext(context.Background(), opts)
}

// NewWithContext creates a new MongoDB client with context support.
// This allows for timeout control during the connection establishment phase.
//
// The context is used for:
// - Connection establishment timeout
// - Initial ping verification
//
// Returns an error if:
// - Options validation fails
// - Connection to MongoDB server fails
// - Initial ping fails
// - Connection pool configuration fails
func NewWithContext(ctx context.Context, opts *Options) (*Client, error) {
	if opts == nil {
		return nil, fmt.Errorf("mongodb options cannot be nil")
	}

	// Validate options
	if err := validateOptions(opts); err != nil {
		return nil, fmt.Errorf("invalid mongodb options: %w", err)
	}

	// Build URI
	uri := BuildURI(opts)

	// Create client options
	clientOpts := mongoopts.Client().ApplyURI(uri)

	// Apply connection pool settings
	if opts.MaxPoolSize > 0 {
		clientOpts.SetMaxPoolSize(opts.MaxPoolSize)
	}
	if opts.MinPoolSize > 0 {
		clientOpts.SetMinPoolSize(opts.MinPoolSize)
	}
	if opts.MaxIdleTime > 0 {
		clientOpts.SetMaxConnIdleTime(opts.MaxIdleTime)
	}
	if opts.MaxConnIdleTime > 0 {
		clientOpts.SetMaxConnIdleTime(opts.MaxConnIdleTime)
	}

	// Apply timeout settings
	if opts.ConnectTimeout > 0 {
		clientOpts.SetConnectTimeout(opts.ConnectTimeout)
	}
	if opts.SocketTimeout > 0 {
		clientOpts.SetSocketTimeout(opts.SocketTimeout)
	}
	if opts.ServerSelectionTimeout > 0 {
		clientOpts.SetServerSelectionTimeout(opts.ServerSelectionTimeout)
	}

	// Apply direct connection
	if opts.Direct {
		clientOpts.SetDirect(opts.Direct)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	// Get database
	var db *mongo.Database
	if opts.Database != "" {
		db = client.Database(opts.Database)
	}

	return &Client{
		client:   client,
		database: db,
		opts:     opts,
	}, nil
}

// Name returns the storage type identifier.
// Implements storage.Client interface.
func (c *Client) Name() string {
	return "mongodb"
}

// Ping checks if the connection to MongoDB is alive.
// It performs a lightweight ping operation to verify connectivity.
// Implements storage.Client interface.
func (c *Client) Ping(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("client is nil")
	}
	return c.client.Ping(ctx, nil)
}

// Close closes the MongoDB connection gracefully.
// It ensures all pending operations are completed and resources are released.
// This method is idempotent and safe to call multiple times.
// Implements storage.Client interface.
func (c *Client) Close() error {
	if c.client == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.client.Disconnect(ctx)
}

// Health returns a HealthChecker function for MongoDB health monitoring.
// The returned function can be called to verify the MongoDB connection status.
// Implements storage.Client interface.
func (c *Client) Health() storage.HealthChecker {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return c.Ping(ctx)
	}
}

// Database returns the default database.
// If no database was specified in options, this returns nil.
//
// Example:
//
//	db := client.Database()
//	collection := db.Collection("users")
func (c *Client) Database() *mongo.Database {
	return c.database
}

// DatabaseByName returns a database by name.
// This allows access to databases other than the default.
//
// Example:
//
//	db := client.DatabaseByName("analytics")
//	collection := db.Collection("events")
func (c *Client) DatabaseByName(name string) *mongo.Database {
	if c.client == nil {
		return nil
	}
	return c.client.Database(name)
}

// Collection returns a collection from the default database.
// If no default database was set, this will panic.
//
// Example:
//
//	collection := client.Collection("users")
//	collection.InsertOne(ctx, bson.M{"name": "John"})
func (c *Client) Collection(name string) *mongo.Collection {
	if c.database == nil {
		panic("no default database set, use CollectionFromDatabase instead")
	}
	return c.database.Collection(name)
}

// CollectionFromDatabase returns a collection from a specific database.
// This is useful when working with multiple databases.
//
// Example:
//
//	collection := client.CollectionFromDatabase("analytics", "events")
//	collection.Find(ctx, bson.M{})
func (c *Client) CollectionFromDatabase(dbName, collName string) *mongo.Collection {
	return c.client.Database(dbName).Collection(collName)
}

// Raw returns the underlying mongo.Client.
// This provides access to the full MongoDB driver functionality
// for operations not exposed by the wrapper.
//
// Example:
//
//	mongoClient := client.Raw()
//	databases, err := mongoClient.ListDatabaseNames(ctx, bson.M{})
func (c *Client) Raw() *mongo.Client {
	return c.client
}

// validateOptions validates MongoDB options before creating the client.
// It ensures that required fields are set and values are reasonable.
func validateOptions(opts *Options) error {
	// If URI is provided, minimal validation needed
	if opts.URI != "" {
		return nil
	}

	// Otherwise validate host/port configuration
	if opts.Host == "" {
		return fmt.Errorf("host is required when URI is not provided")
	}
	if opts.Port <= 0 || opts.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}
