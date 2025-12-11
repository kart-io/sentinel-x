package mysql_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/sentinel-x/pkg/component/mysql"
)

// Example_basicUsage demonstrates basic MySQL client creation and usage.
func Example_basicUsage() {
	// Create MySQL options
	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Port = 3306
	opts.Username = "root"
	opts.Password = "password"
	opts.Database = "testdb"

	// Create client
	client, err := mysql.New(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	// Use the client
	fmt.Printf("Connected to MySQL: %s\n", client.Name())
}

// Example_withContext demonstrates creating a client with context timeout.
func Example_withContext() {
	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

	// Create client with 10-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mysql.NewWithContext(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	// Verify connection
	if err := client.Ping(ctx); err != nil {
		log.Printf("Ping failed: %v", err)
	} else {
		fmt.Println("MySQL connection verified")
	}
}

// Example_healthCheck demonstrates health checking with the MySQL client.
func Example_healthCheck() {
	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

	client, err := mysql.New(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	// Basic health check
	ctx := context.Background()
	err = client.CheckHealth(ctx)
	if err != nil {
		fmt.Printf("MySQL is unhealthy: %v\n", err)
	} else {
		fmt.Println("MySQL is healthy")
	}
}

// Example_factory demonstrates using the Factory pattern.
func Example_factory() {
	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

	// Create factory
	factory := mysql.NewFactory(opts)

	// Create client from factory
	client, err := factory.Create(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	fmt.Printf("Client created via factory: %s\n", client.Name())
}

// Example_gormUsage demonstrates using the underlying GORM database.
func Example_gormUsage() {
	type User struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"size:100"`
		Age  int
	}

	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

	client, err := mysql.New(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	// Get GORM DB
	db := client.DB()

	// Auto migrate
	_ = db.AutoMigrate(&User{})

	// Create records
	db.Create(&User{Name: "Alice", Age: 30})

	// Query records
	var users []User
	db.Where("age > ?", 25).Find(&users)

	fmt.Printf("Found %d users\n", len(users))
}

// Example_connectionPool demonstrates connection pool configuration.
func Example_connectionPool() {
	opts := mysql.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

	// Configure connection pool
	opts.MaxIdleConnections = 10
	opts.MaxOpenConnections = 100
	opts.MaxConnectionLifeTime = 10 * time.Second

	client, err := mysql.New(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = client.Close() }()

	// Get connection pool statistics
	sqlDB, err := client.SqlDB()
	if err != nil {
		log.Fatal(err)
	}

	stats := sqlDB.Stats()
	fmt.Printf("Max open connections: %d\n", stats.MaxOpenConnections)
	fmt.Printf("Open connections: %d\n", stats.OpenConnections)
	fmt.Printf("Idle connections: %d\n", stats.Idle)
}

// Example_errorHandling demonstrates proper error handling.
func Example_errorHandling() {
	opts := mysql.NewOptions()
	// Missing required fields to trigger validation error
	opts.Host = ""
	opts.Database = ""

	client, err := mysql.New(opts)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}
	defer func() { _ = client.Close() }()
}

// Example_multipleClients demonstrates creating multiple clients with different configurations.
func Example_multipleClients() {
	// Create base options
	baseOpts := mysql.NewOptions()
	baseOpts.Host = "localhost"
	baseOpts.Username = "root"

	// Create factory and clone for different databases
	factory := mysql.NewFactory(baseOpts)

	// Clone and customize for different databases
	prodFactory := factory.Clone()
	prodFactory.Options().Database = "production"

	devFactory := factory.Clone()
	devFactory.Options().Database = "development"

	// Create clients
	prodClient, _ := prodFactory.Create(context.Background())
	defer func() { _ = prodClient.Close() }()

	devClient, _ := devFactory.Create(context.Background())
	defer func() { _ = devClient.Close() }()

	fmt.Println("Multiple clients created successfully")
}
