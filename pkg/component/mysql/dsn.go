package mysql

import (
	"fmt"
	"net/url"
)

// BuildDSN creates a MySQL Data Source Name (DSN) from the provided options.
// The DSN format is: username:password@tcp(host:port)/database?params
//
// SECURITY NOTE: This function properly escapes the password to prevent
// DSN injection attacks when passwords contain special characters.
//
// Example:
//
//	opts := &Options{
//	    Host:     "localhost",
//	    Port:     3306,
//	    Username: "root",
//	    Password: "secret",
//	    Database: "mydb",
//	}
//	dsn := BuildDSN(opts)
//	// Returns: root:secret@tcp(localhost:3306)/mydb?charset=utf8mb4&parseTime=True&loc=Local
func BuildDSN(opts *Options) string {
	// Escape password to handle special characters safely.
	// Characters like @, /, :, etc. in passwords would break DSN parsing without escaping.
	escapedPassword := url.QueryEscape(opts.Password)

	// Build DSN according to MySQL driver format
	// username:password@tcp(host:port)/database?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		opts.Username,
		escapedPassword,
		opts.Host,
		opts.Port,
		opts.Database,
	)
	return dsn
}

// buildDSN is an alias for BuildDSN for internal use.
// Deprecated: Use BuildDSN instead.
func buildDSN(opts *Options) string {
	return BuildDSN(opts)
}
