package file

import (
	"bufio"
	"io"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/rs/zerolog/log"
)

func WithFileReader(
	fs billy.Filesystem,
	filename string,
	f func(io.Reader) error,
) error {
	file, err := fs.Open(filepath.Clean(filename))
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Error().
				Str("filename", filename).
				Err(err).
				Msg("failed to close input file")
		}
	}()
	return f(bufio.NewReader(file))
}
