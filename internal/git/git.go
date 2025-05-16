package git

import (
	"context"
	"errors"
	"github-as-s3/internal/consts"
	"github-as-s3/internal/util"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/rs/zerolog/log"
)

type Git struct {
	token    string
	owner    string
	skipPush bool
}

func NewGit(token, owner string) *Git {
	return &Git{
		token: token,
		owner: owner,
	}
}

func (g *Git) SetSkipPush(skipPush bool) {
	g.skipPush = skipPush
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

	err = g.push(ctx, remote, util.GithubURL(g.owner, name))
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

	repo, err := git.PlainCloneContext(ctx, path, false, &git.CloneOptions{
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

func (g *Git) PutRaw(ctx context.Context, repo *git.Repository, key string, src io.ReadCloser) error {
	slog := util.LogCtx(ctx, "git.PutRaw").With().Str("component", "git.PutRaw").Logger()
	slog.Debug().Str("filename", key).Msg("git.PutRaw.Start")

	if repo == nil {
		slog.Error().Msg("repo is nil")
		return errors.New("repo is nil")
	}

	wt, err := repo.Worktree()
	if err != nil {
		slog.Error().Err(err).Msg("failed to get worktree")
		return err
	}

	path := wt.Filesystem.Root()
	if path == "" {
		slog.Error().Msg("worktree path is empty")
		return errors.New("path is empty")
	}
	slog.Debug().Str("path", path).Msg("worktree path resolved")

	dst, err := wt.Filesystem.Create(key)
	if err != nil {
		slog.Error().Err(err).Str("filename", key).Msg("failed to create destination file")
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		slog.Error().Err(err).Str("filename", key).Msg("failed to copy file contents")
		return err
	}
	slog.Debug().Str("filename", key).Msg("file copied successfully")

	_ = src.Close()
	_ = dst.Close()

	_, err = wt.Add(key)
	if err != nil {
		slog.Error().Err(err).Str("filename", key).Msg("failed to add file to git")
		return err
	}

	_, err = wt.Commit("[GHS3] add file "+key, &git.CommitOptions{Author: g.signature()})
	if err != nil {
		if errors.Is(err, git.ErrEmptyCommit) {
			slog.Debug().Msg("empty commit, skipping")
			return nil
		}

		slog.Error().Err(err).Str("filename", key).Msg("failed to commit file")
		return err
	}

	slog.Debug().Str("filename", key).Msg("file committed")

	remote, err := repo.Remote("origin")
	if err != nil {
		slog.Error().Err(err).Msg("failed to get remote 'origin'")
		return err
	}

	err = g.push(ctx, remote, "")
	if err != nil {
		return err
	}

	slog.Debug().Str("filename", key).Msg("git.Put.OK")

	return nil
}

func (g *Git) Put(ctx context.Context, repo *git.Repository, file *multipart.FileHeader) error {
	slog := util.LogCtx(ctx, "git.Put").With().Str("component", "git.Put").Logger()
	slog.Debug().Str("filename", func() string {
		if file != nil {
			return file.Filename
		}
		return ""
	}()).Msg("git.Put.Start")

	if repo == nil {
		slog.Error().Msg("repo is nil")
		return errors.New("repo is nil")
	}

	if file == nil {
		slog.Error().Msg("file is nil")
		return errors.New("file is nil")
	}

	src, err := file.Open()
	if err != nil {
		slog.Error().Err(err).Msg("failed to open file")
		return err
	}
	slog.Debug().Str("filename", file.Filename).Msg("file opened successfully")

	wt, err := repo.Worktree()
	if err != nil {
		slog.Error().Err(err).Msg("failed to get worktree")
		return err
	}

	path := wt.Filesystem.Root()
	if path == "" {
		slog.Error().Msg("worktree path is empty")
		return errors.New("path is empty")
	}
	slog.Debug().Str("path", path).Msg("worktree path resolved")

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			slog.Error().Str("path", path).Msg("worktree path does not exist")
			return errors.New("path does not exist")
		}
		slog.Error().Err(err).Msg("failed to stat worktree path")
		return err
	}

	dst, err := wt.Filesystem.Create(file.Filename)
	if err != nil {
		slog.Error().Err(err).Str("filename", file.Filename).Msg("failed to create destination file")
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		slog.Error().Err(err).Str("filename", file.Filename).Msg("failed to copy file contents")
		return err
	}
	slog.Debug().Str("filename", file.Filename).Msg("file copied successfully")

	_ = src.Close()
	_ = dst.Close()

	_, err = wt.Add(file.Filename)
	if err != nil {
		slog.Error().Err(err).Str("filename", file.Filename).Msg("failed to add file to git")
		return err
	}
	slog.Debug().Str("filename", file.Filename).Msg("file added to git index")

	_, err = wt.Commit("[GHS3] add file "+file.Filename, &git.CommitOptions{
		Author: g.signature(),
	})
	if err != nil {
		// if user uploads the same file with the same content
		// we should let the user do it
		if errors.Is(err, git.ErrEmptyCommit) {
			slog.Debug().Msg("empty commit, skipping")
			return nil
		}

		slog.Error().Err(err).Str("filename", file.Filename).Msg("failed to commit file")
		return err
	}
	slog.Debug().Str("filename", file.Filename).Msg("file committed")

	remote, err := repo.Remote("origin")
	if err != nil {
		slog.Error().Err(err).Msg("failed to get remote 'origin'")
		return err
	}

	err = g.push(ctx, remote, "")
	if err != nil {
		return err
	}

	slog.Debug().Str("filename", file.Filename).Msg("git.Put.OK")
	return nil
}

func (g *Git) Get(ctx context.Context, repo *git.Repository, relativeFilepath string) ([]byte, os.FileInfo, error) {
	if repo == nil {
		return nil, nil, errors.New("repo is nil")
	}

	logger := log.Ctx(ctx).With().Str("component", "git.Get").Str("filename", relativeFilepath).Logger()

	wt, err := repo.Worktree()
	if err != nil {
		return nil, nil, err
	}

	f, err := wt.Filesystem.Open(relativeFilepath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Error().Err(err).Msg("file does not exist")
			return nil, nil, ErrFileNotExists
		}

		logger.Error().Err(err).Msg("failed to open file")
		return nil, nil, err
	}
	defer f.Close()

	fileInfo, error := wt.Filesystem.Stat(relativeFilepath)
	if error != nil {
		logger.Error().Err(error).Msg("failed to get file info")
		return nil, nil, error
	}

	bytes, err := io.ReadAll(f)
	return bytes, fileInfo, err
}

