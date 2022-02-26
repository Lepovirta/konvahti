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
	latestLinkName = "latest"
)

type S3Source struct {
	fs              billy.Filesystem
	config          Config
	minioClient     *minio.Client
	lastChanges     stat.Stat
	latestDirectory string
}

func (s *S3Source) Setup(fs billy.Filesystem, config Config) (err error) {
	config.sanitizeBucketPrefix()
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
	s.latestDirectory = fs.Join(config.Directory, latestLinkName)
	return nil
}

func (s *S3Source) GetDirectory() string {
	return s.latestDirectory
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
		s.latestDirectory,
		nextDirectory,
		s.createDirectoryPopulator(ctx, updated, existing, logger),
	); err != nil {
		return nil, err
	}

	s.lastChanges = files
	return updated, nil
}

func (s *S3Source) createDirectoryPopulator(
	ctx context.Context,
	updated, existing []string,
	logger zerolog.Logger,
) file.DirectoryPopulator {
	return func(fs billy.Filesystem) error {
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
	}
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
	filename := objectKeyToFilename(s.config.BucketPrefix, objectKey)
	file, err := s.prepareTargetFile(fs, filename)
	if err != nil {
		return err
	}
	defer loggedFileClose(file, logger)

	object, err := s.minioClient.GetObject(ctx, s.config.BucketName, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err := object.Close(); err != nil {
			logger.Error().Err(err).Msg("failed to close object handler")
		}
	}()

	logger.Debug().Str("objectKey", objectKey).Str("filename", filename).Msg("downloading file")
	_, err = io.Copy(file, object)
	return err
}

func (s *S3Source) copyLocalFile(
	fs billy.Filesystem,
	objectKey string,
	logger zerolog.Logger,
) error {
	filename := objectKeyToFilename(s.config.BucketPrefix, objectKey)
	file, err := s.prepareTargetFile(fs, filename)
	if err != nil {
		return err
	}
	defer loggedFileClose(file, logger)

	sourceFile, err := s.fs.Open(s.fs.Join(s.latestDirectory, filename))
	if err != nil {
		return err
	}
	defer loggedFileClose(sourceFile, logger)

	logger.Debug().Str("objectKey", objectKey).Str("filename", filename).Msg("copying file")
	_, err = io.Copy(file, sourceFile)
	return err
}

func loggedFileClose(file billy.File, logger zerolog.Logger) {
	if err := file.Close(); err != nil {
		logger.Error().Err(err).Msg("failed to close file handler")
	}
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

func objectKeyToFilename(prefix, key string) string {
	bucketPrefixLen := len(prefix)
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
