package casbin

import (
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestGormIntegration(t *testing.T) {
	// 1. Setup in-memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// 2. Create temporary model file
	modelText := `
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
	tmpModel, err := os.CreateTemp("", "rbac_model.conf")
	assert.NoError(t, err)
	defer os.Remove(tmpModel.Name())

	_, err = tmpModel.WriteString(modelText)
	assert.NoError(t, err)
	tmpModel.Close()

	// 3. Initialize Service
	svc, err := NewServiceWithGorm(db, tmpModel.Name())
	assert.NoError(t, err)

	// 4. Test Policy Management
	// Add user-role mapping
	success, err := svc.AddGroupingPolicy("alice", "admin")
	assert.NoError(t, err)
	assert.True(t, success)

	// Add permission for role
	success, err = svc.AddPolicy("admin", "data1", "read")
	assert.NoError(t, err)
	assert.True(t, success)

	// 5. Test Enforcement
	// Alice should have read access to data1 (via admin role)
	allowed, err := svc.Enforce("alice", "data1", "read")
	assert.NoError(t, err)
	assert.True(t, allowed, "alice should be able to read data1")

	// Alice should not have write access
	allowed, err = svc.Enforce("alice", "data1", "write")
	assert.NoError(t, err)
	assert.False(t, allowed, "alice should not be able to write data1")

	// Bob should not have access
	allowed, err = svc.Enforce("bob", "data1", "read")
	assert.NoError(t, err)
	assert.False(t, allowed, "bob should not have access")
}
