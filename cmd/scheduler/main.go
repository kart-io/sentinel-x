package main

import (
	"math/rand"
	"time"

	_ "go.uber.org/automaxprocs/maxprocs"

	"github.com/kart-io/sentinel-x/cmd/scheduler/app"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	app.NewApp().Run()
}
