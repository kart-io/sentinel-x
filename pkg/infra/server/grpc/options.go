// Package grpc provides gRPC server configuration options.
//
// This file re-exports types from pkg/options/server/grpc for backward compatibility.
package grpc

import (
	options "github.com/kart-io/sentinel-x/pkg/options/server/grpc"
)

// Options contains gRPC server configuration.
type Options = options.Options

// Option is a function that configures Options.
type Option = options.Option

// NewOptions creates a new Options with default values.
var NewOptions = options.NewOptions

// Option functions re-exports.
var (
	WithAddr           = options.WithAddr
	WithTimeout        = options.WithTimeout
	WithMaxRecvMsgSize = options.WithMaxRecvMsgSize
	WithMaxSendMsgSize = options.WithMaxSendMsgSize
	WithReflection     = options.WithReflection
)
