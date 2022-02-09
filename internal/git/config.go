package git

import (
	"fmt"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	gitssh "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

type Config struct {
	URL       string      `yaml:"url"`
	Branch    string      `yaml:"branch"`
	Directory string      `yaml:"directory"`
	HTTPAuth  GitHTTPAuth `yaml:"httpAuth,omitempty"`
	SSHAuth   GitSSHAuth  `yaml:"sshAuth,omitempty"`
}

func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("no Git URL specified")
	}
	if c.Branch == "" {
		return fmt.Errorf("no Git branch specified")
	}
	if c.Directory == "" {
		return fmt.Errorf("no local directory specified")
	}
	return nil
}

func (c *Config) toCloneOptions(cloneOptions *git.CloneOptions) error {
	authMethod, err := c.authMethod()
	if err != nil {
		return err
	}

	cloneOptions.URL = c.URL
	cloneOptions.ReferenceName = plumbing.NewBranchReferenceName(c.Branch)
	cloneOptions.SingleBranch = true
	cloneOptions.Depth = 10
	cloneOptions.Tags = git.NoTags
	cloneOptions.Auth = authMethod

	return cloneOptions.Validate()
}

func (c *Config) authMethod() (transport.AuthMethod, error) {
	if c.HTTPAuth.Token != "" {
		return &githttp.TokenAuth{
			Token: c.HTTPAuth.Token,
		}, nil
	}

	if c.HTTPAuth.Username != "" || c.HTTPAuth.Password != "" {
		return &githttp.BasicAuth{
			Username: c.HTTPAuth.Username,
			Password: c.HTTPAuth.Password,
		}, nil
	}

	if c.SSHAuth.KeyPath != "" {
		return gitssh.NewPublicKeysFromFile(c.SSHAuth.Username, c.SSHAuth.KeyPath, c.SSHAuth.KeyPassword)
	}

	return nil, nil
}

type GitHTTPAuth struct {
	Token    string `yaml:"token"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type GitSSHAuth struct {
	Username    string `yaml:"username"`
	KeyPath     string `yaml:"keyPath"`
	KeyPassword string `yaml:"keyPassword"`
}
