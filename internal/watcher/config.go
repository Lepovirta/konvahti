package watcher

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/kelseyhightower/envconfig"
	"gitlab.com/lepovirta/konvahti/internal/action"
	"gitlab.com/lepovirta/konvahti/internal/file"
	"gitlab.com/lepovirta/konvahti/internal/git"
	"gitlab.com/lepovirta/konvahti/internal/s3"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Name           string          `yaml:"name"`
	Git            *git.Config     `yaml:"git,omitempty"`
	S3             *s3.Config      `yaml:"s3,omitempty"`
	RefreshTimeout time.Duration   `yaml:"refreshTimeout,omitempty"`
	Interval       time.Duration   `yaml:"interval,omitempty"`
	Actions        []action.Config `yaml:"actions,omitempty"`
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
	if c.Git == nil && c.S3 == nil {
		return fmt.Errorf("no remote source specified")
	}
	if c.Git != nil && c.S3 != nil {
		return fmt.Errorf("too many remote sources specified")
	}

	if c.Git != nil {
		if err := c.Git.Validate(); err != nil {
			return fmt.Errorf("invalid git remote source: %w", err)
		}
	}
	if c.S3 != nil {
		if err := c.S3.Validate(); err != nil {
			return fmt.Errorf("invalid s3 remote source: %w", err)
		}
	}

	if len(c.Actions) == 0 {
		return fmt.Errorf("no actions specified")
	}

	for i, action := range c.Actions {
		if err := action.Validate(); err != nil {
			return fmt.Errorf("invalid action %d - %s: %w", i, action.Name, err)
		}
	}

	return nil
}

func (c *Config) ShouldRunOnce() bool {
	return c.Interval <= 0
}

func (c *Config) ctxWithRefreshTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.RefreshTimeout <= 0 {
		return ctx, func() {
			// Does nothing because there's no timeout to cancel
		}
	}
	return context.WithTimeout(ctx, c.RefreshTimeout)
}

func (c *Config) FromEnvVars() error {
	if c.Name == "" {
		return nil
	}
	return envconfig.Process(fmt.Sprintf("konvahti_%s", c.Name), c)
}
