package s3

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeBucketPrefix(t *testing.T) {
	a := assert.New(t)

	a.Equal("/", sanitizeBucketPrefix(""))
	a.Equal("/", sanitizeBucketPrefix("/"))
	a.Equal("/foobar/", sanitizeBucketPrefix("/foobar/"))
	a.Equal("/foobar/", sanitizeBucketPrefix("foobar/"))
	a.Equal("/foobar/", sanitizeBucketPrefix("/foobar"))
	a.Equal("/foo/bar/", sanitizeBucketPrefix("foo/bar"))
	a.Equal("/foo/bar/", sanitizeBucketPrefix("/foo/bar"))
	a.Equal("/foo/bar/", sanitizeBucketPrefix("/foo/bar/"))
	a.Equal("/foo/bar/", sanitizeBucketPrefix("/foo/bar///"))
}
