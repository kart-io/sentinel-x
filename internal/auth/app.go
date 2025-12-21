// Package auth provides the auth service application.
package auth

import (
	"fmt"

	"github.com/kart-io/sentinel-x/internal/bootstrap"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
)

const (
	appName        = "sentinel-auth"
	appDescription = `Sentinel-X Auth Service

The auth service for Sentinel-X platform.

This server provides:
  - Authentication & Authorization`
)

// NewApp creates a new application instance.
func NewApp() *app.App {
	opts := NewOptions()

	return app.NewApp(
		app.WithName(appName),
		app.WithDescription(appDescription),
		app.WithOptions(opts),
		app.WithRunFunc(func() error {
			return Run(opts)
		}),
	)
}

// Run runs the User Center Service with the given options.
func Run(opts *Options) error {
	printBanner(opts)

	bootstrapOpts := &bootstrap.Options{
		AppName:    appName,
		AppVersion: app.GetVersion(),
		ServerMode: opts.Server.Mode.String(),
		LogOpts:    opts.Log,
		ServerOpts: opts.Server,
		JWTOpts:    opts.JWT,
		MySQLOpts:  opts.MySQL,
		RedisOpts:  opts.Redis,
	}

	return bootstrap.Run(bootstrapOpts)
}

func printBanner(_ *Options) {
	fmt.Printf("Starting %s...\n", appName)
	// Simplified banner for now
}
