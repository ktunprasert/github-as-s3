package git

import (
	"context"
	"errors"
	"github-as-s3/internal/consts"
	"github-as-s3/internal/util"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/rs/zerolog/log"
)

type Git struct {
	token   string
	owner   string
	storage storage.Storer
}

func NewGit(token, owner string) *Git {
	log.Debug().Str("owner", owner).Msg("NewGit called")
	return &Git{
		token:   token,
		owner:   owner,
		storage: memory.NewStorage(),
	}
}

// Used when we create a new repo via GitHub but
// it's empty
func (g *Git) InitRepo(ctx context.Context, name string) (*git.Repository, error) {
	slog := util.LogCtx(ctx, "git.InitRepo").With().Str("component", "git.InitRepo").Logger()

	slog.Debug().Str("repo_name", name).Msg("git.InitRepo.Start")
	path, err := os.MkdirTemp("", "ghs3-"+name)
	if err != nil {
		return nil, err
	}

	slog.Debug().Str("path", path).Msg("Temp directory created for InitRepo")
	repo, err := git.PlainInit(path, false)
	if err != nil {
		return nil, err
	}

	remote, err := repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{util.GithubURL(g.owner, name)},
	})
	if err != nil {
		return nil, err
	}

	err = os.WriteFile(filepath.Join(path, ".ghs3"), []byte("Managed by ghs3"), 0644)
	if err != nil {
		return nil, err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	_, err = wt.Add(".ghs3")
	if err != nil {
		return nil, err
	}

	_, err = wt.Commit("batman", &git.CommitOptions{
		Author: g.signature(),
	})
	if err != nil {
		return nil, err
	}

	err = remote.PushContext(ctx, &git.PushOptions{
		RemoteName: consts.Origin,
		RemoteURL:  util.GithubURL(g.owner, name),
		Auth:       g.auth(),
	})
	if err != nil {
		return nil, err
	}

	slog.Debug().Str("repo_name", name).Msg("git.InitRepo.OK")
	return repo, nil
}

func (g *Git) Clone(ctx context.Context, name string) (*git.Repository, error) {
	slog := util.LogCtx(ctx, "git.Clone").With().Str("component", "git.Clone").Logger()
	slog.Debug().Str("repo_name", name).Msg("git.Clone.Start")

	path, err := os.MkdirTemp("", "ghs3-"+name)
	if err != nil {
		return nil, err
	}
	// log.Ctx(ctx).With
	slog.Debug().Str("path", path).Msg("Temp directory created for Clone")

	repo, err := git.PlainCloneContext(ctx, path, true, &git.CloneOptions{
		URL:           util.GithubURL(g.owner, name),
		Auth:          g.auth(),
		ReferenceName: consts.Master,
		SingleBranch:  true,
	})

	if err != nil {
		if errors.Is(err, transport.ErrEmptyRemoteRepository) {
			slog.Debug().Str("repo_name", name).Msg("Remote repository is empty, calling InitRepo")
			repo, err = g.InitRepo(ctx, name)
			if err != nil {
				return nil, err
			}

		} else {
			return nil, err
		}

	}

	slog.Debug().Str("repo_name", name).Msg("git.Clone.OK")
	return repo, nil
}

// func (g *Git)

func (g *Git) auth() transport.AuthMethod {
	return &http.BasicAuth{Username: "non-empty-string", Password: g.token}
}

func (g *Git) signature() *object.Signature {
	return &object.Signature{
		Name:  "GHS3",
		Email: "ktunprasert@outlook.com",
		When:  time.Now(),
	}
}
