package main

import (
	"github.com/play-with-docker/play-with-docker/config"
	"github.com/play-with-docker/play-with-docker/handlers"
)

func main() {
	config.ParseFlags()
	handlers.Bootstrap()
	handlers.Register()
}
