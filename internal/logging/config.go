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
	OutputStream        string `yaml:"outputStream,omitempty"`
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

func (c *Config) Setup(stdout io.Writer, stderr io.Writer) (zerolog.Logger, error) {
	// Globals
	if c.TimestampFieldName != "" {
		zerolog.TimestampFieldName = c.TimestampFieldName
	}
	if c.TimestampFormat != "" {
		zerolog.TimeFieldFormat = c.TimestampFormat
	}
	if c.Level != "" {
		level, err := c.parseLevel()
		if err != nil {
			return zerolog.Nop(), err
		}
		zerolog.SetGlobalLevel(level)
	}

	// Logger customization
	var outStream io.Writer
	if strings.ToUpper(c.OutputStream) == "STDOUT" {
		outStream = stdout
	} else if strings.ToUpper(c.OutputStream) == "STDERR" {
		outStream = stderr
	} else if c.OutputStream == "" {
		outStream = stderr
	} else {
		return zerolog.Nop(), fmt.Errorf("invalid output stream %s", c.OutputStream)
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
	return zerolog.ParseLevel(c.Level)
}
