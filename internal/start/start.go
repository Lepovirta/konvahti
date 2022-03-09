package start

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"gitlab.com/lepovirta/konvahti/internal/env"
	"gitlab.com/lepovirta/konvahti/internal/watcher"
	"golang.org/x/sync/errgroup"
)

type MainProgram struct {
	env      env.Env
	config   Config
	watchers []watcher.Watcher
	logger   zerolog.Logger
}

func (m *MainProgram) Setup(
	env env.Env,
	configFileName string,
) (err error) {
	m.env = env

	if isSTDIN(configFileName) {
		if err := m.config.FromYAML(env.Stdin); err != nil {
			return fmt.Errorf("failed to read configuration from STDIN: %w", err)
		}
	} else {
		if err := m.config.FromYAMLFile(env.Fs, configFileName); err != nil {
			return fmt.Errorf("failed to read configuration file %s: %w", configFileName, err)
		}
	}

	if err := m.config.FromEnvVars(); err != nil {
		return fmt.Errorf("failed to load config from env: %w", err)
	}

	if err := m.config.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	m.logger, err = m.setupLogging()
	if err != nil {
		return
	}

	m.logger.Debug().Msgf("found %d watcher configs", len(m.config.Watchers))
	m.watchers = make([]watcher.Watcher, len(m.config.Watchers))
	for i, config := range m.config.Watchers {
		err = m.watchers[i].Setup(&env, config, m.logger)
		if err != nil {
			return
		}
	}

	m.env = env
	return nil
}

func (m *MainProgram) setupLogging() (logger zerolog.Logger, err error) {
	logger, err = m.config.Logging.Setup(m.env.Stdout, m.env.Stderr)
	if err != nil {
		err = fmt.Errorf("failed to set up logging: %w", err)
		return
	}
	return
}

func isSTDIN(configFileName string) bool {
	switch configFileName {
	case "-", "STDIN":
		return true
	default:
		return false
	}
}

func (m *MainProgram) Run(ctx context.Context) error {
	if len(m.watchers) == 0 {
		return fmt.Errorf("no watchers configured")
	}
	eg, ctx := errgroup.WithContext(ctx)

	for _, watcher := range m.watchers {
		w := watcher
		eg.Go(func() error {
			return w.Run(ctx)
		})
	}
	return eg.Wait()
}
