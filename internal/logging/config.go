package logging

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-git/go-billy/v5"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/lepovirta/konvahti/internal/file"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Level               string `yaml:"level,omitempty"`
	EnablePrettyLogging bool   `yaml:"enablePrettyLogging,omitempty"`
	OutputStreamName    string `yaml:"outputStream,omitempty"`
	TimestampFormat     string `yaml:"timestampFormat,omitempty"`
	TimestampFieldName  string `yaml:"timestampFieldName,omitempty"`
}

func (c *Config) FromYAML(r io.Reader) error {
	return yaml.NewDecoder(r).Decode(c)
}

func (c *Config) FromYAMLFile(fs billy.Filesystem, filename string) error {
	return file.WithFileReader(
		fs,
		filename,
		func(r io.Reader) error {
			return c.FromYAML(r)
		},
	)
}

func (c *Config) Validate() error {
	_, err := c.parseLevel()
	if err != nil {
		return err
	}
	outStreamName := strings.ToUpper(c.OutputStreamName)
	if outStreamName != "STDOUT" && outStreamName != "STDERR" {
		return fmt.Errorf("invalid output stream %s", c.OutputStreamName)
	}
	return nil
}

func (c *Config) Setup(stdout io.Writer, stderr io.Writer) (zerolog.Logger, error) {
	var err error

	// Globals
	if c.TimestampFieldName != "" {
		zerolog.TimestampFieldName = c.TimestampFieldName
	}
	if c.TimestampFormat != "" {
		zerolog.TimeFieldFormat = c.TimestampFormat
	}
	var level zerolog.Level
	level, err = c.parseLevel()
	if err != nil {
		return zerolog.Nop(), err
	}
	zerolog.SetGlobalLevel(level)

	// Logger customization
	outStreamName := strings.ToUpper(c.OutputStreamName)
	var outStream io.Writer
	switch outStreamName {
	case "STDOUT":
		outStream = stdout
	case "STDERR", "":
		outStream = stderr
	default:
		return zerolog.Nop(), fmt.Errorf("invalid output stream %s", c.OutputStreamName)
	}

	if c.EnablePrettyLogging {
		outStream = zerolog.ConsoleWriter{Out: outStream}
	}

	l := zerolog.New(outStream).With().Timestamp().Logger()

	// Make the logger global
	log.Logger = l

	return l, nil
}

func (c *Config) parseLevel() (zerolog.Level, error) {
	if c.Level == "" {
		return zerolog.InfoLevel, nil
	}
	return zerolog.ParseLevel(c.Level)
}
