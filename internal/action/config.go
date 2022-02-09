package action

import (
	"context"
	"fmt"
	"time"

	"github.com/gobwas/glob"
	"gitlab.com/lepovirta/konvahti/internal/envvars"
	"gitlab.com/lepovirta/konvahti/internal/file"
)

type Config struct {
	Name              string          `yaml:"name"`
	MatchFiles        []string        `yaml:"matchFiles"`
	EnvVars           envvars.EnvVars `yaml:"env"`
	InheritAllEnvVars bool            `yaml:"inheritAllEnvVars"`
	InheritEnvVars    []string        `yaml:"inheritEnvVars,omitempty"`
	WorkDir           string          `yaml:"workDirectory,omitempty"`
	PreCommand        []string        `yaml:"preCommand,omitempty"`
	Command           []string        `yaml:"command"`
	PostCommand       []string        `yaml:"postCommand,omitempty"`
	Timeout           time.Duration   `yaml:"timeout,omitempty"`
	MaxRetries        int             `yaml:"maxRetries"`
}

func (c *Config) Validate() error {
	if len(c.Command) == 0 {
		return fmt.Errorf("no action command specified for action %s", c.Name)
	}
	return nil
}

func (c *Config) ctxWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.Timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.Timeout)
}

func (c *Config) Matcher() (glob.Glob, error) {
	return file.NewPathGlob(c.MatchFiles)
}
