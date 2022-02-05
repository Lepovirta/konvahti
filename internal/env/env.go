package env

import (
	"io"
	"os"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"gitlab.com/lepovirta/konvahti/internal/envvars"
	"gitlab.com/lepovirta/konvahti/internal/exec"
)

type Env struct {
	Fs       billy.Filesystem
	EnvVars  envvars.EnvVars
	Executor exec.Executor
	Stdin    io.Reader
	Stdout   io.Writer
	Stderr   io.Writer
}

func RealEnv() Env {
	return Env{
		Fs:       osfs.New(""),
		Executor: exec.NewExecutor(),
		EnvVars:  os.Environ(),
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
	}
}
