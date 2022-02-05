package watcher

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-git/go-billy/v5"
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
	if c.Name == "" {
		return fmt.Errorf("no name specified")
	}

	if err := onlyOneNonNil("source", c.Git, c.S3); err != nil {
		return err
	}

	if c.Git != nil {
		if err := c.Git.Validate(); err != nil {
			return err
		}
	}
	if c.S3 != nil {
		if err := c.S3.Validate(); err != nil {
			return err
		}
	}

	if len(c.Actions) == 0 {
		return fmt.Errorf("no actions specified")
	}

	for i, action := range c.Actions {
		if err := action.Validate(); err != nil {
			return fmt.Errorf("invalid action %d: %w", i, err)
		}
	}

	return nil
}

func onlyOneNonNil(what string, items ...interface{}) error {
	nonNilCount := 0
	for i := range items {
		if items[i] != nil {
			nonNilCount += 1
		}
		if nonNilCount > 1 {
			return fmt.Errorf("more than one %s specified", what)
		}
	}
	if nonNilCount == 0 {
		return fmt.Errorf("no %ss specified", what)
	}
	return nil
}

func (c *Config) ShouldRunOnce() bool {
	return c.Interval <= 0
}

func (c *Config) ctxWithRefreshTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.RefreshTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.RefreshTimeout)
}
