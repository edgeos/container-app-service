package main

import (
	"flag"
	"log"
	"runtime"

	"github.build.ge.com/container-app-service/config"
	"github.build.ge.com/container-app-service/handlers"
)

func main() {
	var path string
	flag.StringVar(&path, "config", "", "Configuration file path")
	flag.Parse()

	var cfg config.Config
	var err error
	if path != "" {
		file := path + "/ecs.json"
		if cfg, err = config.NewConfig(file); err != nil {
			log.Fatalf("Error loading configuration: %s", err)
		}

		go handlers.Start(cfg)

		runtime.Goexit()

	} else {
		flag.Usage()
	}
}
