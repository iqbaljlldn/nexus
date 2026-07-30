package main

import (
	"nexus-be/pkg/config"
	"nexus-be/pkg/logger"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}
	log := logger.New(*cfg)
	defer log.Sync()

	r := InitializeRouter(log)

	r.Run()
}
