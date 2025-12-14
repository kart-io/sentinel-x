package auth_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	drivermysql "gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/kart-io/sentinel-x/pkg/security/authz/casbin"
	"github.com/kart-io/sentinel-x/pkg/security/authz/casbin/infrastructure/mysql"
	"github.com/kart-io/sentinel-x/pkg/security/authz/casbin/infrastructure/redis"
	redisgo "github.com/redis/go-redis/v9"
)

// MockRepository is a simple in-memory repository for testing
type MockRepository struct {
	policies []*casbin.Policy
}

func (m *MockRepository) LoadPolicies(_ context.Context) ([]*casbin.Policy, error) {
	return m.policies, nil
}

func (m *MockRepository) SavePolicies(_ context.Context, policies []*casbin.Policy) error {
	m.policies = policies
	return nil
}

func (m *MockRepository) AddPolicy(_ context.Context, p *casbin.Policy) error {
	m.policies = append(m.policies, p)
	return nil
}

func (m *MockRepository) RemovePolicy(_ context.Context, p *casbin.Policy) error {
	var newPolicies []*casbin.Policy
	for _, policy := range m.policies {
		if policy.PType == p.PType && policy.V0 == p.V0 && policy.V1 == p.V1 && policy.V2 == p.V2 {
			continue
		}
		newPolicies = append(newPolicies, policy)
	}
	m.policies = newPolicies
	return nil
}

func (m *MockRepository) RemoveFilteredPolicy(_ context.Context, _ string, _ int, _ ...string) error {
	// Simplified implementation for test
	m.policies = []*casbin.Policy{}
	return nil
}

func (m *MockRepository) AuthorizeWithContext(_ context.Context, _, _, _ string, _ map[string]interface{}) (bool, error) {
	// Simplified implementation for test
	// This function is added based on the user's instruction,
	// but its body is a placeholder and refers to 'm.policies'
	// which is consistent with the MockRepository struct.
	// The original instruction had 'm.policies = []*casbin.Policy{}'
	// which is kept here for faithfulness, though it clears policies.
	m.policies = []*casbin.Policy{}
	return false, nil // Placeholder return
}

func TestPermissionService(t *testing.T) {
	// Create a mock repository
	repo := &MockRepository{}

	// Create a temporary model file for testing
	modelConf := `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`
	f, err := os.CreateTemp("", "model.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err := f.WriteString(modelConf); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	svc, err := casbin.NewPermissionService(f.Name(), repo)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Add policy
	if _, err := svc.AddPolicy("alice", "data1", "read"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddPolicy("bob", "data2", "write"); err != nil {
		t.Fatal(err)
	}

	// Check permission
	allowed, err := svc.Enforce("alice", "data1", "read")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("alice should have read access to data1")
	}

	allowed, err = svc.Enforce("alice", "data1", "write")
	if err != nil {
		t.Fatal(err)
	}
	if allowed {
		t.Error("alice should not have write access to data1")
	}

	// RBAC
	if _, err := svc.AddGroupingPolicy("charlie", "admin"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddPolicy("admin", "data1", "write"); err != nil {
		t.Fatal(err)
	}

	allowed, err = svc.Enforce("charlie", "data1", "write")
	if err != nil {
		t.Fatal(err)
	}
	if !allowed {
		t.Error("charlie (admin) should have write access to data1")
	}
}

// Example_mysql shows how to use MySQL repository
func Example_mysql() {
	// This is just an example, won't run without real DB
	if os.Getenv("REAL_DB_TEST") != "true" {
		fmt.Println("Skipping MySQL example")
		return
	}

	dsn := "user:pass@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
	db, _ := gorm.Open(drivermysql.Open(dsn), &gorm.Config{})

	repo, _ := mysql.NewRepository(db)
	svc, _ := casbin.NewPermissionService("model.conf", repo)

	_, _ = svc.AddPolicy("alice", "data1", "read")
	allowed, _ := svc.Enforce("alice", "data1", "read")
	fmt.Println(allowed)
}

// Example_redis shows how to use Redis repository
func Example_redis() {
	// This is just an example, won't run without real Redis
	if os.Getenv("REAL_REDIS_TEST") != "true" {
		fmt.Println("Skipping Redis example")
		return
	}

	rdb := redisgo.NewClient(&redisgo.Options{
		Addr: "localhost:6379",
	})

	repo := redis.NewRepository(rdb)
	svc, _ := casbin.NewPermissionService("model.conf", repo)

	// Setup Watcher
	w := redis.NewWatcher(rdb)
	svc.SetWatcher(w)
	defer w.Close()

	_, _ = svc.AddPolicy("alice", "data1", "read")
	allowed, _ := svc.Enforce("alice", "data1", "read")
	fmt.Println(allowed)
}
