// +build integration

package start

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"gitlab.com/lepovirta/konvahti/internal/env"
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
)

var (
	testExpectedResult1 = testExpectedResultBase + testContent1 + "\n"
	testExpectedResult2 = testExpectedResult1 + testExpectedResultBase + testContent2 + "\n"
	testExpectedResult3 = testExpectedResult2 + testExpectedResultBase + testContent3 + "\n"
	sourceGitURL        = ""
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
}

func TestSuccessfulRunWithGit(t *testing.T) {
	a := assert.New(t)
	e := env.RealEnv()
	configPath := e.Fs.Join(testDataDirectory, "integrationtest_git.yaml")
	testResultFilepath := e.Fs.Join(testDataDirectory, testResultFilename)
	sourceGitPath := e.Fs.Join(testDataDirectory, sourceGitDirectoryName)
	config := konvahtiGitConfig()

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
  directory: _testdata/git
refreshTimeout: 4s
actions:
  - name: integrationtest
    env:
      WRITEME: helloworld
    command:
      - sh
      - ../%s
      - %s
`, sourceGitURL, writeshFilename, writeshArg)
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
