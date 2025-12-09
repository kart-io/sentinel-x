// Package main is the entry point for the user-center server.
package main

import (
	_ "go.uber.org/automaxprocs/maxprocs"

	"github.com/kart-io/sentinel-x/example/server/user-center/app"
)

func main() {
	app.NewApp().Run()
}
