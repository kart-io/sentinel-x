package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/kart-io/sentinel-x/pkg/component/mongodb"
)

func main() {
	// Example 1: Basic usage with host/port
	fmt.Println("=== Example 1: Basic Usage ===")
	opts := mongodb.NewOptions()
	opts.Host = "localhost"
	opts.Port = 27017
	opts.Database = "testdb"
	opts.Username = "admin"
	// Password will be read from MONGODB_PASSWORD environment variable

	client, err := mongodb.New(opts)
	if err != nil {
		log.Printf("Failed to create MongoDB client: %v\n", err)
		// In production, handle this error appropriately
	} else {
		defer func() { _ = client.Close() }()

		// Health check
		ctx := context.Background()
		if err := client.Ping(ctx); err != nil {
			log.Printf("MongoDB unhealthy: %v\n", err)
		} else {
			fmt.Println("MongoDB connection healthy!")
		}

		// Use storage.Client interface
		fmt.Printf("Storage type: %s\n", client.Name())

		// Use health checker
		healthChecker := client.Health()
		if err := healthChecker(); err != nil {
			log.Printf("Health check failed: %v\n", err)
		} else {
			fmt.Println("Health check passed!")
		}
	}

	// Example 2: Using URI
	fmt.Println("\n=== Example 2: Using URI ===")
	opts2 := mongodb.NewOptions()
	opts2.URI = "mongodb://localhost:27017/testdb"

	client2, err := mongodb.New(opts2)
	if err != nil {
		log.Printf("Failed to create MongoDB client: %v\n", err)
	} else {
		defer func() { _ = client2.Close() }()
		fmt.Println("Connected using URI!")
	}

	// Example 3: Using Factory Pattern
	fmt.Println("\n=== Example 3: Factory Pattern ===")
	opts3 := mongodb.NewOptions()
	opts3.Host = "localhost"
	opts3.Database = "myapp"

	factory := mongodb.NewFactory(opts3)
	client3, err := factory.Create(context.Background())
	if err != nil {
		log.Printf("Failed to create MongoDB client via factory: %v\n", err)
	} else {
		defer func() { _ = client3.Close() }()
		fmt.Println("Created client via factory!")
	}

	// Example 4: Working with Collections (requires MongoDB to be running)
	fmt.Println("\n=== Example 4: Collection Operations ===")
	if client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Get collection
		collection := client.Collection("users")

		// Insert document
		doc := bson.M{
			"name":      "Alice",
			"age":       30,
			"createdAt": time.Now(),
		}

		result, err := collection.InsertOne(ctx, doc)
		if err != nil {
			log.Printf("Failed to insert document: %v\n", err)
		} else {
			fmt.Printf("Inserted document with ID: %v\n", result.InsertedID)
		}
	}

	// Example 5: Multiple Databases
	fmt.Println("\n=== Example 5: Multiple Databases ===")
	if client != nil {
		// Access different databases
		analyticsDB := client.DatabaseByName("analytics")
		eventsCollection := analyticsDB.Collection("events")
		fmt.Printf("Accessed collection: %s\n", eventsCollection.Name())
	}

	// Example 6: Connection Pool Configuration
	fmt.Println("\n=== Example 6: Connection Pool ===")
	opts6 := mongodb.NewOptions()
	opts6.Host = "localhost"
	opts6.MaxPoolSize = 100
	opts6.MinPoolSize = 10
	opts6.MaxIdleTime = 10 * time.Minute
	opts6.MaxConnIdleTime = 5 * time.Minute
	opts6.ConnectTimeout = 10 * time.Second
	opts6.SocketTimeout = 30 * time.Second

	fmt.Printf("Pool configuration: Max=%d, Min=%d\n", opts6.MaxPoolSize, opts6.MinPoolSize)

	// Example 7: Replica Set Configuration
	fmt.Println("\n=== Example 7: Replica Set ===")
	opts7 := mongodb.NewOptions()
	opts7.Host = "mongo1.example.com,mongo2.example.com,mongo3.example.com"
	opts7.ReplicaSet = "rs0"
	opts7.Database = "myapp"
	opts7.AuthSource = "admin"

	fmt.Printf("Replica set: %s\n", opts7.ReplicaSet)
}
