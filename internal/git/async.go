package git

import (
	"context"
	"errors"
	"github-as-s3/internal/consts"
	"github-as-s3/internal/util"
	"io"
	"os"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/rs/zerolog/log"
)

type ChangeRequest struct {
	ctx           context.Context
	ObjectKey     string // e.g., "path/to/my-file.txt"
	Type          string // e.g., "PUT", "DELETE"
	Content       []byte // Content for PUT operations, or nil for DELETE
	CommitMessage string // Pre-formatted commit message for this change
	// You might also need a way to pass any specific S3 headers if your Git mapping requires them.
	ok    chan bool
	tries int
}

type RepoWorker struct {
	sync.RWMutex
	// ctx         context.Context
	isRunning   bool
	changeQueue chan *ChangeRequest
	path        string
	repo        *git.Repository
	ga          *GitAsync
}

func (w *RepoWorker) Start(bucket string) {
	debounceTimer := time.NewTimer(0)
	<-debounceTimer.C // Consume the initial expiration

	logger := log.With().Str("component", "RepoWorker").Str("path", w.path).Str("bucket", bucket).Logger()

	logger.Debug().Msg("RepoWorker Start")
	defer logger.Info().Msg("RepoWorker stopped")

	tries := 0

	// defer func() {
	// 	logger.Info().Msg("Removing from workers map")
	// 	repoWorkers.Lock()
	// 	defer repoWorkers.Unlock()
	// 	delete(repoWorkers.channels, bucket)
	// }()

	wt, err := w.repo.Worktree()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get worktree")
		return
	}

	remote, err := w.repo.Remote(consts.Origin)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get remote")
		return
	}

	for {
		select {
		case change := <-w.changeQueue:
			if err := change.ctx.Err(); err != nil {
				logger.Error().Err(err).Msg("skipping work")
				continue
			}

			if change.tries > 3 {
				logger.Error().Msg("Too many attempts, bumping from queue")
				change.ok <- false
				close(change.ok)
				continue
			}

			slog := logger.With().Str("change", change.Type).Str("key", change.ObjectKey).Logger()
			switch change.Type {
			case consts.Put:
				dst, err := wt.Filesystem.Create(change.ObjectKey)
				if err != nil {
					slog.Error().Err(err).Msg("Failed to create file")
					// try again
					change.tries++
					w.changeQueue <- change
					continue
				}

				_, err = dst.Write(change.Content)
				if err != nil {
					slog.Error().Err(err).Msg("Failed to write file")
					// try again
					change.tries++
					w.changeQueue <- change
					_ = dst.Close()
					continue
				}
				_ = dst.Close()

				_, err = wt.Add(change.ObjectKey)
				if err != nil {
					slog.Error().Err(err).Msg("Failed to add file to git")
					// try again
					change.tries++
					w.changeQueue <- change
					_ = dst.Close()
					continue
				}

			case consts.Delete:
				err := wt.Filesystem.Remove(change.ObjectKey)
				if err != nil {
					slog.Error().Err(err).Msg("Failed to delete file")
					// try again
					change.tries++
					w.changeQueue <- change
					continue
				}

				_, err = wt.Remove(change.ObjectKey)
				if err != nil {
					slog.Error().Err(err).Msg("Failed to remove file from git")
					// try again
					change.tries++
					w.changeQueue <- change
					continue
				}
			}

			hash, err := wt.Commit(change.CommitMessage, &git.CommitOptions{Author: w.ga.signature()})
			if err != nil && !errors.Is(err, git.ErrEmptyCommit) {
				slog.Error().Err(err).Msg("Failed to commit file")
				// try again
				change.tries++
				w.changeQueue <- change
				continue
			}

			slog.Info().Any("hash", hash).Msg("change completed")

			if err == nil {
				debounceTimer.Reset(500 * time.Millisecond) // Adjust debounce interval as needed
			}

			change.ok <- true

		case <-debounceTimer.C:
			if tries > 3 {
				logger.Error().Msg("Too many tries to push, stopping worker")
				return
			}

			if w.ga.skipPush {
				logger.Debug().Msg("Skipping push to remote")
				debounceTimer.Stop()

				continue
			}

			w.Lock()
			err = remote.Push(&git.PushOptions{
				RemoteName: consts.Origin,
				Auth:       w.ga.auth(),
			})
			w.Unlock()

			if err != nil {
				if errors.Is(err, git.NoErrAlreadyUpToDate) {
					logger.Debug().Msg("No changes to push")
					debounceTimer.Stop()
					continue
				}

				logger.Error().Err(err).Msg("failed to push change, trying again in 500ms")
				tries++
				debounceTimer.Reset(500 * time.Millisecond)
				continue
			}

			tries = 0

			debounceTimer.Stop()

			// prevents a race if a new request comes in right after processing
			debounceTimer.Reset(0) // Reset to immediate expiration for the next cycle
			<-debounceTimer.C      // Consume it
		}
	}
}

func (rw *RepoWorker) Stop() {}

var repoWorkers = struct {
	sync.RWMutex
	channels map[string]*RepoWorker
}{channels: make(map[string]*RepoWorker)}

