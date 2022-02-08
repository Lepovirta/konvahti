package main

import (
	"context"
	"flag"
	"os"

	"github.com/rs/zerolog/log"
	"gitlab.com/lepovirta/konvahti/internal/env"
	"gitlab.com/lepovirta/konvahti/internal/start"
)

func main() {
	if err := mainWithErr(); err != nil {
		if err == flag.ErrHelp {
			return
		}
		log.Panic().Err(err).Msg("failed to run konvahti")
	}
}

func mainWithErr() error {
	var params cliParams
	params.setup()

	if err := params.parse(os.Args[1:]); err != nil {
		return err
	}
	if err := params.validate(); err != nil {
		params.printUsage()
		return err
	}

	var prg start.MainProgram
	if err := prg.Setup(
		env.RealEnv(),
		params.configFilenames,
		params.logConfigFilename,
	); err != nil {
		return err
	}

	ctx := context.Background()
	if err := prg.Run(ctx); err != nil {
		return err
	}

	return nil
}
