// +build integration

package start

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"gitlab.com/lepovirta/konvahti/internal/env"
	"gitlab.com/lepovirta/konvahti/internal/retry"
)

const (
	testDataDirectory = "_testdata"
	writeshFilename   = "write.sh"
	writesh           = `#!/bin/sh
echo "running write.sh!"
echo "$WRITEME" >> ../result.txt
echo "$@" >> ../result.txt
cat content.txt >> ../result.txt
echo "" >> ../result.txt`
	writeshArg             = "fromargs"
	testContentFilename    = "content.txt"
	testContent1           = "fromcontentfile1"
	testContent2           = "fromcontentfile2"
	testContent3           = "fromcontentfile3"
	testExpectedResultBase = "helloworld\nfromargs\n"
	testResultFilename     = "result.txt"
	sourceGitDirectoryName = "sourcegit"
	s3BucketName           = "bukit"
	s3BucketPrefix         = "priifiks"
)

var (
	testExpectedResult1 = testExpectedResultBase + testContent1 + "\n"
	testExpectedResult2 = testExpectedResult1 + testExpectedResultBase + testContent2 + "\n"
	testExpectedResult3 = testExpectedResult2 + testExpectedResultBase + testContent3 + "\n"
	sourceGitURL        = ""
	s3Endpoint          = "127.0.0.1:9000"
	s3AccessKeyId       = "AKIAIOSFODNN7EXAMPLE"
	s3SecretAccessKey   = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	s3ObjectPath        = fmt.Sprintf("%s/%s", s3BucketPrefix, testContentFilename)
)

var (
	minioClient *minio.Client = nil
)

func TestMain(m *testing.M) {
	os.Exit(testMain(m))
}

func testMain(m *testing.M) int {
	defer teardownIntegrationTest()
	if err := setupIntegrationTest(); err != nil {
		log.Err(err).Msg("integration test setup failed")
		return 1
	}
	return m.Run()
}

func setupIntegrationTest() error {
	var err error
	ctx := context.Background()

	// Enable debug logging for integration tests
	if err := os.Setenv("KONVAHTI_LOG_LEVEL", "debug"); err != nil {
		return err
	}

	// Set up Minio client for S3 access
	if address, ok := os.LookupEnv("MINIO_ENDPOINT"); ok {
		s3Endpoint = address
	}
	if username, ok := os.LookupEnv("MINIO_USERNAME"); ok {
		s3AccessKeyId = username
	}
	if password, ok := os.LookupEnv("MINIO_PASSWORD"); ok {
		s3SecretAccessKey = password
	}
	minioClient, err = minio.New(s3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3AccessKeyId, s3SecretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		return err
	}

	// Prepare bucket
	if err := retry.Retry(
		ctx,
		1,
		retry.ExponentialBackoff(time.Millisecond*10, time.Minute*10),
		func(ctx context.Context) error {
			var rerr minio.ErrorResponse
			err := minioClient.MakeBucket(ctx, s3BucketName, minio.MakeBucketOptions{})
			if errors.As(err, &rerr) {
				if rerr.Code == "BucketAlreadyOwnedByYou" {
					return nil
				}
			}
			return err
		},
	); err != nil {
		return err
	}

	// Set up Git source path
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}
	sourceGitURL = fmt.Sprintf(
		"file://%s/%s/%s",
		workingDir, testDataDirectory, sourceGitDirectoryName,
	)

	// Prepare the test data directory
	if err := os.Mkdir(testDataDirectory, 0750); err != nil {
		return err
	}

	// Prepare the test executable that konvahti runs
	if err := os.WriteFile(
		filepath.Join(testDataDirectory, writeshFilename),
		[]byte(writesh),
		0700,
	); err != nil {
		return err
	}

	return nil
}

func teardownIntegrationTest() {
	if err := os.RemoveAll(testDataDirectory); err != nil {
		log.Error().Err(err).Msg("failed to delete test data directory")
	}

	if err := minioClient.RemoveBucketWithOptions(
		context.Background(),
		s3BucketName,
		minio.RemoveBucketOptions{
			ForceDelete: true,
		},
	); err != nil {
		log.Error().Err(err).Msg("failed to delete s3 test bucket")
	}
}

