package file

import (
	"github.com/gobwas/glob"
)

const pathGlobSeparator = '/'

type multiGlob struct {
	globs []glob.Glob
}

func (m *multiGlob) Match(s string) bool {
	for _, glob := range m.globs {
		if glob.Match(s) {
			return true
		}
	}
	return false
}

type matchAlways struct {}

func (m *matchAlways) Match(s string) bool {
	return true
}

func NewPathGlob(patterns []string) (glob.Glob, error) {
	if len(patterns) == 0 {
		return &matchAlways{}, nil
	}

	globs := make([]glob.Glob, 0, len(patterns))
	for _, pattern := range patterns {
		glob, err := glob.Compile(pattern, pathGlobSeparator)
		if err != nil {
			return nil, err
		}
		globs = append(globs, glob)
	}
	return &multiGlob{globs}, nil
}
