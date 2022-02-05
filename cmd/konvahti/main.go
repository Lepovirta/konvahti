package main

import (
	"context"
	"flag"

	"github.com/rs/zerolog/log"
	"gitlab.com/lepovirta/konvahti/internal/env"
	"gitlab.com/lepovirta/konvahti/internal/start"
)

var logConfigFileName string

func init() {
	flag.StringVar(&logConfigFileName, "logConfig", "", "Location of the log configuration file")
}

func main() {
	flag.Parse()
	configFiles := flag.Args()

	var prg start.MainProgram
	if err := prg.Setup(
		env.RealEnv(),
		configFiles,
		logConfigFileName,
	); err != nil {
		log.Panic().Err(err).Msg("failed to init konvahti")
	}

	ctx := context.Background()
	if err := prg.Run(ctx); err != nil {
		log.Panic().Err(err).Msg("failed to run konvahti")
	}
}