func TestSuccessfulRunWithGit(t *testing.T) {
	a := assert.New(t)
	e := env.RealEnv()
	configPath := e.Fs.Join(testDataDirectory, "integrationtest_git.yaml")
	testResultFilepath := e.Fs.Join(testDataDirectory, testResultFilename)
	sourceGitPath := e.Fs.Join(testDataDirectory, sourceGitDirectoryName)
	config := konvahtiGitConfig()

	// Clear the result file after the end of the test
	defer func() {
		err := e.Fs.Remove(testResultFilepath)
		if err != nil {
			log.Error().Err(err).Msg("failed to delete result")
		}
	}()

	// Write the configuration file
	if err := util.WriteFile(e.Fs, configPath, []byte(config), 0660); !a.NoError(err) {
		return
	}
	defer func() {
		err := e.Fs.Remove(configPath)
		if err != nil {
			log.Error().Err(err).Msg("failed to delete config")
		}
	}()

	// Set up the main program
	var prg MainProgram
	if err := prg.Setup(e, []string{configPath}, ""); !a.NoError(err) {
		return
	}

	// Prepare the Git directory where konvahti pulls configs from
	testContentFilepath := e.Fs.Join(sourceGitPath, testContentFilename)
	repo, err := git.PlainInit(sourceGitPath, false)
	if !a.NoError(err) {
		return
	}
	defer func() {
		err := util.RemoveAll(e.Fs, sourceGitPath)
		if err != nil {
			log.Error().Err(err).Msg("failed to delete source git repo")
		}
	}()

	// Write the first change to the source git repo
	if err := util.WriteFile(
		e.Fs,
		testContentFilepath,
		[]byte(testContent1),
		0640,
	); !a.NoError(err) {
		return
	}
	if err := commitFile(repo, testContentFilename, "content1"); !a.NoError(err) {
		return
	}

	// Run konvahti
	if err := prg.Run(context.Background()); !a.NoError(err) {
		return
	}

	// Verify that konvahti picked up the changes and ran the appropriate commands
	result, err := util.ReadFile(e.Fs, testResultFilepath)
	if !a.NoError(err) {
		return
	}
	a.Equal(testExpectedResult1, string(result))

	// Change the content in source git repo
	if err := util.WriteFile(
		e.Fs,
		testContentFilepath,
		[]byte(testContent2),
		0640,
	); !a.NoError(err) {
		return
	}
	if err := commitFile(repo, testContentFilename, "content2"); !a.NoError(err) {
		return
	}

	// Run konvahti again
	if err := prg.Run(context.Background()); !a.NoError(err) {
		return
	}

	// Verify that konvahti picked up the changes and ran the appropriate commands
	result, err = util.ReadFile(e.Fs, testResultFilepath)
	if !a.NoError(err) {
		return
	}
	a.Equal(testExpectedResult2, string(result))

	// Change the content in source git repo
	if err := util.WriteFile(
		e.Fs,
		testContentFilepath,
		[]byte(testContent3),
		0640,
	); !a.NoError(err) {
		return
	}
	if err := commitFile(repo, testContentFilename, "content3"); !a.NoError(err) {
		return
	}

	// Re-create konvahti and run it again
	if err := prg.Setup(e, []string{configPath}, ""); !a.NoError(err) {
		return
	}
	if err := prg.Run(context.Background()); !a.NoError(err) {
		return
	}

	// Verify that konvahti picked up the changes and ran the appropriate commands
	result, err = util.ReadFile(e.Fs, testResultFilepath)
	if !a.NoError(err) {
		return
	}
	a.Equal(testExpectedResult3, string(result))
}

func konvahtiGitConfig() string {
	return fmt.Sprintf(`
name: integrationtest_git
git:
  url: %s
  branch: master
  directory: %s/git
refreshTimeout: 4s
actions:
  - name: integrationtest
    env:
      WRITEME: helloworld
    command:
      - sh
      - ../%s
      - %s
`, sourceGitURL, testDataDirectory, writeshFilename, writeshArg)
}

