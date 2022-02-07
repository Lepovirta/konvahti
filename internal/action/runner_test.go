package action

import (
	"context"
	"fmt"
	osexec "os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"gitlab.com/lepovirta/konvahti/internal/envvars"
	"gitlab.com/lepovirta/konvahti/internal/exec"
)

var (
	testBgEnvVars = envvars.EnvVars{
		"PATH_TO_CONFIGURER=/opt/myconfigurer/bin/myconfigurer",
		"SOMETHING_FROM_BG=helloworld",
	}
	testEnvVars = envvars.FromKeyValue(
		"APP_TOKEN", "supersecret",
	)
	testExpectedEnvVars   = testBgEnvVars.Join(testEnvVars)
	testDefaultWorkDir    = "/var/konvahti/data"
	testWorkDir           = "testdir"
	errSimulatedExitError = fmt.Errorf("simulated exit error")
)

func oneMsBackoff(attempt int) time.Duration {
	return 1 * time.Millisecond
}

func TestExecuteSuccessful(t *testing.T) {
	ctx := context.Background()
	executor := newFakeExecutor()
	executor.responses["prepare.sh"] = response{0, nil}
	executor.responses["reload.py"] = response{0, nil}
	executor.responses["report.sh"] = response{0, nil}
	var runner Runner
	err := runner.Setup(
		executor,
		oneMsBackoff,
		testDefaultWorkDir,
		testBgEnvVars,
		Config{
			Name:              "success",
			InheritAllEnvVars: true,
			PreCommand:        []string{"prepare.sh", "1"},
			Command:           []string{"reload.py", "2"},
			PostCommand:       []string{"report.sh", "3"},
			EnvVars:           testEnvVars,
			MaxRetries:        2,
		},
	)
	if !assert.NoError(t, err) {
		return
	}

	if err := runner.Run(ctx, log.Logger); !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, []exec.Command{
		{
			Args:    []string{"prepare.sh", "1"},
			WorkDir: testDefaultWorkDir,
			Env:     testExpectedEnvVars,
		},
		{
			Args:    []string{"reload.py", "2"},
			WorkDir: testDefaultWorkDir,
			Env:     testExpectedEnvVars,
		},
		{
			Args:    []string{"report.sh", "3"},
			WorkDir: testDefaultWorkDir,
			Env: testExpectedEnvVars.Join(
				envvars.FromKeyValue("KONVAHTI_ACTION_STATUS", "success"),
			),
		},
	}, executor.calls)
}

func TestExecuteFailingPreCommand(t *testing.T) {
	ctx := context.Background()
	executor := newFakeExecutor()
	executor.responses["prepare.sh"] = response{1, errSimulatedExitError}
	executor.responses["reload.py"] = response{0, nil}
	executor.responses["report.sh"] = response{0, nil}
	var runner Runner
	err := runner.Setup(
		executor,
		oneMsBackoff,
		testDefaultWorkDir,
		testBgEnvVars,
		Config{
			Name:              "prefail",
			InheritAllEnvVars: true,
			PreCommand:        []string{"prepare.sh", "1"},
			Command:           []string{"reload.py", "2"},
			PostCommand:       []string{"report.sh", "3"},
			WorkDir:           testWorkDir,
			EnvVars:           testEnvVars,
			MaxRetries:        3,
		},
	)
	if !assert.NoError(t, err) {
		return
	}

	err = runner.Run(ctx, log.Logger)
	assert.Error(t, err)
	assert.Equal(t, errSimulatedExitError, err)
	assert.Equal(t, []exec.Command{
		{
			Args:    []string{"prepare.sh", "1"},
			WorkDir: filepath.Join(testDefaultWorkDir, testWorkDir),
			Env:     testExpectedEnvVars,
		},
		{
			Args:    []string{"prepare.sh", "1"},
			WorkDir: filepath.Join(testDefaultWorkDir, testWorkDir),
			Env:     testExpectedEnvVars,
		},
		{
			Args:    []string{"prepare.sh", "1"},
			WorkDir: filepath.Join(testDefaultWorkDir, testWorkDir),
			Env:     testExpectedEnvVars,
		},
		{
			Args:    []string{"prepare.sh", "1"},
			WorkDir: filepath.Join(testDefaultWorkDir, testWorkDir),
			Env:     testExpectedEnvVars,
		},
		{
			Args:    []string{"report.sh", "3"},
			WorkDir: filepath.Join(testDefaultWorkDir, testWorkDir),
			Env: testExpectedEnvVars.Join(
				envvars.FromKeyValue("KONVAHTI_ACTION_STATUS", "failed"),
			),
		},
	}, executor.calls)
}

