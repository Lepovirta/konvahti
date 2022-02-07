package watcher

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gitlab.com/lepovirta/konvahti/internal/action"
	"gitlab.com/lepovirta/konvahti/internal/env"
	"gitlab.com/lepovirta/konvahti/internal/retry"
)

var (
	retryStrat = retry.ExponentialBackoff(time.Millisecond*10, time.Minute*10)
)

type Watcher struct {
	env        *env.Env
	config     Config
	fileSource FileSource
	logger     zerolog.Logger
	runners    []action.Runner
}

func (w *Watcher) Setup(
	env *env.Env,
	config Config,
	logger zerolog.Logger,
) (err error) {
	w.fileSource, err = fileSourceFromConfig(env, &config)
	if err != nil {
		return
	}
	actionDefaultDirectory := w.fileSource.GetDirectory()

	// The list of actions are fetched first before running them,
	// so that all of them can be logged.
	w.runners = make([]action.Runner, len(config.Actions))
	for i, actionConfig := range config.Actions {
		w.runners[i].Setup(
			env.Executor,
			retryStrat,
			actionDefaultDirectory,
			env.EnvVars,
			actionConfig,
		)
		if err != nil {
			return err
		}
	}

	w.env = env
	w.config = config
	w.logger = logger.With().Str("configName", config.Name).Logger()
	return
}

func (s *Watcher) Run(ctx context.Context) error {
	ctx = s.logger.WithContext(ctx)

	if s.config.ShouldRunOnce() {
		return s.runOnce(ctx)
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Errors are not propagated here so that we can
			// try again after the interval has elapsed.
			_ = s.runOnce(ctx)
			time.Sleep(s.config.Interval)
		}
	}
}

func (s *Watcher) runOnce(ctx context.Context) error {
	refreshCtx, refreshCancel := s.config.ctxWithRefreshTimeout(ctx)
	defer refreshCancel()
	log.Debug().Str("stage", "refresh").Msg("refreshing file source")
	changedFiles, err := s.fileSource.Refresh(refreshCtx)
	if err != nil {
		return err
	}

	logger := s.fileSource.GetLogCtx(&s.logger)
	for _, i := range s.findActionsToRun(changedFiles, logger) {
		if err := s.runners[i].Run(ctx, logger); err != nil {
			return err
		}
	}

	return nil
}

func (s *Watcher) findActionsToRun(
	changedFiles []string,
	logger zerolog.Logger,
) (actionsToRun []int) {
	actionsToRun = make([]int, 0, len(s.runners))
	for i, runner := range s.runners {
		if filename := runner.MatchAny(changedFiles); filename != "" {
			logger.Debug().Str("filename", filename).Str("action", runner.Name()).Msg("match found")
			actionsToRun = append(actionsToRun, i)
		}
	}
	return
}