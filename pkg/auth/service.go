package auth

import (
	"fmt"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
)

// PermissionService defines the interface for permission management
type PermissionService interface {
	// Enforce checks if a "subject" can access a "object" with an "action"
	Enforce(sub, obj, act string) (bool, error)

	// AddPolicy adds a permission rule
	AddPolicy(sub, obj, act string) (bool, error)

	// RemovePolicy removes a permission rule
	RemovePolicy(sub, obj, act string) (bool, error)

	// AddGroupingPolicy adds a role inheritance rule (user -> role)
	AddGroupingPolicy(user, role string) (bool, error)

	// RemoveGroupingPolicy removes a role inheritance rule
	RemoveGroupingPolicy(user, role string) (bool, error)

	// LoadPolicy reloads policies from storage
	LoadPolicy() error

	// SavePolicy saves policies to storage
	SavePolicy() error

	// GetEnforcer returns the underlying Casbin enforcer (use with caution)
	GetEnforcer() *casbin.Enforcer

	// SetWatcher sets the watcher for distributed synchronization
	SetWatcher(w Watcher)
}

type service struct {
	enforcer *casbin.Enforcer
	watcher  Watcher
}

// NewPermissionService creates a new permission service
func NewPermissionService(modelPath string, repo Repository) (PermissionService, error) {
	// Load model
	m, err := model.NewModelFromFile(modelPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load model: %w", err)
	}

	// Create adapter
	a := NewAdapter(repo)

	// Create enforcer
	e, err := casbin.NewEnforcer(m, a)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	// Load policies
	if err := e.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load policies: %w", err)
	}

	return &service{enforcer: e}, nil
}

func (s *service) Enforce(sub, obj, act string) (bool, error) {
	return s.enforcer.Enforce(sub, obj, act)
}

func (s *service) AddPolicy(sub, obj, act string) (bool, error) {
	res, err := s.enforcer.AddPolicy(sub, obj, act)
	if err == nil && res && s.watcher != nil {
		_ = s.watcher.Update()
	}
	return res, err
}

func (s *service) RemovePolicy(sub, obj, act string) (bool, error) {
	res, err := s.enforcer.RemovePolicy(sub, obj, act)
	if err == nil && res && s.watcher != nil {
		_ = s.watcher.Update()
	}
	return res, err
}

func (s *service) AddGroupingPolicy(user, role string) (bool, error) {
	res, err := s.enforcer.AddGroupingPolicy(user, role)
	if err == nil && res && s.watcher != nil {
		_ = s.watcher.Update()
	}
	return res, err
}

func (s *service) RemoveGroupingPolicy(user, role string) (bool, error) {
	res, err := s.enforcer.RemoveGroupingPolicy(user, role)
	if err == nil && res && s.watcher != nil {
		_ = s.watcher.Update()
	}
	return res, err
}

func (s *service) LoadPolicy() error {
	return s.enforcer.LoadPolicy()
}

func (s *service) SavePolicy() error {
	return s.enforcer.SavePolicy()
}

func (s *service) GetEnforcer() *casbin.Enforcer {
	return s.enforcer
}

func (s *service) SetWatcher(w Watcher) {
	s.watcher = w
	w.SetUpdateCallback(func(msg string) {
		_ = s.LoadPolicy()
	})
}
