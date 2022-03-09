package start

import (
	"fmt"
	"io"

	"github.com/go-git/go-billy/v5"
	"gitlab.com/lepovirta/konvahti/internal/file"
	"gitlab.com/lepovirta/konvahti/internal/logging"
	"gitlab.com/lepovirta/konvahti/internal/watcher"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Watchers []watcher.Config `yaml:"watchers"`
	Logging  logging.Config   `yaml:"log"`
}

func (c *Config) Validate() error {
	if len(c.Watchers) == 0 {
		return fmt.Errorf("no watcher configurations provided")
	}

	for _, w := range c.Watchers {
		if err := w.Validate(); err != nil {
			return err
		}
	}
	return c.Logging.Validate()
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

func (c *Config) FromEnvVars() error {
	if err := c.Logging.FromEnvVars(); err != nil {
		return err
	}
	for _, watcher := range c.Watchers {
		if err := watcher.FromEnvVars(); err != nil {
			return err
		}
	}
	return nil
}
