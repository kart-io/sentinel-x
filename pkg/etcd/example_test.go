package etcd_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/kart-io/sentinel-x/pkg/etcd"
	etcdopts "github.com/kart-io/sentinel-x/pkg/options/etcd"
)

// ExampleNew demonstrates basic client creation and usage.
func ExampleNew() {
	// Create options with default values
	opts := etcdopts.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}

	// Create a new etcd client
	client, err := etcd.New(opts)
	if err != nil {
		log.Fatalf("failed to create etcd client: %v", err)
	}
	defer client.Close()

	// Verify connectivity
	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		log.Printf("ping failed: %v", err)
		return
	}

	fmt.Println("Successfully connected to etcd")
}

// ExampleNewWithContext demonstrates creating a client with custom timeout.
func ExampleNewWithContext() {
	opts := etcdopts.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}

	// Create a context with 30 second timeout for initialization
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := etcd.NewWithContext(ctx, opts)
	if err != nil {
		log.Fatalf("failed to create etcd client: %v", err)
	}
	defer client.Close()

	fmt.Println("Client created successfully")
}

// ExampleClient_CheckHealth demonstrates health checking.
func ExampleClient_CheckHealth() {
	opts := etcdopts.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}

	client, err := etcd.New(opts)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Perform comprehensive health check
	ctx := context.Background()
	status := client.CheckHealth(ctx)

	if status.Healthy {
		fmt.Printf("etcd is healthy (latency: %v)\n", status.Latency)
	} else {
		fmt.Printf("etcd is unhealthy: %v\n", status.Error)
	}
}

// ExampleClient_IsHealthy demonstrates simple boolean health check.
func ExampleClient_IsHealthy() {
	opts := etcdopts.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}

	client, err := etcd.New(opts)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	if client.IsHealthy(ctx) {
		fmt.Println("etcd cluster is healthy")
	} else {
		fmt.Println("etcd cluster is unhealthy")
	}
}

// ExampleClient_Raw demonstrates using the raw etcd client.
func ExampleClient_Raw() {
	opts := etcdopts.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}

	client, err := etcd.New(opts)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	// Get the raw etcd client for advanced operations
	rawClient := client.Raw()

	ctx := context.Background()

	// Use etcd-specific features
	_, err = rawClient.Put(ctx, "/config/app-name", "sentinel-x")
	if err != nil {
		log.Printf("put failed: %v", err)
		return
	}

	resp, err := rawClient.Get(ctx, "/config/app-name")
	if err != nil {
		log.Printf("get failed: %v", err)
		return
	}

	if len(resp.Kvs) > 0 {
		fmt.Printf("Value: %s\n", resp.Kvs[0].Value)
	}
}

// ExampleClient_KV demonstrates using the KV interface.
func ExampleClient_KV() {
	opts := etcdopts.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}

	client, err := etcd.New(opts)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	kv := client.KV()

	// Put a key-value pair
	_, err = kv.Put(ctx, "key", "value")
	if err != nil {
		log.Printf("put failed: %v", err)
		return
	}

	// Get the value
	resp, err := kv.Get(ctx, "key")
	if err != nil {
		log.Printf("get failed: %v", err)
		return
	}

	if len(resp.Kvs) > 0 {
		fmt.Printf("Value: %s\n", resp.Kvs[0].Value)
	}
}

// ExampleClient_Lease demonstrates using the Lease interface.
func ExampleClient_Lease() {
	opts := etcdopts.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}

	client, err := etcd.New(opts)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	lease := client.Lease()

	// Grant a 60-second lease
	resp, err := lease.Grant(ctx, 60)
	if err != nil {
		log.Printf("lease grant failed: %v", err)
		return
	}

	fmt.Printf("Lease ID: %d, TTL: %d seconds\n", resp.ID, resp.TTL)

	// Put a key with the lease
	kv := client.KV()
	_, err = kv.Put(ctx, "temp-key", "temp-value")
	if err != nil {
		log.Printf("put with lease failed: %v", err)
		return
	}
}

// ExampleNewFactory demonstrates using the factory pattern.
func ExampleNewFactory() {
	opts := etcdopts.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}

	// Create a factory
	factory := etcd.NewFactory(opts)

	// Create clients from the factory
	ctx := context.Background()
	client, err := factory.Create(ctx)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	fmt.Println("Client created via factory")
}

// ExampleFactory_MustCreate demonstrates using MustCreate for initialization.
func ExampleFactory_MustCreate() {
	opts := etcdopts.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}

	factory := etcd.NewFactory(opts)

	// MustCreate panics on error - suitable for initialization
	ctx := context.Background()
	client := factory.MustCreate(ctx)
	defer client.Close()

	fmt.Println("Client created successfully")
}

// ExampleClient_withAuthentication demonstrates creating an authenticated client.
func ExampleClient_withAuthentication() {
	opts := etcdopts.NewOptions()
	opts.Endpoints = []string{"localhost:2379"}
	opts.Username = "root"
	opts.Password = "secret"

	client, err := etcd.New(opts)
	if err != nil {
		log.Fatalf("failed to create authenticated client: %v", err)
	}
	defer client.Close()

	fmt.Println("Authenticated client created")
}
