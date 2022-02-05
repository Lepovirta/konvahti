package file

import (
	"io"
	"io/ioutil"
	"testing"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/stretchr/testify/assert"
)

func TestWithFileReaderHappy(t *testing.T) {
	filename := "hello.txt"
	filedata := []byte("hello world!")
	dataFromFile := []byte{}

	fs := memfs.New()
	assert.NoError(t, util.WriteFile(fs, filename, filedata, 0666))

	err := WithFileReader(fs, filename, func(r io.Reader) error {
		bs, err := ioutil.ReadAll(r)
		dataFromFile = append(dataFromFile, bs...)
		return err
	})

	assert.NoError(t, err)
	assert.Equal(t, filedata, dataFromFile)
}

func TestWithFileReaderUnhappy(t *testing.T) {
	filename := "hello.txt"
	filedata := []byte("hello world!")
	dataFromFile := []byte{}

	fs := memfs.New()
	assert.NoError(t, util.WriteFile(fs, filename, filedata, 0666))

	err := WithFileReader(fs, "invalid_" + filename, func(r io.Reader) error {
		bs, err := ioutil.ReadAll(r)
		dataFromFile = append(dataFromFile, bs...)
		return err
	})

	assert.Error(t, err)
	assert.Empty(t, dataFromFile)
}