func (g *Git) List(ctx context.Context, repo *git.Repository) (map[string]os.FileInfo, error) {
	slog := util.LogCtx(ctx, "git.List").With().Str("component", "git.List").Logger()
	slog.Debug().Msg("git.List.Start")

	wt, err := repo.Worktree()
	if err != nil {
		slog.Error().Err(err).Msg("failed to get worktree")
		return nil, err
	}

	root := wt.Filesystem.Root()
	files := make(map[string]os.FileInfo)
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Error().Err(err).Str("path", path).Msg("error walking path")
			return err
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			slog.Error().Err(err).Str("path", path).Msg("error getting relative path")
			return err
		}

		switch true {
		case strings.HasPrefix(rel, ".git"):
			slog.Debug().Str("path", path).Str("rel", rel).Msg("skipping dir")
			return filepath.SkipDir
		case strings.HasPrefix(rel, ".ghs3"), rel == ".":
			return nil
		}

		slog.Debug().Str("path", path).Str("rel", rel).Msg("walking path")

		files[rel] = info
		return nil
	})
	if err != nil {
		slog.Error().Err(err).Msg("error walking file tree")
		return nil, err
	}
	slog.Debug().Int("file_count", len(files)).Msg("git.List.OK")
	return files, nil
}

func (g *Git) Delete(ctx context.Context, repo *git.Repository, relativeFilepath string) error {
	slog := util.LogCtx(ctx, "git.Delete").With().Str("component", "git.Delete").Logger()
	slog.Debug().Str("filename", relativeFilepath).Msg("git.Delete.Start")

	wt, err := repo.Worktree()
	if err != nil {
		slog.Error().Err(err).Msg("failed to get worktree")
		return err
	}
	err = wt.Filesystem.Remove(relativeFilepath)
	if err != nil {
		slog.Error().Err(err).Str("filename", relativeFilepath).Msg("failed to remove file from filesystem")
		return err
	}
	slog.Debug().Str("filename", relativeFilepath).Msg("file removed from filesystem")

	_, err = wt.Remove(relativeFilepath)
	if err != nil {
		slog.Error().Err(err).Str("filename", relativeFilepath).Msg("failed to remove file from git index")
		return err
	}
	slog.Debug().Str("filename", relativeFilepath).Msg("file removed from git index")

	_, err = wt.Commit("[GHS3] delete file "+relativeFilepath, &git.CommitOptions{
		Author: g.signature(),
	})
	if err != nil {
		slog.Error().Err(err).Str("filename", relativeFilepath).Msg("failed to commit file deletion")
		return err
	}
	slog.Debug().Str("filename", relativeFilepath).Msg("file deletion committed")

	remote, err := repo.Remote("origin")
	if err != nil {
		slog.Error().Err(err).Msg("failed to get remote 'origin'")
		return err
	}

	err = g.push(ctx, remote, "")
	if err != nil {
		slog.Error().Err(err).Msg("failed to push file deletion to remote")
		return err
	}
	slog.Debug().Str("filename", relativeFilepath).Msg("git.Delete.OK")
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

func (g *Git) push(ctx context.Context, remote *git.Remote, reponame string) error {
	if g.skipPush {
		log.Ctx(ctx).Debug().Msg("skipping push")
		return nil
	}

	if remote == nil {
		return errors.New("remote is nil")
	}

	pushOpts := &git.PushOptions{
		RemoteName: consts.Origin,
		Auth:       g.auth(),
	}

	if len(reponame) > 0 {
		pushOpts.RemoteURL = util.GithubURL(g.owner, reponame)
	}

	err := remote.PushContext(ctx, pushOpts)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Debug().Msg("push OK")
	return nil
}
