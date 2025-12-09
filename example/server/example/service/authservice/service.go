// Package authservice provides the authentication service.
package authservice

// Service implements the auth business logic.
type Service struct{}

// NewService creates a new auth service.
func NewService() *Service {
	return &Service{}
}

// ServiceName returns the service name.
func (s *Service) ServiceName() string {
	return "auth"
}
