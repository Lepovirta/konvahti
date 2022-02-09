package start

import (
	"bufio"
	"context"
	"fmt"

	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"gitlab.com/lepovirta/konvahti/internal/env"
	"gitlab.com/lepovirta/konvahti/internal/logging"
	"gitlab.com/lepovirta/konvahti/internal/watcher"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

const (
	appName = "konvahti"
)

type MainProgram struct {
	env      env.Env
	watchers []watcher.Watcher
	logger   zerolog.Logger
}

func (m *MainProgram) Setup(
	env env.Env,
	configFileNames []string,
	logConfigFileName string,
) (err error) {
	m.logger, err = setupLogging(&env, logConfigFileName)
	if err != nil {
		return
	}

	var configs []watcher.Config
	err = readConfigs(&env, &configs, configFileNames)
	if err != nil {
		return
	}

	m.logger.Debug().Msgf("found %d configs", len(configs))
	m.watchers = make([]watcher.Watcher, len(configs))
	for i, config := range configs {
		err = m.watchers[i].Setup(&env, config, m.logger)
		if err != nil {
			return
		}
	}

	m.env = env
	return nil
}

func setupLogging(env *env.Env, logConfigFileName string) (logger zerolog.Logger, err error) {
	var loggingConfig logging.Config
	if logConfigFileName != "" {
		if err = loggingConfig.FromYAMLFile(env.Fs, logConfigFileName); err != nil {
			err = fmt.Errorf("failed to parse log config from file %s: %w", logConfigFileName, err)
			return
		}
	}
	if err = envconfig.Process(appName+"_log", &loggingConfig); err != nil {
		err = fmt.Errorf("failed to read log config from env vars: %w", err)
		return
	}
	logger, err = loggingConfig.Setup(env.Stdout, env.Stderr)
	if err != nil {
		err = fmt.Errorf("failed to set up logging: %w", err)
		return
	}
	return
}

func readConfigs(env *env.Env, configs *[]watcher.Config, configFileNames []string) error {
	if isSTDIN(configFileNames) {
		if err := yaml.NewDecoder(bufio.NewReader(env.Stdin)).Decode(configs); err != nil {
			return fmt.Errorf("failed to read configs from STDIN: %w", err)
		}
	} else {
		*configs = make([]watcher.Config, len(configFileNames))
		for i, configFileName := range configFileNames {
			if err := (*configs)[i].FromYAMLFile(env.Fs, configFileName); err != nil {
				return fmt.Errorf("failed to read config from %s: %w", configFileName, err)
			}
			if (*configs)[i].Name == "" {
				(*configs)[i].Name = fmt.Sprintf("%d", i)
			}
			if err := (*configs)[i].Validate(); err != nil {
				return fmt.Errorf("invalid configuration %d - %s: %w", i, (*configs)[i].Name, err)
			}
		}
	}
	return nil
}

func isSTDIN(configFileNames []string) bool {
	if len(configFileNames) != 1 {
		return false
	}
	switch configFileNames[0] {
	case "-", "STDIN":
		return true
	}
	return false
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