type GitAsync struct {
	*Git
}

func NewGitAsync(git *Git) *GitAsync {
	return &GitAsync{git}
}

func (ga GitAsync) Head(ctx context.Context, name, filepath, version string) (*object.File, *object.Commit, error) {
	logger := log.Ctx(ctx).With().Str("component", "gitasync.Head").Str("repo", name).Str("path", filepath).Logger()
	logger.Debug().Msg("gitasync.Head Start")
	defer logger.Debug().Msg("gitasync.Head End")

	w, err := ga.ensureWorker(ctx, name)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to ensure worker")
		return nil, nil, err
	}

	cmt, err := w.repo.CommitObject(plumbing.Hash([]byte(version)))
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get commit object")
		return nil, nil, err
	}

	file, err := cmt.File(filepath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get file from commit")
		return nil, nil, err
	}

	return file, cmt, nil
}

func (ga GitAsync) Clone(ctx context.Context, bucket string) (*git.Repository, error) {
	logger := log.Ctx(ctx).With().Str("component", "gitasync.Clone").Str("bucket", bucket).Logger()
	logger.Debug().Msg("gitasync.Clone Start")

	worker, err := ga.ensureWorker(ctx, bucket)
	if err != nil {
		return nil, err
	}

	logger.Debug().Msg("gitasync.Clone End")
	return worker.repo, nil
}

func (ga GitAsync) PutRaw(ctx context.Context, repo *git.Repository, bucket, key string, dst io.ReadCloser) error {
	logger := log.Ctx(ctx).With().Str("component", "gitasync.PutRaw").Str("bucket", bucket).Str("key", key).Logger()
	logger.Debug().Msg("gitasync.PutRaw Start")
	defer logger.Debug().Msg("gitasync.PutRaw End")

	if repo == nil {
		logger.Error().Msg("repo is nil")
		return errors.New("repo is nil")
	}

	worker, err := ga.ensureWorker(ctx, bucket)
	if err != nil {
		return err
	}

	fileContent, err := io.ReadAll(dst)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to read file content")
		return err
	}

	change := ChangeRequest{
		ctx:           ctx,
		ObjectKey:     key,
		Content:       fileContent,
		Type:          consts.Put,
		CommitMessage: "Put file " + key,
		ok:            make(chan bool),
	}

	worker.changeQueue <- &change
	if ok := <-change.ok; !ok {
		return errors.New("failed to process change request")
	}

	return nil
}

func (ga GitAsync) Delete(ctx context.Context, repo *git.Repository, bucket, relativeFilepath string) error {
	logger := log.Ctx(ctx).With().Str("component", "gitasync.Delete").Str("relativeFilepath", relativeFilepath).Logger()
	logger.Debug().Msg("gitasync.Delete Start")

	if repo == nil {
		logger.Error().Msg("repo is nil")
		return errors.New("repo is nil")
	}

	worker, err := ga.ensureWorker(ctx, bucket)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to ensure worker")
		return err
	}

	change := ChangeRequest{
		ctx:           ctx,
		ObjectKey:     relativeFilepath,
		Type:          consts.Delete,
		CommitMessage: "Delete file " + relativeFilepath,
		ok:            make(chan bool),
	}

	worker.changeQueue <- &change
	if ok := <-change.ok; !ok {
		return errors.New("failed to process change request")
	}

	logger.Debug().Msg("gitasync.Delete End")

	return nil
}

func (ga GitAsync) ensureWorker(ctx context.Context, bucket string) (*RepoWorker, error) {
	repoWorkers.RLock()
	if w, exists := repoWorkers.channels[bucket]; exists {
		repoWorkers.RUnlock()

		// if !w.isRunning {
		// 	w.Start(bucket)
		// }
		return w, nil
	}
	repoWorkers.RUnlock()

	logger := log.Ctx(ctx).With().Str("component", "gitasync.EnsureWorker").Str("bucket", bucket).Logger()
	logger.Debug().Msg("gitasync.EnsureWorker Start")

	var w *RepoWorker

	path, err := os.MkdirTemp("", "ghs3-"+bucket)
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainClone(path, false, &git.CloneOptions{
		URL:           util.GithubURL(ga.owner, bucket),
		Auth:          ga.auth(),
		RemoteName:    consts.Origin,
		ReferenceName: consts.Master,
		SingleBranch:  true,
	})
	if err != nil {
		if !errors.Is(err, transport.ErrEmptyRemoteRepository) {
			return nil, err
		}

		log.Debug().Str("repo_name", bucket).Msg("Remote repository is empty, calling InitRepo")
		repo, err = ga.InitRepo(ctx, bucket)
		if err != nil {
			return nil, err
		}
	}

	repoWorkers.Lock()
	w = &RepoWorker{
		path:        path,
		changeQueue: make(chan *ChangeRequest, 100), // Buffered channel
		repo:        repo,
		ga:          &ga,
	}
	repoWorkers.channels[bucket] = w
	repoWorkers.Unlock()

	go w.Start(bucket)

	return w, nil
}
