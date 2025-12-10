# RBAC Audit Logging

## Overview

The RBAC module includes comprehensive audit logging functionality to track all permission-related changes. This feature allows you to monitor who made changes, what was changed, and when it happened.

## Features

- Track all role and permission changes
- Record user role assignments and revocations
- Monitor role hierarchy modifications
- Optional and non-intrusive (disabled by default)
- Extensible through custom audit logger implementations
- Zero performance impact when disabled

## Audit Event Types

The following events are automatically logged when audit logging is enabled:

| Event Type | Description |
|------------|-------------|
| `role_added` | A new role was created |
| `role_removed` | A role was deleted |
| `role_updated` | A role's permissions were modified |
| `permission_added` | A permission was added to a role |
| `permission_removed` | A permission was removed from a role |
| `user_role_assigned` | A role was assigned to a user |
| `user_role_revoked` | A role was revoked from a user |
| `role_hierarchy_changed` | A role's parent hierarchy was modified |

## Usage

### Basic Usage with Default Logger

```go
package main

import (
    "github.com/kart-io/sentinel-x/pkg/security/authz"
    "github.com/kart-io/sentinel-x/pkg/security/authz/rbac"
)

func main() {
    // Create RBAC with audit logging disabled (default)
    r := rbac.New()

    // All operations work normally, no audit logs
    _ = r.AddRole("admin", authz.NewPermission("*", "*"))
}
```

### Custom Audit Logger Implementation

```go
package main

import (
    "fmt"
    "time"

    "github.com/kart-io/sentinel-x/pkg/security/authz"
    "github.com/kart-io/sentinel-x/pkg/security/authz/rbac"
)

// CustomAuditLogger implements the AuditLogger interface
type CustomAuditLogger struct {
    // Add your logging backend here (database, file, etc.)
}

func (l *CustomAuditLogger) Log(event rbac.AuditEvent) {
    // Implement your audit logging logic
    fmt.Printf("[AUDIT] %s | Type: %s | Actor: %s | Target: %s | Details: %v\n",
        event.Timestamp.Format(time.RFC3339),
        event.Type,
        event.Actor,
        event.Target,
        event.Details,
    )
}

func main() {
    // Create RBAC with custom audit logger
    auditLogger := &CustomAuditLogger{}
    r := rbac.New(rbac.WithAuditLogger(auditLogger))

    // All permission changes will be logged
    _ = r.AddRole("admin", authz.NewPermission("*", "*"))
    _ = r.AssignRole("user-123", "admin")
    _ = r.RevokeRole("user-123", "admin")
    _ = r.RemoveRole("admin")
}
```

### Using the Built-in Logger Package

```go
package main

import (
    "github.com/kart-io/sentinel-x/pkg/security/authz"
    "github.com/kart-io/sentinel-x/pkg/security/authz/rbac"
)

func main() {
    // The defaultAuditLogger uses the kart-io/logger package
    // Create a simple wrapper that uses it
    type loggerAudit struct{}

    func (l *loggerAudit) Log(event rbac.AuditEvent) {
        logger.Infow("rbac audit",
            "type", event.Type,
            "timestamp", event.Timestamp,
            "actor", event.Actor,
            "target", event.Target,
            "details", event.Details,
        )
    }

    r := rbac.New(rbac.WithAuditLogger(&loggerAudit{}))

    // Audit logs will be written using the logger package
    _ = r.AddRole("editor", authz.NewPermission("posts", "*"))
}
```

## Audit Event Structure

Each audit event contains the following information:

```go
type AuditEvent struct {
    Type      AuditEventType         // Type of the event
    Timestamp time.Time              // When the event occurred
    Actor     string                 // User who performed the action
    Target    string                 // Object being operated on
    Details   map[string]interface{} // Additional event details
}
```

### Event Details by Type

#### role_added

```json
{
    "permissions": [...],
    "count": 3
}
```

#### role_removed

```json
{
    "permissions": [...]
}
```

#### user_role_assigned

```json
{
    "role": "admin"
}
```

#### user_role_revoked

```json
{
    "role": "admin"
}
```

#### role_hierarchy_changed

```json
{
    "old_parents": ["parent1", "parent2"],
    "new_parents": ["parent3"]
}
```

## Performance Considerations

- **Disabled by default**: No performance overhead when audit logging is not enabled
- **Optional feature**: Use `WithAuditLogger()` option only when needed
- **Async logging**: Consider implementing asynchronous logging in your custom logger for high-throughput scenarios
- **Minimal overhead**: Audit logging only adds a simple nil check and function call when enabled

## Integration with Persistence Layer

If you're using a persistent store (database, Redis, etc.), you can integrate audit logging:

```go
type DatabaseAuditLogger struct {
    db *sql.DB
}

func (l *DatabaseAuditLogger) Log(event rbac.AuditEvent) {
    // Store audit event in database
    _, err := l.db.Exec(`
        INSERT INTO rbac_audit_log
        (event_type, timestamp, actor, target, details)
        VALUES (?, ?, ?, ?, ?)
    `,
        event.Type,
        event.Timestamp,
        event.Actor,
        event.Target,
        json.Marshal(event.Details),
    )
    if err != nil {
        // Handle error
    }
}

// Use it with RBAC
r := rbac.New(
    rbac.WithStore(myStore),
    rbac.WithAuditLogger(&DatabaseAuditLogger{db: db}),
)
```

## Best Practices

1. **Always log in production**: Enable audit logging in production environments to track security-related changes

2. **Implement async logging**: For high-throughput systems, use buffered channels or message queues

3. **Include context**: Add relevant context (user ID, IP address, session ID) to the Actor field

4. **Retention policy**: Implement log rotation and retention policies for audit logs

5. **Monitor audit logs**: Set up alerts for suspicious activities (mass role changes, unauthorized modifications)

6. **Compliance**: Use audit logs to meet compliance requirements (SOC 2, HIPAA, GDPR, etc.)

## Testing

The module includes comprehensive tests for audit logging:

```bash
# Run audit logging tests
go test -v ./pkg/security/authz/rbac/... -run TestAuditLogger

# Run all RBAC tests
go test -v ./pkg/security/authz/rbac/...
```

## Example Output

When using the default logger, audit logs appear as structured JSON:

```json
{
    "timestamp": "2025-12-10T18:55:30.337088+08:00",
    "level": "info",
    "message": "rbac audit",
    "type": "role_added",
    "actor": "admin",
    "target": "viewer",
    "details": {
        "permissions": [
            {
                "resource": "posts",
                "action": "read",
                "effect": "allow"
            }
        ],
        "count": 1
    }
}
```

## Limitations

- The `Actor` field must be set by the caller (RBAC doesn't automatically detect the current user)
- Audit events are logged synchronously by default (implement async in your custom logger if needed)
- No built-in log rotation (implement in your custom logger)

## Future Enhancements

Potential future improvements:

- Built-in async logging support
- Structured query API for audit logs
- Integration with OpenTelemetry for distributed tracing
- Audit log signing and verification
- Compressed audit log storage
