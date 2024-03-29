package file

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathGlob(t *testing.T) {
	glob, err := NewPathGlob([]string{
		"content/*.md",
		"assets/*.{css,js}",
		"assets/**/*.{css,js}",
	})
	if !assert.NoError(t, err) {
		return
	}

	testdata := []struct{
		result bool
		value string
	}{
		{
			result: false,
			value: "README.md",
		},
		{
			result: false,
			value: "assets/README.md",
		},
		{
			result: true,
			value: "content/index.md",
		},
		{
			result: true,
			value: "assets/main.css",
		},
		{
			result: true,
			value: "assets/index.js",
		},
		{
			result: true,
			value: "assets/mymodule/mylib/main.js",
		},
		{
			result: true,
			value: "assets/fun,ky.js",
		},
	}

	for i, data := range testdata {
		assert.Equal(
			t,
			data.result,
			glob.Match(data.value),
			`%d: match("%v") != %v`,
			i, data.value, data.result,
		)
	}
}

func TestEmptyPathGlob(t *testing.T) {
	testdata := []string{
		"README.md",
		"assets/README.md",
		"content/index.md",
		"assets/mymodule/mylib/main.js",
		"assets/fun,ky.js",
	}

	glob, err := NewPathGlob([]string{})
	if !assert.NoError(t, err) {
		return
	}

	for i, data := range testdata {
		assert.True(
			t,
			glob.Match(data),
			`%d: empty match("%v") != true`,
			i, data,
		)
	}

	glob, err = NewPathGlob(nil)
	if !assert.NoError(t, err) {
		return
	}

	for i, data := range testdata {
		assert.True(
			t,
			glob.Match(data),
			`%d: nil match("%v") != true`,
			i, data,
		)
	}
}
