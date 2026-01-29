package etcd

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/kart-io/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Registrar handles service registration for Traefik via ETCD.
type Registrar struct {
	client      *clientv3.Client
	serviceName string
	addr        string
	rule        string
	leaseID     clientv3.LeaseID
	stopCh      chan struct{}
}

// NewRegistrar creates a new Registrar.
func NewRegistrar(client *clientv3.Client, serviceName, addr, rule string) *Registrar {
	return &Registrar{
		client:      client,
		serviceName: serviceName,
		addr:        addr,
		rule:        rule,
		stopCh:      make(chan struct{}),
	}
}

// Register registers the service to ETCD for Traefik discovery.
func (r *Registrar) Register(ctx context.Context) error {
	// 1. Create a lease with 10 seconds TTL
	leaseResp, err := r.client.Grant(ctx, 10)
	if err != nil {
		return fmt.Errorf("failed to grant lease: %w", err)
	}
	r.leaseID = leaseResp.ID

	// 2. KeepAlive the lease
	ch, err := r.client.KeepAlive(context.Background(), r.leaseID)
	if err != nil {
		return fmt.Errorf("failed to keep alive lease: %w", err)
	}

	// Monitor KeepAlive channel
	go func() {
		for {
			select {
			case <-r.stopCh:
				return
			case _, ok := <-ch:
				if !ok {
					logger.Warn("ETCD KeepAlive channel closed")
					return
				}
			}
		}
	}()

	// 3. Register Keys for Traefik
	// Traefik KV structure:
	// traefik/http/routers/<name>/rule -> <rule>
	// traefik/http/routers/<name>/service -> <name>
	// traefik/http/services/<name>/loadbalancer/servers/<id>/url -> <addr>

	// Generate a unique ID for this instance based on address
	hash := md5.Sum([]byte(r.addr))
	instanceID := hex.EncodeToString(hash[:])

	kv := clientv3.NewKV(r.client)
	ops := []clientv3.Op{
		clientv3.OpPut(fmt.Sprintf("traefik/http/routers/%s/rule", r.serviceName), r.rule, clientv3.WithLease(r.leaseID)),
		clientv3.OpPut(fmt.Sprintf("traefik/http/routers/%s/service", r.serviceName), r.serviceName, clientv3.WithLease(r.leaseID)),
		clientv3.OpPut(fmt.Sprintf("traefik/http/services/%s/loadbalancer/servers/%s/url", r.serviceName, instanceID), r.addr, clientv3.WithLease(r.leaseID)),
	}

	_, err = kv.Txn(ctx).Then(ops...).Commit()
	if err != nil {
		return fmt.Errorf("failed to register service keys: %w", err)
	}

	logger.Infow("Service registered to ETCD for Traefik",
		"service", r.serviceName,
		"addr", r.addr,
		"rule", r.rule,
	)
	return nil
}

// Close deregisters the service and stops the KeepAlive.
func (r *Registrar) Close() {
	close(r.stopCh)
	if r.leaseID != 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = r.client.Revoke(ctx, r.leaseID)
		logger.Info("Service deregistered from ETCD")
	}
}
