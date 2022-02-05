package exec

import (
	"context"
	"os/exec"

	"gitlab.com/lepovirta/konvahti/internal/stringlogger"
)

type LogLine func(string)

type Executor interface {
	Run(
		ctx context.Context,
		command Command,
		logStdout LogLine,
		logStderr LogLine,
	) (int, error)
}

type osExecutor struct{}

func (oe *osExecutor) Run(
	ctx context.Context,
	command Command,
	logStdout LogLine,
	logStderr LogLine,
) (int, error) {
	cmd := command.ToOSCommand(ctx)
	cmd.Stdout = stringlogger.New(logStdout)
	cmd.Stderr = stringlogger.New(logStderr)
	err := cmd.Run()
	if err != nil {
		if eErr, ok := err.(*exec.ExitError); ok {
			return eErr.ExitCode(), eErr
		}
		return -1, err
	}
	return 0, nil
}

func NewExecutor() Executor {
	return &osExecutor{}
}
