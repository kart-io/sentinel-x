// Package app provides the User Center Service application.
package app

import (
	"fmt"

	"github.com/kart-io/sentinel-x/internal/bootstrap"
	"github.com/kart-io/sentinel-x/internal/user-center/router"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/echo"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
	"github.com/kart-io/sentinel-x/pkg/infra/app"
)

const (
	appName        = "sentinel-user-center"
	appDescription = `Sentinel-X User Center Service

The user center service for Sentinel-X platform.

This server provides:
  - User management
  - Authentication & Authorization
  - Profile management`
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

	bootstrapOpts := &bootstrap.BootstrapOptions{
		AppName:      appName,
		AppVersion:   app.GetVersion(),
		ServerMode:   opts.Server.Mode.String(),
		LogOpts:      opts.Log,
		ServerOpts:   opts.Server,
		JWTOpts:      opts.JWT,
		MySQLOpts:    opts.MySQL,
		RedisOpts:    opts.Redis,
		RegisterFunc: router.Register,
	}

	return bootstrap.Run(bootstrapOpts)
}

func printBanner(opts *Options) {
	fmt.Printf("Starting %s...\n", appName)
	// Simplified banner for now
}
