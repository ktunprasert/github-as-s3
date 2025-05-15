package git

import (
	"context"
	"errors"
	"github-as-s3/internal/consts"
	"github-as-s3/internal/util"
	"io"
	"mime/multipart"
	"os"
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

	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	f, err := wt.Filesystem.Create(".ghs3")
	if err != nil {
		return nil, err
	}

	_, err = wt.Add(f.Name())
	if err != nil {
		return nil, err
	}

	_ = f.Close()

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

func (g *Git) Put(ctx context.Context, repo *git.Repository, file *multipart.FileHeader) error {
	if repo == nil {
		return errors.New("repo is nil")
	}

	if file == nil {
		return errors.New("file is nil")
	}

	src, err := file.Open()
	if err != nil {
		return err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	path := wt.Filesystem.Root()
	if path == "" {
		return errors.New("path is empty")
	}

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return errors.New("path does not exist")
		}
		return err
	}

	dst, err := wt.Filesystem.Create(file.Filename)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	_ = src.Close()
	_ = dst.Close()

	_, err = wt.Add(file.Filename)
	if err != nil {
		return err
	}

	_, err = wt.Commit("[GHS3] add file "+file.Filename, &git.CommitOptions{
		Author: g.signature(),
	})
	if err != nil {
		return err
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return err
	}

	err = remote.PushContext(ctx, &git.PushOptions{
		RemoteName: consts.Origin,
		Auth:       g.auth(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (g *Git) Get(ctx context.Context, repoName, relativeFilepath string) error {
	return nil
}

func (g *Git) List(ctx context.Context, repoName string) ([]string, error) {
	return nil, nil
}

func (g *Git) Delete(ctx context.Context, repoName string) error {
	return nil
}

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
