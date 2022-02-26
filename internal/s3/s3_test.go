package s3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObjectKeyToFilename(t *testing.T) {
	a := assert.New(t)

	a.Equal("file1", objectKeyToFilename("/foobar/", "/foobar/file1"))
	a.Equal("dir/file1", objectKeyToFilename("/foobar/", "/foobar/dir/file1"))
	a.Equal("file1", objectKeyToFilename("/foobar/dir/", "/foobar/dir/file1"))
}
