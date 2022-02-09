package file

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

var (
	testFiles = []struct {
		name string
		data []byte
	}{
		{
			name: "README",
			data: []byte("Welcome!"),
		},
		{
			name: "main.py",
			data: []byte("print('hi')"),
		},
		{
			name: "texts/hello.txt",
			data: []byte("Hello world!"),
		},
		{
			name: "texts/goodbye.txt",
			data: []byte("Goodbye world!"),
		},
	}
	errExpected = fmt.Errorf("expected error")
)

const (
	testDataDir = "_testdata"
	latestDirectory = "/var/lib/stuff/latest"
	firstDirectory = "/var/lib/stuff/first"
	secondDirectory = "/var/lib/stuff/second"
)

func TestSwapDirectoryOK(t *testing.T) {
	fs := osfs.New(testDataDir)
	defer func() {
		if err := util.RemoveAll(fs, "."); err != nil {
			log.Error().Err(err).Msg("failed to erase test files")
		}
	}()

	// Prepare environment
	if err := fs.MkdirAll(firstDirectory, 0750); !assert.NoError(t, err) {
		return
	}
	if err := fs.Symlink(filepath.Base(firstDirectory), latestDirectory); !assert.NoError(t, err) {
		return
	}

	// Create first directory files
	for _, testFile := range testFiles[:3] {
		if err := fs.MkdirAll(fs.Join(firstDirectory, filepath.Dir(testFile.name)), 0750); !assert.NoError(t, err) {
			return
		}
		if err := util.WriteFile(fs, fs.Join(firstDirectory, testFile.name), testFile.data, 0660); !assert.NoError(t, err) {
			return
		}
	}

	// Swap second to latest
	writeFiles := func(tempFs billy.Filesystem) error {
		for _, testFile := range testFiles[1:] {
			if err := tempFs.MkdirAll(filepath.Dir(testFile.name), 0750); !assert.NoError(t, err) {
				return err
			}
			if err := util.WriteFile(tempFs, testFile.name, testFile.data, 0660); !assert.NoError(t, err) {
				return err
			}
		}
		return nil
	}
	if err := SwapDirectory(fs, latestDirectory, secondDirectory, writeFiles); !assert.NoError(t, err) {
		return
	}

	// Check that the latest directory contains only second directory contents
	for _, testFile := range testFiles[1:] {
		filename := fs.Join(latestDirectory, testFile.name)
		data, err := util.ReadFile(fs, filename)
		if !assert.NoError(t, err) {
			t.Logf("file: %s", filename)
		}
		assert.Equal(t, testFile.data, data)
	}
	if _, err := fs.Stat(fs.Join(latestDirectory, testFiles[0].name)); assert.Error(t, err) {
		assert.True(t, os.IsNotExist(err))
	}
}

func TestSwapDirectoryFail(t *testing.T) {
	fs := osfs.New(testDataDir)
	defer func() {
		if err := util.RemoveAll(fs, "."); err != nil {
			log.Error().Err(err).Msg("failed to erase test files")
		}
	}()

	// Prepare environment
	if err := fs.MkdirAll(firstDirectory, 0750); !assert.NoError(t, err) {
		return
	}
	if err := fs.Symlink(filepath.Base(firstDirectory), latestDirectory); !assert.NoError(t, err) {
		return
	}

	// Create first directory files
	for _, testFile := range testFiles[:3] {
		if err := fs.MkdirAll(fs.Join(firstDirectory, filepath.Dir(testFile.name)), 0750); !assert.NoError(t, err) {
			return
		}
		if err := util.WriteFile(fs, fs.Join(firstDirectory, testFile.name), testFile.data, 0660); !assert.NoError(t, err) {
			return
		}
	}

	// Swap second to latest
	writeFiles := func(tempFs billy.Filesystem) error {
		for i, testFile := range testFiles[1:] {
			if i > 1 {
				// Intentionally produce a syntetic error to simulate file population errors
				return errExpected
			}
			if err := tempFs.MkdirAll(filepath.Dir(testFile.name), 0750); !assert.NoError(t, err) {
				return err
			}
			if err := util.WriteFile(tempFs, testFile.name, testFile.data, 0660); !assert.NoError(t, err) {
				return err
			}
		}
		return nil
	}
	if err := SwapDirectory(fs, latestDirectory, secondDirectory, writeFiles); assert.Error(t, err) {
		assert.Equal(t, errExpected, err)
	}

	// Check that the latest directory contains only first directory contents
	for _, testFile := range testFiles[:3] {
		filename := fs.Join(latestDirectory, testFile.name)
		data, err := util.ReadFile(fs, filename)
		if !assert.NoError(t, err) {
			t.Logf("file: %s", filename)
		}
		assert.Equal(t, testFile.data, data)
	}
	if _, err := fs.Stat(fs.Join(latestDirectory, testFiles[3].name)); assert.Error(t, err) {
		assert.True(t, os.IsNotExist(err))
	}

	// Ensure that the second directory (and its link) no longer exists
	if _, err := fs.Stat(secondDirectory); assert.Error(t, err) {
		assert.True(t, os.IsNotExist(err))
	}
	if _, err := fs.Lstat(secondDirectory + linkSuffix); assert.Error(t, err) {
		assert.True(t, os.IsNotExist(err))
	}
}
