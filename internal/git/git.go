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
	logger := gs.GetLogCtx(zerolog.Ctx(ctx))

	if gs.repository == nil {
		logger.Debug().Msg("no local repo found. cloning.")
		repo, err := gs.clone(ctx)
		if err != nil {
			return nil, err
		}
		gs.repository = repo
		return gitListCurrentFiles(repo)
	}

	prevHead, err := gs.repository.Head()
	if err != nil {
		return nil, err
	}

	logger.Debug().Msg("fetching latest changes")
	if err := gs.pull(ctx); err != nil {
		if err == git.NoErrAlreadyUpToDate {
			logger.Debug().Msg("no changes found")
			return nil, nil
		}
		return nil, err
	}

	return gitListChangedFiles(gs.repository, prevHead)
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

	for name, _, werr := walker.Next(); werr != nil; {
		if werr != io.EOF {
			return nil, werr
		}
		files = append(files, name)
	}
	return
}

func gitListChangedFiles(repo *git.Repository, ref *plumbing.Reference) (files []string, err error) {
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
	for _, fileStat := range patch.Stats() {
		files = append(files, fileStat.Name)
	}
	return
}

func (gs *GitSource) GetLogCtx(logger *zerolog.Logger) zerolog.Logger {
	var currentCommitHash string
	if gs.repository != nil {
		if ref, err := gs.repository.Head(); err == nil {
			currentCommitHash = ref.Hash().String()
		}
	}

	return logger.With().
		Str("gitUrl", gs.cloneOptions.URL).
		Str("gitBranch", gs.config.Branch).
		Str("gitHash", currentCommitHash).
		Logger()
}