func commitFile(repo *git.Repository, filename string, commitMsg string) error {
	wtree, err := repo.Worktree()
	if err != nil {
		return err
	}
	if _, err := wtree.Add(filename); err != nil {
		return err
	}
	_, err = wtree.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Konvahti",
			Email: "konvahti@example.org",
			When:  time.Now(),
		},
	})
	return err
}

func TestSuccessfulRunWithS3(t *testing.T) {
	ctx := context.Background()
	a := assert.New(t)
	e := env.RealEnv()
	configPath := e.Fs.Join(testDataDirectory, "integrationtest_s3.yaml")
	testResultFilepath := e.Fs.Join(testDataDirectory, testResultFilename)
	config := konvahtiS3Config()

	// Clear the result file after the end of the test
	defer func() {
		err := e.Fs.Remove(testResultFilepath)
		if err != nil {
			log.Error().Err(err).Msg("failed to delete result")
		}
	}()

	// Write the configuration file
	if err := util.WriteFile(e.Fs, configPath, []byte(config), 0660); !a.NoError(err) {
		return
	}
	defer func() {
		err := e.Fs.Remove(configPath)
		if err != nil {
			log.Error().Err(err).Msg("failed to delete config")
		}
	}()

	// Set up the main program
	var prg MainProgram
	if err := prg.Setup(e, []string{configPath}, ""); !a.NoError(err) {
		return
	}

	// Write the first change to the s3 bucket
	if err := s3Upload(ctx, testContent1); !a.NoError(err) {
		return
	}

	// Run konvahti
	if err := prg.Run(context.Background()); !a.NoError(err) {
		return
	}

	// Verify that konvahti picked up the changes and ran the appropriate commands
	result, err := util.ReadFile(e.Fs, testResultFilepath)
	if !a.NoError(err) {
		return
	}
	a.Equal(testExpectedResult1, string(result))

	// Change the content in s3 bucket
	if err := s3Upload(ctx, testContent2); !a.NoError(err) {
		return
	}

	// Run konvahti again
	if err := prg.Run(context.Background()); !a.NoError(err) {
		return
	}

	// Verify that konvahti picked up the changes and ran the appropriate commands
	result, err = util.ReadFile(e.Fs, testResultFilepath)
	if !a.NoError(err) {
		return
	}
	a.Equal(testExpectedResult2, string(result))

	// Change the content in s3 bucket
	if err := s3Upload(ctx, testContent3); !a.NoError(err) {
		return
	}

	// Re-create konvahti and run it again
	if err := prg.Setup(e, []string{configPath}, ""); !a.NoError(err) {
		return
	}
	if err := prg.Run(context.Background()); !a.NoError(err) {
		return
	}

	// Verify that konvahti picked up the changes and ran the appropriate commands
	result, err = util.ReadFile(e.Fs, testResultFilepath)
	if !a.NoError(err) {
		return
	}
	a.Equal(testExpectedResult3, string(result))
}

func konvahtiS3Config() string {
	return fmt.Sprintf(`
name: integrationtest_s3
s3:
  endpoint: %s
  accessKeyId: %s
  secretAccessKey: %s
  bucketName: %s
  bucketPrefix: %s
  directory: %s
  disableTls: true
refreshTimeout: 4s
actions:
  - name: integrationtest
    env:
      WRITEME: helloworld
    command:
      - sh
      - ../%s
      - %s
`,
		s3Endpoint, s3AccessKeyId, s3SecretAccessKey,
		s3BucketName, s3BucketPrefix, testDataDirectory,
		writeshFilename, writeshArg,
	)
}

func s3Upload(ctx context.Context, content string) error {
	_, err := minioClient.PutObject(
		ctx,
		s3BucketName,
		s3ObjectPath,
		strings.NewReader(content),
		int64(len(content)),
		minio.PutObjectOptions{},
	)
	return err
}
