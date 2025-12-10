// Package main is the entry point for the Sentinel-X API server.
package main

import (
	_ "go.uber.org/automaxprocs/maxprocs"

	api "github.com/kart-io/sentinel-x/internal/api"
)

func main() {
	api.NewApp().Run()
}
