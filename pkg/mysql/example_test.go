package mysql_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/sentinel-x/pkg/middleware"
	"github.com/kart-io/sentinel-x/pkg/mysql"
	mysqlOpts "github.com/kart-io/sentinel-x/pkg/options/mysql"
)

// Example_basicUsage demonstrates basic MySQL client creation and usage.
func Example_basicUsage() {
	// Create MySQL options
	opts := mysqlOpts.NewOptions()
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
	defer client.Close()

	// Use the client
	fmt.Printf("Connected to MySQL: %s\n", client.Name())
}

// Example_withContext demonstrates creating a client with context timeout.
func Example_withContext() {
	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

	// Create client with 10-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mysql.NewWithContext(ctx, opts)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Verify connection
	if err := client.Ping(ctx); err != nil {
		log.Printf("Ping failed: %v", err)
	} else {
		fmt.Println("MySQL connection verified")
	}
}

// Example_healthCheck demonstrates health checking with the MySQL client.
func Example_healthCheck() {
	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

	client, err := mysql.New(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Basic health check
	status := mysql.CheckHealth(client, 5*time.Second)
	if status.Healthy {
		fmt.Printf("MySQL is healthy (latency: %v)\n", status.Latency)
	} else {
		fmt.Printf("MySQL is unhealthy: %v\n", status.Error)
	}

	// Health check with statistics
	status, stats := mysql.HealthWithStats(client, 5*time.Second)
	if status.Healthy {
		fmt.Printf("Open connections: %v\n", stats["open_connections"])
		fmt.Printf("Idle connections: %v\n", stats["idle_connections"])
	}
}

// Example_factory demonstrates using the Factory pattern.
func Example_factory() {
	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

	// Create factory
	factory := mysql.NewFactory(opts)

	// Create client from factory
	client, err := factory.Create(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	fmt.Printf("Client created via factory: %s\n", client.Name())
}

// Example_gormUsage demonstrates using the underlying GORM database.
func Example_gormUsage() {
	type User struct {
		ID   uint   `gorm:"primaryKey"`
		Name string `gorm:"size:100"`
		Age  int
	}

	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

	client, err := mysql.New(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Get GORM DB
	db := client.DB()

	// Auto migrate
	db.AutoMigrate(&User{})

	// Create records
	db.Create(&User{Name: "Alice", Age: 30})

	// Query records
	var users []User
	db.Where("age > ?", 25).Find(&users)

	fmt.Printf("Found %d users\n", len(users))
}

// Example_healthMiddleware demonstrates integration with health check middleware.
func Example_healthMiddleware() {
	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Database = "testdb"

	client, err := mysql.New(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Register with health middleware
	// Convert storage.HealthChecker to middleware.HealthChecker
	healthMgr := middleware.GetHealthManager()
	healthChecker := middleware.HealthChecker(client.Health())
	healthMgr.RegisterChecker("mysql", healthChecker)

	// The health endpoint will now include MySQL status
	fmt.Println("MySQL health check registered")
}

// Example_connectionPool demonstrates connection pool configuration.
func Example_connectionPool() {
	opts := mysqlOpts.NewOptions()
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
	defer client.Close()

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

// Example_dsnBuilder demonstrates DSN building.
func Example_dsnBuilder() {
	opts := mysqlOpts.NewOptions()
	opts.Host = "localhost"
	opts.Port = 3306
	opts.Username = "root"
	opts.Password = "secret"
	opts.Database = "mydb"

	dsn := mysql.BuildDSN(opts)
	fmt.Println("DSN built successfully")
	// DSN will be: root:secret@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local
	_ = dsn
}

// Example_errorHandling demonstrates proper error handling.
func Example_errorHandling() {
	opts := mysqlOpts.NewOptions()
	// Missing required fields to trigger validation error
	opts.Host = ""
	opts.Database = ""

	client, err := mysql.New(opts)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		return
	}
	defer client.Close()
}

// Example_multipleClients demonstrates creating multiple clients with different configurations.
func Example_multipleClients() {
	// Create base options
	baseOpts := mysqlOpts.NewOptions()
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
	defer prodClient.Close()

	devClient, _ := devFactory.Create(context.Background())
	defer devClient.Close()

	fmt.Println("Multiple clients created successfully")
}
