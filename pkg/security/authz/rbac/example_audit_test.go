package rbac_test

import (
	"fmt"
	"time"

	"github.com/kart-io/sentinel-x/pkg/security/authz"
	"github.com/kart-io/sentinel-x/pkg/security/authz/rbac"
)

// CustomAuditLogger is a custom implementation of AuditLogger.
type CustomAuditLogger struct{}

func (l *CustomAuditLogger) Log(event rbac.AuditEvent) {
	fmt.Printf("[AUDIT] %s | Type: %s | Actor: %s | Target: %s | Details: %v\n",
		event.Timestamp.Format(time.RFC3339),
		event.Type,
		event.Actor,
		event.Target,
		event.Details,
	)
}

// Example demonstrates using audit logging with RBAC.
func Example_auditLogging() {
	// Create RBAC with custom audit logger
	customLogger := &CustomAuditLogger{}
	r := rbac.New(rbac.WithAuditLogger(customLogger))

	// Add roles - will be audited
	_ = r.AddRole("admin", authz.NewPermission("*", "*"))
	_ = r.AddRole("editor", authz.NewPermission("posts", "*"))

	// Assign roles - will be audited
	_ = r.AssignRole("user-1", "admin")
	_ = r.AssignRole("user-2", "editor")

	// Revoke role - will be audited
	_ = r.RevokeRole("user-2", "editor")

	// Remove role - will be audited
	_ = r.RemoveRole("editor")

	// Output will show all audit log entries
}

// simpleAuditLogger is a simple audit logger implementation.
type simpleAuditLogger struct{}

// Log implements AuditLogger interface.
func (l *simpleAuditLogger) Log(event rbac.AuditEvent) {
	// Use any logging implementation here
	fmt.Printf("Audit: %v\n", event)
}

// Example demonstrates using the default audit logger.
func Example_defaultAuditLogger() {
	// The default audit logger uses the logger package internally
	// You can also create a simple logger that uses the global logger
	logger := &simpleAuditLogger{}

	r := rbac.New(rbac.WithAuditLogger(logger))

	// All permission changes will be logged using the logger package
	_ = r.AddRole("viewer", authz.NewPermission("posts", "read"))
	_ = r.AssignRole("user-100", "viewer")

	// Logs will appear in standard logger output
}

// Example demonstrates RBAC without audit logging.
func Example_withoutAuditLogging() {
	// Create RBAC without audit logger (default behavior)
	r := rbac.New()

	// All operations work normally, but no audit logs are generated
	_ = r.AddRole("admin", authz.NewPermission("*", "*"))
	_ = r.AssignRole("user-1", "admin")

	// No audit logging overhead when not needed
}
