package watcher

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"gitlab.com/lepovirta/konvahti/internal/env"
	"gitlab.com/lepovirta/konvahti/internal/git"
	"gitlab.com/lepovirta/konvahti/internal/s3"
)

type FileSource interface {
	Refresh(ctx context.Context) ([]string, error)
	GetLogCtx(logger *zerolog.Logger) zerolog.Logger
	GetDirectory() string
}

func fileSourceFromConfig(env *env.Env, config *Config) (FileSource, error) {
	if config.Git != nil {
		var s git.GitSource
		s.Setup(*config.Git)
		return &s, nil
	}
	if config.S3 != nil {
		var s s3.S3Source
		s.Setup(env.Fs, *config.S3)
		return &s, nil
	}
	return nil, fmt.Errorf("no remote source specified for config %s", config.Name)
}
