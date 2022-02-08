package action

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/gobwas/glob"
	"github.com/rs/zerolog"
	"gitlab.com/lepovirta/konvahti/internal/envvars"
	"gitlab.com/lepovirta/konvahti/internal/exec"
	"gitlab.com/lepovirta/konvahti/internal/retry"
)

var (
	ErrCommandFail = fmt.Errorf("command exited with non-zero status code")
)

const (
	outcomeEnvKey  = "KONVAHTI_ACTION_STATUS"
	outcomeSuccess = "success"
	outcomeFailed  = "failed"
)

type Runner struct {
	executor   exec.Executor
	retryStrat retry.BackoffGen
	workDir    string
	envVars    envvars.EnvVars
	matcher    glob.Glob
	config     Config
}

func (r *Runner) Setup(
	executor exec.Executor,
	retryStrat retry.BackoffGen,
	defaultWorkDir string,
	envVars envvars.EnvVars,
	config Config,
) (err error) {
	r.matcher, err = config.Matcher()
	if err != nil {
		return
	}
	r.executor = executor
	r.retryStrat = retryStrat
	r.workDir = resolveWorkDir(defaultWorkDir, config.WorkDir)
	r.config = config
	r.inheritEnvVars(envVars)
	return
}

func (r *Runner) Name() string {
	return r.config.Name
}

func (r *Runner) inheritEnvVars(envVars envvars.EnvVars) {
	if r.config.InheritAllEnvVars {
		r.envVars = envVars
	} else {
		for _, envVarName := range r.config.InheritEnvVars {
			if value, ok := envVars.Lookup(envVarName); ok {
				r.envVars = r.envVars.Add(envVarName, value)
			}
		}
	}
	r.envVars = r.envVars.Join(r.config.EnvVars)
}

func (r *Runner) MatchAny(filenames []string) string {
	for _, s := range filenames {
		if r.matcher.Match(s) {
			return s
		}
	}
	return ""
}

func (r *Runner) Run(
	ctx context.Context,
	logger zerolog.Logger,
) (execErr error) {
	logCtx := logger.With().Str("action", r.config.Name)
	logger = logCtx.Logger()
	logger.Debug().Msg("executing command")

	if r.config.PreCommand != nil {
		execErr = retry.Retry(
			ctx,
			r.config.MaxRetries,
			r.retryStrat,
			func(ctx context.Context) error {
				preCommandCtx, preCommandCancel := r.config.ctxWithTimeout(ctx)
				defer preCommandCancel()
				return r.runCommand(
					preCommandCtx,
					logCtx.Str("stage", "preCommand"),
					r.config.PreCommand,
					nil,
				)
			},
		)
	}

	// Instead of returning immediately on error, the if-statement is used here to filter
	// the primary command run. This is done so that the post-command is always run
	// regardless of the outcome.
	if execErr == nil {
		execErr = retry.Retry(
			ctx,
			r.config.MaxRetries,
			r.retryStrat,
			func(ctx context.Context) error {
				commandCtx, commandCancel := r.config.ctxWithTimeout(ctx)
				defer commandCancel()
				return r.runCommand(
					commandCtx,
					logCtx.Str("stage", "command"),
					r.config.Command,
					nil,
				)
			},
		)
	}

	if r.config.PostCommand != nil {
		_ = retry.Retry(
			ctx,
			r.config.MaxRetries,
			r.retryStrat,
			func(ctx context.Context) error {
				outcome := outcomeFailed
				if execErr == nil {
					outcome = outcomeSuccess
				}
				postCommandCtx, postCommandCancel := r.config.ctxWithTimeout(ctx)
				defer postCommandCancel()
				return r.runCommand(
					postCommandCtx,
					logCtx.Str("stage", "postCommand"),
					r.config.PostCommand,
					envvars.FromKeyValue(outcomeEnvKey, outcome),
				)
			},
		)
	}

	if execErr != nil {
		logger := logCtx.Logger()
		logger.Error().Err(execErr).
			Str("event", "action_failed").
			Msg("action execution failed")
	} else {
		logger.Info().
			Str("event", "action_success").
			Msg("action executed successfully")
	}

	return
}

func (r *Runner) runCommand(
	ctx context.Context,
	logCtx zerolog.Context,
	command []string,
	extraEnvVars envvars.EnvVars,
) error {
	logCtx = logCtx.Array("command", strSliceToZerologArr(command))
	logger := logCtx.Logger()

	logger.Debug().Msg("running command")
	returnCode, err := r.executor.Run(
		ctx,
		exec.Command{
			Args:    command,
			Env:     r.envVars.Join(extraEnvVars),
			WorkDir: r.workDir,
		},
		lineLogger(logCtx.Str("event", "stdout")),
		lineLogger(logCtx.Str("event", "stderr")),
	)
	logCtx = logCtx.Int("returnCode", returnCode)
	logger = logCtx.Logger()

	if err != nil {
		logEvent := logger.
			Error().
			Err(err)

		if err == context.DeadlineExceeded {
			logEvent.Str("event", "deadline_exceeded").Msg("deadline exceeded")
		} else {
			logEvent.Str("event", "command_failed").Msg("command execution failed")
		}
		return err
	}

	logger.Debug().
		Str("event", "command_success").
		Msg("command executed successfully")
	return nil
}

func strSliceToZerologArr(ss []string) (arr *zerolog.Array) {
	arr = zerolog.Arr()
	for _, s := range ss {
		arr.Str(s)
	}
	return
}

func lineLogger(logCtx zerolog.Context) func(string) {
	logger := logCtx.Logger()
	logEvent := logger.Info()
	return func(line string) {
		logEvent.Msg(line)
	}
}

func resolveWorkDir(defaultWorkDir string, desiredDir string) string {
	if desiredDir == "" {
		return defaultWorkDir
	}

	if filepath.IsAbs(desiredDir) {
		return desiredDir
	}

	return filepath.Join(defaultWorkDir, desiredDir)
}
