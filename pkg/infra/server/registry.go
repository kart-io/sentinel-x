package server

import (
	"sync"

	"github.com/kart-io/sentinel-x/pkg/infra/server/service"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	"github.com/kart-io/sentinel-x/pkg/infra/server/transport/grpc"
)

// Registry manages service registration for gRPC transports.
type Registry struct {
	mu        sync.RWMutex
	services  map[string]service.Service
	grpcDescs []*transport.GRPCServiceDesc
}

// NewRegistry creates a new service registry.
func NewRegistry() *Registry {
	return &Registry{
		services:  make(map[string]service.Service),
		grpcDescs: make([]*transport.GRPCServiceDesc, 0),
	}
}

// RegisterService registers a service with gRPC handlers.
func (r *Registry) RegisterService(svc service.Service, grpcDesc *transport.GRPCServiceDesc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := svc.ServiceName()
	r.services[name] = svc

	if grpcDesc != nil {
		r.grpcDescs = append(r.grpcDescs, grpcDesc)
	}

	return nil
}

// RegisterGRPC registers only a gRPC handler for a service.
func (r *Registry) RegisterGRPC(svc service.Service, desc *transport.GRPCServiceDesc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := svc.ServiceName()
	r.services[name] = svc
	r.grpcDescs = append(r.grpcDescs, desc)

	return nil
}

// GetService returns a registered service by name.
func (r *Registry) GetService(name string) (service.Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	svc, ok := r.services[name]
	return svc, ok
}

// GetAllServices returns all registered services.
func (r *Registry) GetAllServices() []service.Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]service.Service, 0, len(r.services))
	for _, svc := range r.services {
		services = append(services, svc)
	}
	return services
}

// ApplyToGRPC applies all gRPC services to the given gRPC server.
func (r *Registry) ApplyToGRPC(server *grpc.Server) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, desc := range r.grpcDescs {
		if err := server.RegisterGRPCService(desc); err != nil {
			return err
		}
	}
	return nil
}
