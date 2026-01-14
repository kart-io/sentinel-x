// Package http provides HTTP server options re-exported from pkg/options/server/http.
package http

import (
	options "github.com/kart-io/sentinel-x/pkg/options/server/http"
)

// Options is re-exported from pkg/options/server/http for convenience.
type Options = options.Options

// NewOptions is re-exported from pkg/options/server/http for convenience.
var NewOptions = options.NewOptions
