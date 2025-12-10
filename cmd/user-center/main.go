// Package main is the entry point for the Sentinel-X User Center Service.
package main

import (
	_ "go.uber.org/automaxprocs/maxprocs"

	usercenter "github.com/kart-io/sentinel-x/internal/user-center"
)

func main() {
	usercenter.NewApp().Run()
}
