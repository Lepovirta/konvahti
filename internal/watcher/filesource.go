package watcher

import (
	"context"
	"fmt"

	"gitlab.com/lepovirta/konvahti/internal/env"
	"gitlab.com/lepovirta/konvahti/internal/git"
	"gitlab.com/lepovirta/konvahti/internal/s3"
)

type FileSource interface {
	Refresh(ctx context.Context) ([]string, error)
	GetDirectory() string
}

func fileSourceFromConfig(env *env.Env, config *Config) (FileSource, error) {
	if config.Git != nil {
		var s git.GitSource
		if err := s.Setup(*config.Git); err != nil {
			return nil, err
		}
		return &s, nil
	}
	if config.S3 != nil {
		var s s3.S3Source
		if err := s.Setup(env.Fs, *config.S3); err != nil {
			return nil, err
		}
		return &s, nil
	}
	return nil, fmt.Errorf("no remote source specified for config %s", config.Name)
}
