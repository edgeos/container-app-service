package main

import (
	"flag"
	"log"
	"runtime"

	"github.build.ge.com/PredixEdgeOS/container-app-service/cappsdversion"
	"github.build.ge.com/PredixEdgeOS/container-app-service/config"
	"github.build.ge.com/PredixEdgeOS/container-app-service/handlers"
)

func main() {
	var path string
	flag.StringVar(&path, "config", "", "Configuration file path")

	printVersion := flag.Bool("version", false, "Print version information")
	printHelp := flag.Bool("help", false, "Print usage")

	flag.Parse()

	var cfg config.Config
	var err error

	if *printVersion {
		cappsdversion.PrintVersion()
		return
	}
	if *printHelp {
		flag.Usage()
		return
	}
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
