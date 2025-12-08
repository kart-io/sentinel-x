// Package main is the entry point for the example server.
package main

import (
	_ "go.uber.org/automaxprocs/maxprocs"

	"github.com/kart-io/sentinel-x/example/server/example/app"
)

func main() {
	app.NewApp().Run()
}
