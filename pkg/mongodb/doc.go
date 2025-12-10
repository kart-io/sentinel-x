// Package mongodb provides a MongoDB storage client implementation for the sentinel-x project.
//
// This package implements the storage.Client interface and provides a robust,
// production-ready MongoDB client built on top of the official MongoDB Go driver. It includes:
//
// - Connection pooling with configurable parameters
// - Health checking and monitoring
// - Graceful shutdown support
// - Context-aware operations
// - Support for both URI and host/port connection styles
// - Replica set and direct connection support
//
// # Basic Usage
//
// Create a MongoDB client with default options:
//
//	opts := mongodbOpts.NewOptions()
//	opts.Host = "localhost"
//	opts.Database = "myapp"
//
//	client, err := mongodb.New(opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
// # Using URI Connection
//
// Connect using a MongoDB URI:
//
//	opts := mongodbOpts.NewOptions()
//	opts.URI = "mongodb://username:password@localhost:27017/mydb?authSource=admin"
//
//	client, err := mongodb.New(opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Using with Context
//
// Create a client with timeout control:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	client, err := mongodb.NewWithContext(ctx, opts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Health Checking
//
// Basic health check:
//
//	if err := client.Ping(context.Background()); err != nil {
//	    log.Printf("MongoDB unhealthy: %v", err)
//	}
//
// Using the Health() method for integration with health check systems:
//
//	healthChecker := client.Health()
//	if err := healthChecker(); err != nil {
//	    log.Printf("Health check failed: %v", err)
//	}
//
// # Using the Factory Pattern
//
// Create clients using the factory:
//
//	factory := mongodb.NewFactory(opts)
//	client, err := factory.Create(context.Background())
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Working with Collections
//
// Access collections from the default database:
//
//	collection := client.Collection("users")
//	_, err := collection.InsertOne(ctx, bson.M{
//	    "name": "Alice",
//	    "age": 30,
//	})
//
// Access collections from a specific database:
//
//	collection := client.CollectionFromDatabase("analytics", "events")
//	cursor, err := collection.Find(ctx, bson.M{"type": "login"})
//
// # Connection Pool Configuration
//
// Configure connection pool parameters:
//
//	opts := mongodbOpts.NewOptions()
//	opts.MaxPoolSize = 100
//	opts.MinPoolSize = 10
//	opts.MaxIdleTime = 10 * time.Minute
//	opts.MaxConnIdleTime = 5 * time.Minute
//
//	client, err := mongodb.New(opts)
//
// # Timeout Configuration
//
// Configure various timeout parameters:
//
//	opts := mongodbOpts.NewOptions()
//	opts.ConnectTimeout = 10 * time.Second
//	opts.SocketTimeout = 30 * time.Second
//	opts.ServerSelectionTimeout = 30 * time.Second
//
//	client, err := mongodb.New(opts)
//
// # Replica Set Configuration
//
// Connect to a replica set:
//
//	opts := mongodbOpts.NewOptions()
//	opts.Host = "mongo1.example.com,mongo2.example.com,mongo3.example.com"
//	opts.ReplicaSet = "rs0"
//	opts.Database = "myapp"
//
//	client, err := mongodb.New(opts)
//
// # Integration with Health Middleware
//
// Register with the health check middleware:
//
//	healthMgr := middleware.GetHealthManager()
//	healthMgr.RegisterChecker("mongodb", client.Health())
//
// # Error Handling
//
// The package provides detailed error messages for common scenarios:
//
//	client, err := mongodb.New(opts)
//	if err != nil {
//	    switch {
//	    case strings.Contains(err.Error(), "invalid mongodb options"):
//	        log.Printf("Configuration error: %v", err)
//	    case strings.Contains(err.Error(), "failed to connect"):
//	        log.Printf("Connection error: %v", err)
//	    case strings.Contains(err.Error(), "failed to ping"):
//	        log.Printf("Ping error: %v", err)
//	    default:
//	        log.Printf("Unknown error: %v", err)
//	    }
//	}
//
// # Accessing the Raw MongoDB Client
//
// Get the underlying mongo.Client for advanced operations:
//
//	mongoClient := client.Raw()
//
//	// Use MongoDB driver features
//	databases, err := mongoClient.ListDatabaseNames(ctx, bson.M{})
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Thread Safety
//
// The MongoDB client is safe for concurrent use by multiple goroutines.
// The underlying MongoDB driver handles connection pooling and thread safety automatically.
//
// # Security Considerations
//
// For production use, always provide the password via the MONGODB_PASSWORD
// environment variable instead of passing it through CLI flags or configuration files:
//
//	export MONGODB_PASSWORD="your-secure-password"
//	opts := mongodbOpts.NewOptions()
//	opts.Host = "localhost"
//	opts.Username = "admin"
//	// Password is automatically read from MONGODB_PASSWORD env var
//	opts.Validate()
package mongodb
