package mongodb

import (
	"fmt"
	"net/url"
	"strings"
)

// BuildURI builds a MongoDB URI from options.
// If URI is already set in options, it returns that.
// Otherwise, it constructs a URI from host, port, username, password, etc.
func BuildURI(opts *Options) string {
	// If URI is already provided, use it
	if opts.URI != "" {
		return opts.URI
	}

	// Build URI from components
	var uri strings.Builder

	uri.WriteString("mongodb://")

	// Add credentials if provided
	if opts.Username != "" {
		uri.WriteString(url.QueryEscape(opts.Username))
		if opts.Password != "" {
			uri.WriteString(":")
			uri.WriteString(url.QueryEscape(opts.Password))
		}
		uri.WriteString("@")
	}

	// Add host and port
	uri.WriteString(opts.Host)
	if opts.Port != 0 {
		uri.WriteString(fmt.Sprintf(":%d", opts.Port))
	}

	// Add database if provided
	if opts.Database != "" {
		uri.WriteString("/")
		uri.WriteString(opts.Database)
	} else {
		uri.WriteString("/")
	}

	// Add query parameters
	params := url.Values{}

	if opts.AuthSource != "" && opts.AuthSource != "admin" {
		params.Add("authSource", opts.AuthSource)
	}

	if opts.ReplicaSet != "" {
		params.Add("replicaSet", opts.ReplicaSet)
	}

	if opts.Direct {
		params.Add("directConnection", "true")
	}

	if len(params) > 0 {
		uri.WriteString("?")
		uri.WriteString(params.Encode())
	}

	return uri.String()
}
