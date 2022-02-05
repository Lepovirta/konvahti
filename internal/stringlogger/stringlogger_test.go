package stringlogger

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testLines = []string{
		"somebody set up us the bomb",
		"main screen turn on",
		"all your base are belong to us",
		"",
		"you have no chance to survive make your time",
		"move ZIG",
		"for great justice",
	}
	testTextBytes = []byte(strings.Join(testLines, DefaultSeparator))
)

func TestStringLogger(t *testing.T) {
	// collect lines logged strings to a list
	lines := make([]string, 0, 100)
	collect := func(s string) {
		lines = append(lines, s)
	}
	logger := New(collect)

	// log lines in 10 character batches
	for i := 0; i < len(testTextBytes); i += 10 {
		end := i + 10
		if end > len(testTextBytes) {
			end = len(testTextBytes)
		}
		logger.Write(testTextBytes[i:end])
	}
	logger.Close()

	// collected lines should match the logged lines
	assert.Equal(t, testLines, lines)
}
