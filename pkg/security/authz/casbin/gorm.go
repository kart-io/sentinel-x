package casbin

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

// NewGormEnforcer creates a new Casbin enforcer with GORM adapter
func NewGormEnforcer(db *gorm.DB, modelPath string) (*casbin.Enforcer, error) {
	// Initialize the adapter with existing GORM instance
	// This will automatically create the casbin_rule table if it doesn't exist
	a, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return nil, fmt.Errorf("failed to create gorm adapter: %w", err)
	}

	// Create the enforcer with the model and adapter
	e, err := casbin.NewEnforcer(modelPath, a)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	// Load policies from DB
	if err := e.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	return e, nil
}

// NewServiceWithGorm creates a PermissionService using GORM adapter
func NewServiceWithGorm(db *gorm.DB, modelPath string) (PermissionService, error) {
	e, err := NewGormEnforcer(db, modelPath)
	if err != nil {
		return nil, err
	}
	return &service{enforcer: e}, nil
}
