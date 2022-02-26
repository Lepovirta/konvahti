package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

const (
	appName   = "konvahti"
	extraHelp = `
Example usage:
  # Run konvahti with a configuration file
  konvahti -config config.yaml

  # Run konvahti with configuration loaded from STDIN
  cat config.yaml | konvahti -config -`
)

type cliParams struct {
	fls            *flag.FlagSet
	configFilename string
}

func (c *cliParams) setup() {
	c.fls = flag.NewFlagSet(appName, flag.ContinueOnError)
	c.fls.StringVar(&c.configFilename, "config", "", "Location of the Konvahti configuration file")
	c.fls.SetOutput(os.Stderr)
	c.fls.Usage = func() {
		if _, err := fmt.Fprintf(c.fls.Output(), "Usage of %s:\n", c.fls.Name()); err != nil {
			log.Printf("failed to print usage: %s", err)
		}
		c.fls.PrintDefaults()
		if _, err := fmt.Fprintln(c.fls.Output(), extraHelp); err != nil {
			log.Printf("failed to print usage: %s", err)
		}
	}
}

func (c *cliParams) validate() error {
	if c.configFilename == "" {
		return fmt.Errorf("no configuration file specified")
	}
	return nil
}

func (c *cliParams) parse(args []string) error {
	if err := c.fls.Parse(args); err != nil {
		return err
	}
	return nil
}

func (c *cliParams) printUsage() {
	c.fls.Usage()
}
