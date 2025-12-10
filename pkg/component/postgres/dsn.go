package postgres

import (
	"fmt"
	"net/url"
	"strings"
)

// BuildDSN creates a PostgreSQL DSN (Data Source Name) from the provided options.
//
// SECURITY NOTE: This function properly escapes the password to prevent
// DSN injection attacks when passwords contain special characters.
//
// The DSN format is:
// host=<host> port=<port> user=<username> password=<password> dbname=<database> sslmode=<sslmode>
//
// Example:
//
//	host=localhost port=5432 user=postgres password=secret dbname=mydb sslmode=disable
func BuildDSN(opts *Options) string {
	if opts == nil {
		return ""
	}

	// Escape password for PostgreSQL DSN format.
	// PostgreSQL uses space-separated key=value pairs, so we need to:
	// 1. Escape single quotes by doubling them
	// 2. Wrap the password in single quotes if it contains special characters
	escapedPassword := escapePostgresValue(opts.Password)

	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		opts.Host,
		opts.Port,
		opts.Username,
		escapedPassword,
		opts.Database,
		opts.SSLMode,
	)
}

// BuildURI creates a PostgreSQL connection URI from the provided options.
// This format is useful for some drivers that prefer URI format over DSN.
//
// The URI format is:
// postgresql://username:password@host:port/database?sslmode=<sslmode>
//
// Example:
//
//	postgresql://postgres:secret@localhost:5432/mydb?sslmode=disable
func BuildURI(opts *Options) string {
	if opts == nil {
		return ""
	}

	// URL-encode the password for URI format
	encodedPassword := url.QueryEscape(opts.Password)

	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		opts.Username,
		encodedPassword,
		opts.Host,
		opts.Port,
		opts.Database,
		opts.SSLMode,
	)
}

// escapePostgresValue escapes a value for PostgreSQL DSN format.
// If the value contains spaces or special characters, it wraps the value in single quotes
// and escapes any existing single quotes by doubling them.
func escapePostgresValue(value string) string {
	// If empty, return empty quotes
	if value == "" {
		return "''"
	}

	// Check if value needs quoting
	needsQuoting := strings.ContainsAny(value, " '\\")

	if needsQuoting {
		// Escape single quotes by doubling them
		escaped := strings.ReplaceAll(value, "'", "''")
		// Escape backslashes
		escaped = strings.ReplaceAll(escaped, "\\", "\\\\")
		return "'" + escaped + "'"
	}

	return value
}