func TestExecuteFailingCommand(t *testing.T) {
	ctx := context.Background()
	expectedEnvVars := envvars.FromKeyValue(
		"PATH_TO_CONFIGURER", "/opt/myconfigurer/bin/myconfigurer",
	).Join(testEnvVars)
	executor := newFakeExecutor()
	executor.responses["prepare.sh"] = response{0, nil}
	executor.responses["reload.py"] = response{1, errSimulatedExitError}
	executor.responses["report.sh"] = response{0, nil}
	var runner Runner
	err := runner.Setup(
		executor,
		oneMsBackoff,
		testDefaultWorkDir,
		testBgEnvVars,
		Config{
			Name:           "commandfail",
			InheritEnvVars: []string{"PATH_TO_CONFIGURER"},
			PreCommand:     []string{"prepare.sh", "1"},
			Command:        []string{"reload.py", "2"},
			PostCommand:    []string{"report.sh", "3"},
			EnvVars:        testEnvVars,
			MaxRetries:     1,
		},
	)
	if !assert.NoError(t, err) {
		return
	}

	err = runner.Run(ctx, log.Logger)
	assert.Error(t, err)
	assert.Equal(t, errSimulatedExitError, err)
	assert.Equal(t, []exec.Command{
		{
			Args:    []string{"prepare.sh", "1"},
			WorkDir: testDefaultWorkDir,
			Env:     expectedEnvVars,
		},
		{
			Args:    []string{"reload.py", "2"},
			WorkDir: testDefaultWorkDir,
			Env:     expectedEnvVars,
		},
		{
			Args:    []string{"reload.py", "2"},
			WorkDir: testDefaultWorkDir,
			Env:     expectedEnvVars,
		},
		{
			Args:    []string{"report.sh", "3"},
			WorkDir: testDefaultWorkDir,
			Env: expectedEnvVars.Join(
				envvars.FromKeyValue("KONVAHTI_ACTION_STATUS", "failed"),
			),
		},
	}, executor.calls)
}

func TestExecuteFailingPostCommand(t *testing.T) {
	ctx := context.Background()
	workDir := "/" + testWorkDir
	executor := newFakeExecutor()
	executor.responses["prepare.sh"] = response{0, nil}
	executor.responses["reload.py"] = response{0, nil}
	executor.responses["report.sh"] = response{1, errSimulatedExitError}
	var runner Runner
	err := runner.Setup(
		executor,
		oneMsBackoff,
		testDefaultWorkDir,
		testBgEnvVars,
		Config{
			Name:              "postfail",
			InheritAllEnvVars: true,
			PreCommand:        []string{"prepare.sh", "1"},
			Command:           []string{"reload.py", "2"},
			PostCommand:       []string{"report.sh", "3"},
			WorkDir:           workDir,
			EnvVars:           testEnvVars,
			MaxRetries:        2,
		},
	)
	if !assert.NoError(t, err) {
		return
	}

	err = runner.Run(ctx, log.Logger)
	assert.NoError(t, err)
	assert.Equal(t, []exec.Command{
		{
			Args:    []string{"prepare.sh", "1"},
			WorkDir: workDir,
			Env:     testExpectedEnvVars,
		},
		{
			Args:    []string{"reload.py", "2"},
			WorkDir: workDir,
			Env:     testExpectedEnvVars,
		},
		{
			Args:    []string{"report.sh", "3"},
			WorkDir: workDir,
			Env: testExpectedEnvVars.Join(
				envvars.FromKeyValue("KONVAHTI_ACTION_STATUS", "success"),
			),
		},
		{
			Args:    []string{"report.sh", "3"},
			WorkDir: workDir,
			Env: testExpectedEnvVars.Join(
				envvars.FromKeyValue("KONVAHTI_ACTION_STATUS", "success"),
			),
		},
		{
			Args:    []string{"report.sh", "3"},
			WorkDir: workDir,
			Env: testExpectedEnvVars.Join(
				envvars.FromKeyValue("KONVAHTI_ACTION_STATUS", "success"),
			),
		},
	}, executor.calls)
}

type response struct {
	code int
	err  error
}

type fakeExecutor struct {
	responses map[string]response
	calls     []exec.Command
}

func newFakeExecutor() *fakeExecutor {
	return &fakeExecutor{
		responses: make(map[string]response),
	}
}

func (f *fakeExecutor) Run(
	ctx context.Context,
	cmd exec.Command,
	logStdout exec.LogLine,
	logStderr exec.LogLine,
) (int, error) {
	f.calls = append(f.calls, cmd)
	if r, ok := f.responses[cmd.Args[0]]; ok {
		return r.code, r.err
	}
	return -1, osexec.ErrNotFound
}
