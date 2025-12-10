// Package main is the entry point for the Sentinel-X API server.
package main

import (
	_ "go.uber.org/automaxprocs/maxprocs"

	"github.com/kart-io/sentinel-x/cmd/api/app"
)

func main() {
	app.NewApp().Run()
}
