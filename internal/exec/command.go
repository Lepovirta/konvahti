package exec

import (
	"context"
	"os/exec"

	"gitlab.com/lepovirta/konvahti/internal/envvars"
)

type Command struct {
	Args    []string
	Env     envvars.EnvVars
	WorkDir string
}

func (c *Command) ToOSCommand(ctx context.Context) (cmd *exec.Cmd) {
	cmd = exec.CommandContext(ctx, c.Args[0], c.Args[1:]...)
	cmd.Dir = c.WorkDir
	cmd.Env = c.Env
	return
}
