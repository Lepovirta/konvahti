package s3

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog"
	"gitlab.com/lepovirta/konvahti/internal/file"
	"gitlab.com/lepovirta/konvahti/internal/stat"
)

const (
	currentLinkName = "current"
)

type S3Source struct {
	fs               billy.Filesystem
	config           Config
	minioClient      *minio.Client
	lastChanges      stat.Stat
	currentDirectory string
}

func (s *S3Source) Setup(fs billy.Filesystem, config Config) (err error) {
	s.minioClient, err = minio.New(config.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.AccessKeyId, config.SecretAccessKey, ""),
		Secure: !config.DisableTLS,
	})
	if err != nil {
		return
	}

	s.fs = fs
	s.config = config
	s.lastChanges = nil
	s.currentDirectory = fs.Join(config.Directory, currentLinkName)
	return nil
}

func (s *S3Source) GetDirectory() string {
	return s.currentDirectory
}

func (s *S3Source) Refresh(ctx context.Context) ([]string, error) {
	logger := s.getLogCtx(zerolog.Ctx(ctx))
	logger.Info().Msg("refreshing files from S3")

	files, err := s.listFiles(ctx)
	if err != nil {
		return nil, err
	}

	updated, existing := s.lastChanges.Updated(files)

	nextDirectoryName := timestampString()
	nextDirectory := s.fs.Join(s.config.Directory, nextDirectoryName)

	if err := file.SwapDirectory(
		s.fs,
		s.currentDirectory,
		nextDirectory,
		func(fs billy.Filesystem) error {
			for _, objectKey := range updated {
				if err := s.pullObject(ctx, fs, objectKey, logger); err != nil {
					return err
				}
			}
			for _, objectKey := range existing {
				if err := s.copyLocalFile(fs, objectKey, logger); err != nil {
					return err
				}
			}
			return nil
		},
	); err != nil {
		return nil, err
	}

	s.lastChanges = files
	return updated, nil
}

func timestampString() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}

func (s *S3Source) pullObject(
	ctx context.Context,
	fs billy.Filesystem,
	objectKey string,
	logger zerolog.Logger,
) error {
	filename := s.objectKeyToFilename(objectKey)
	file, err := s.prepareTargetFile(fs, filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close file handler")
		}
	}()

	object, err := s.minioClient.GetObject(ctx, s.config.BucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err := object.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close object handler")
		}
	}()

	_, err = io.Copy(file, object)
	return err
}

func (s *S3Source) copyLocalFile(
	fs billy.Filesystem,
	objectKey string,
	logger zerolog.Logger,
) error {
	filename := s.objectKeyToFilename(objectKey)
	file, err := s.prepareTargetFile(fs, filename)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close file handler")
		}
	}()

	sourceFile, err := s.fs.Open(s.fs.Join(s.currentDirectory, filename))
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close file handler")
		}
	}()

	_, err = io.Copy(file, sourceFile)
	return err
}

func (s *S3Source) prepareTargetFile(fs billy.Filesystem, filename string) (billy.File, error) {
	dirname := filepath.Dir(filename)

	if err := fs.MkdirAll(dirname, 0750); err != nil {
		return nil, err
	}
	return fs.Create(filename)
}

func (s *S3Source) listFiles(ctx context.Context) (files stat.Stat, err error) {
	files = make(stat.Stat, 100)

	objectsCh := s.minioClient.ListObjects(ctx, s.config.BucketName, minio.ListObjectsOptions{
		Prefix:    s.config.BucketPrefix,
		Recursive: true,
	})

	for object := range objectsCh {
		if object.Err != nil {
			return nil, object.Err
		}
		files[object.Key] = object.LastModified
	}
	return
}

func (s *S3Source) objectKeyToFilename(key string) string {
	bucketPrefixLen := len(s.config.BucketPrefix)
	if bucketPrefixLen >= len(key) {
		return ""
	}

	filename := key[bucketPrefixLen:]
	if filename[0] == '/' {
		return filename[1:]
	}
	return filename
}

func (s *S3Source) getLogCtx(logger *zerolog.Logger) zerolog.Logger {
	return logger.With().
		Str("stage", "refresh").
		Str("s3Endpoint", s.config.Endpoint).
		Str("s3BucketName", s.config.BucketName).
		Str("s3BucketPrefix", s.config.BucketPrefix).
		Logger()
}
