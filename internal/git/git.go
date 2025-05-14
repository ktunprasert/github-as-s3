package git

import (
	"context"
	"github-as-s3/internal/util"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/memory"
)

type Git struct {
	token   string
	owner   string
	storage storage.Storer
}

func NewGit(token, owner string) *Git {
	return &Git{
		token:   token,
		owner:   owner,
		storage: memory.NewStorage(),
	}
}

// Used when we create a new repo via GitHub but
// it's empty
// func (g *Git) InitRepo(name string) error {
// 	repo, err := git.Init(g.storage, nil)
// 	if err != nil {
// 		return err
// 	}

// 	// create a new remote
// 	remote, err := repo.CreateRemote(&config.RemoteConfig{
// 		Name: "origin",
// 		URLs: []string{
// 			util.GithubURL(g.owner, name),
// 		},
// 		Mirror: false,
// 	})

// 	repo.Storer.Add

// 	// remote.Push
// }

func (g *Git) Clone(ctx context.Context, name string) (*git.Repository, error) {
	path, err := os.MkdirTemp("", "ghs3-"+name)
	if err != nil {
		return nil, err
	}

	repo, err := git.PlainCloneContext(ctx, path, true, &git.CloneOptions{
		URL:           util.GithubURL(g.owner, name),
		Auth:          &http.BasicAuth{Username: "non-empty-string", Password: g.token},
		ReferenceName: "master",
		SingleBranch:  true,
	})
	// repo, err := git.Clone(g.storage, nil, &git.CloneOptions{
	// })

	if err != nil {
		return nil, err
	}

	return repo, nil
}

// func github
