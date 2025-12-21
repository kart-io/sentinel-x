// Package main is the entry point for the Sentinel-X Auth server.
package main

import (
	"github.com/kart-io/sentinel-x/internal/auth"
	_ "github.com/kart-io/sentinel-x/pkg/infra/adapter/gin"
)

func main() {
	auth.NewApp().Run()
}
