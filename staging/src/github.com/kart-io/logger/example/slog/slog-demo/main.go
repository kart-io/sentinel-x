package main

import (
	"errors"

	"github.com/kart-io/logger"
)

func main() {
	var m = map[string]string{
		"name": "kart",
	}

	err := errors.New("name is not found")
	logger.Errorw("main get map name", "err", err)
	if val, ok := m["name"]; ok {
		logger.Infow("main get map name", "name", val)
	} else {
		err := errors.New("name is not found")
		logger.Errorw("main get map name", "name", err)
	}
}
