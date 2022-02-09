package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

const (
	appName = "konvahti"
	extraHelp = `
  config [config2 config3 ...]
        Location of the konvahti configuration files

Example usage:
  # Run konvahti with a single configuration
  konvahti config.yaml

  # Run konvahti with multiple configurations
  konvahti config1.yaml config2.yaml

  # Run konvahti with configurations loaded from STDIN
  konvahti -

  # Run konvahti with custom log configuration file
  konvahti -logConfig logconfig.yaml config.yaml

Environment variables:
  KONVAHTI_LOG_LEVEL
        The lowest priority level logs to include in the log output:
        trace, debug, info, warn, error, fatal, panic, disabled

  KONVAHTI_LOG_ENABLEPRETTYLOGGING
        When set to 'true', use text log output instead of JSON.

  KONVAHTI_LOG_OUTPUTSTREAM
        Stream to write logs to: stdout, stderr.
        Default: stderr

  KONVAHTI_LOG_TIMESTAMPFORMAT
        Format of the log timestamps. Uses Go's time syntax.

  KONVAHTI_LOG_TIMESTAMPFIELDNAME
        Name of the timestamp field in JSON log output.`
)

type cliParams struct {
	fls               *flag.FlagSet
	logConfigFilename string
	configFilenames   []string
}

func (c *cliParams) setup() {
	c.fls = flag.NewFlagSet(appName, flag.ContinueOnError)
	c.fls.StringVar(&c.logConfigFilename, "logConfig", "", "Location of the log configuration file")
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

func (c *cliParams) parse(args []string) error {
	if err := c.fls.Parse(args); err != nil {
		return err
	}
	c.configFilenames = c.fls.Args()
	return nil
}

func (c *cliParams) validate() error {
	if len(c.configFilenames) == 0 {
		return fmt.Errorf("no configurations specified")
	}
	return nil
}

func (c *cliParams) printUsage() {
	c.fls.Usage()
}
