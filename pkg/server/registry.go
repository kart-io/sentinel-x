package server

import (
	"sync"

	"github.com/kart-io/sentinel-x/pkg/server/service"
	"github.com/kart-io/sentinel-x/pkg/server/transport"
	"github.com/kart-io/sentinel-x/pkg/server/transport/grpc"
	"github.com/kart-io/sentinel-x/pkg/server/transport/http"
)

// Registry manages service registration for both HTTP and gRPC transports.
type Registry struct {
	mu           sync.RWMutex
	services     map[string]service.Service
	httpHandlers map[string]transport.HTTPHandler
	grpcDescs    []*transport.GRPCServiceDesc
}

// NewRegistry creates a new service registry.
func NewRegistry() *Registry {
	return &Registry{
		services:     make(map[string]service.Service),
		httpHandlers: make(map[string]transport.HTTPHandler),
		grpcDescs:    make([]*transport.GRPCServiceDesc, 0),
	}
}

// RegisterService registers a service with both HTTP and gRPC handlers.
// This is the unified registration method that allows a single service
// to be exposed through both protocols.
func (r *Registry) RegisterService(svc service.Service, httpHandler transport.HTTPHandler, grpcDesc *transport.GRPCServiceDesc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := svc.ServiceName()
	r.services[name] = svc

	if httpHandler != nil {
		r.httpHandlers[name] = httpHandler
	}

	if grpcDesc != nil {
		r.grpcDescs = append(r.grpcDescs, grpcDesc)
	}

	return nil
}

// RegisterHTTP registers only an HTTP handler for a service.
func (r *Registry) RegisterHTTP(svc service.Service, handler transport.HTTPHandler) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := svc.ServiceName()
	r.services[name] = svc
	r.httpHandlers[name] = handler

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

// ApplyToHTTP applies all HTTP handlers to the given HTTP server.
func (r *Registry) ApplyToHTTP(server *http.Server) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, handler := range r.httpHandlers {
		svc, ok := r.services[name]
		if !ok {
			continue
		}
		if err := server.RegisterHTTPHandler(svc, handler); err != nil {
			return err
		}
	}
	return nil
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
