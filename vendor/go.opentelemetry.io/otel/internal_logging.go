// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otel // import "go.opentelemetry.io/otel"

import (
	"go.opentelemetry.io/otel/internal/global"

	"github.com/go-logr/logr"
)

// SetLogger configures the logger used internally to opentelemetry.
func SetLogger(logger logr.Logger) {
	global.SetLogger(logger)
}
