package file

import (
	// "os"

	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
	"github.com/rs/zerolog/log"
)

const (
	linkSuffix = "_ln"
)

type DirectoryPopulator func(billy.Filesystem)error

func SwapDirectory(
	fs billy.Filesystem,
	targetDirectoryLink string,
	newDirectory string,
	directoryPopulator DirectoryPopulator,
) error {
	newDirectoryLink := newDirectory + linkSuffix

	// Prepare the new directory
	if err := fs.MkdirAll(newDirectory, 0750); err != nil {
		return err
	}
	defer cleanupNewDirectory(fs, targetDirectoryLink, newDirectory)

	// Populate the temporary directory
	tempFs, err := fs.Chroot(newDirectory)
	if err != nil {
		return err
	}
	if err := directoryPopulator(tempFs); err != nil {
		return err
	}

	// Shift links
	if err := fs.Symlink(filepath.Base(newDirectory), newDirectoryLink); err != nil {
		return err
	}
	defer cleanupNewDirectoryLink(fs, newDirectoryLink)

	log.Debug().
		Str("newDirectoryLink", newDirectoryLink).
		Str("targetDirectoryLink", targetDirectoryLink).
		Msg("replacing target directory link with the new link")
	if err := fs.Rename(newDirectoryLink, targetDirectoryLink); err != nil {
		return err
	}
	return nil
}

func cleanupNewDirectory(fs billy.Filesystem, targetDirectoryLink, newDirectory string) {
	dir, err := fs.Readlink(targetDirectoryLink)
	if err != nil {
		return
	}
	if filepath.Base(newDirectory) != dir {
		if err := util.RemoveAll(fs, newDirectory); err != nil {
			log.Error().Err(err).Msg("failed to clean new directory")
		}
	}
}

func cleanupNewDirectoryLink(fs billy.Filesystem, newDirectoryLink string) {
	if _, err := fs.Lstat(newDirectoryLink); err != nil {
		if !os.IsNotExist(err) {
			log.Error().Err(err).Msg("failed to get new directory link info")
		}
		return
	}
	if err := fs.Remove(newDirectoryLink); err != nil {
		log.Error().Err(err).Msg("failed to delete new directory link")
	}
}
