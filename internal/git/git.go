package git

import (
	"context"
	"io"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog"
)

type GitSource struct {
	config         Config
	cloneOptions   git.CloneOptions
	pullOptions    git.PullOptions
	repository     *git.Repository
}

func (gs *GitSource) Setup(config Config) error {
	gs.config = config
	if err := config.toCloneOptions(&gs.cloneOptions); err != nil {
		return err
	}
	return cloneOptionsToPullOptions(&gs.pullOptions, &gs.cloneOptions)
}

func cloneOptionsToPullOptions(p *git.PullOptions, o *git.CloneOptions) error {
	p.RemoteName = o.RemoteName
	p.ReferenceName = o.ReferenceName
	p.Depth = o.Depth
	p.Auth = o.Auth
	return p.Validate()
}

func (gs *GitSource) clone(ctx context.Context) (*git.Repository, error) {
	repo, err := git.PlainCloneContext(ctx, gs.config.Directory, false, &gs.cloneOptions)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (gs *GitSource) pull(ctx context.Context) error {
	wt, err := gs.repository.Worktree()
	if err != nil {
		return err
	}
	return wt.PullContext(ctx, &gs.pullOptions)
}

func (gs *GitSource) GetDirectory() string {
	return gs.config.Directory
}

func (gs *GitSource) Refresh(ctx context.Context) ([]string, error) {
	// Repository not set up yet -> initialize it
	if gs.repository == nil {
		return gs.refreshInit(ctx)
	}

	// Repository is already set up, so we can just pull latest changes
	logger := gs.getLogCtx(zerolog.Ctx(ctx))
	logger.Info().Msg("refreshing files from Git")
	return gs.refreshExisting(ctx, logger)
}

func (gs *GitSource) refreshInit(ctx context.Context) ([]string, error) {
	var err error

	// Repository not set up yet, but one might already exist locally,
	// so we can try opening it and pulling the latest changes.
	// This is usually in situations where konvahti is rebooted.
	gs.repository, err = git.PlainOpen(gs.config.Directory)
	if err == nil {
		logger := gs.getLogCtx(zerolog.Ctx(ctx))
		logger.Info().Msg("refreshing files from a Git repo found on file system")
		return gs.refreshExisting(ctx, logger)
	}

	// Repository not found locally, so we need to clone it first.
	// Since we don't have any previous commit to compare changes to,
	// we can just list the files found in the repository.
	if err == git.ErrRepositoryNotExists {
		gs.repository, err = gs.clone(ctx)
	}
	if err != nil {
		return nil, err
	}

	logger := gs.getLogCtx(zerolog.Ctx(ctx))
	logger.Info().Msg("providing list of files cloned from Git")
	return gitListCurrentFiles(gs.repository)
}

func gitListCurrentFiles(repo *git.Repository) (files []string, err error) {
	ref, err := repo.Head()
	if err != nil {
		return
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return
	}

	tree, err := commit.Tree()
	if err != nil {
		return
	}

	walker := object.NewTreeWalker(tree, true, nil)
	defer walker.Close()

	var name string
	for ; err == nil; name, _, err = walker.Next() {
		if name != "" {
			files = append(files, name)
		}
	}

	// EOF is expected, so we don't need to propagate it as an error.
	if err == io.EOF {
		err = nil
	}
	return
}

func (gs *GitSource) refreshExisting(
	ctx context.Context,
	logger zerolog.Logger,
) ([]string, error) {
	prevHead, err := gs.repository.Head()
	if err != nil {
		return nil, err
	}

	logger.Debug().Msg("pulling latest changes from git remote")
	if err := gs.pull(ctx); err != nil {
		if err == git.NoErrAlreadyUpToDate {
			logger.Debug().Msg("no changes found")
			return nil, nil
		}
		return nil, err
	}

	return gitListChangedFiles(gs.repository, prevHead, logger)
}

func gitListChangedFiles(
	repo *git.Repository,
	ref *plumbing.Reference,
	logger zerolog.Logger,
) (files []string, err error) {
	prevCommit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return
	}
	prevTree, err := prevCommit.Tree()
	if err != nil {
		return
	}
	curHead, err := repo.Head()
	if err != nil {
		return
	}
	curCommit, err := repo.CommitObject(curHead.Hash())
	if err != nil {
		return
	}
	curTree, err := curCommit.Tree()
	if err != nil {
		return
	}
	patch, err := curTree.Patch(prevTree)
	if err != nil {
		return
	}

	logger.Debug().
		Str("gitHashNext", curHead.Hash().String()).
		Msg("getting list of changed files")
	for _, fileStat := range patch.Stats() {
		files = append(files, fileStat.Name)
	}
	return
}

func (gs *GitSource) getLogCtx(logger *zerolog.Logger) zerolog.Logger {
	var currentCommitHash string
	if gs.repository != nil {
		if ref, err := gs.repository.Head(); err == nil {
			currentCommitHash = ref.Hash().String()
		}
	}

	return logger.With().
		Str("stage", "refresh").
		Str("gitUrl", gs.cloneOptions.URL).
		Str("gitBranch", gs.config.Branch).
		Str("gitHash", currentCommitHash).
		Logger()
}
