package github

import (
	"context"
	"errors"
	"fmt"
	"github-as-s3/internal/util"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v72/github"
	"github.com/rs/zerolog/log"
)

var requiredPermissions = []string{"repo", "delete_repo"}

type GitHub struct {
	client *github.Client
	owner  string
}

func NewGitHub(token, owner string) *GitHub {
	client := github.NewClient(nil).WithAuthToken(token)

	return &GitHub{
		client: client,
		owner:  owner,
	}
}

// nice to have
func (gh *GitHub) CheckPermissions(ctx context.Context) (bool, error) {
	var resp map[string]any

	req, err := http.NewRequest(http.MethodGet, gh.client.BaseURL.String(), nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	ghResp, err := gh.client.Do(ctx, req, &resp)
	if err != nil {
		return false, fmt.Errorf("check permissions: %w", err)
	}

	permissionStr := ghResp.Header.Get("X-OAuth-Scopes")
	if permissionStr == "" {
		return false, fmt.Errorf("no permissions found")
	}

	permissions := strings.Split(permissionStr, ",")

	missing := checkMissingPermissions(permissions)

	if len(missing) == 0 {
		return true, nil
	}

	return false, fmt.Errorf("missing required permissions: %v", missing)
}

func (gh *GitHub) CreateRepo(ctx context.Context, name string, isPrivate bool) error {
	repoName := util.RepoName(name)

	visibility := "public"
	if isPrivate {
		visibility = "private"
	}

	repo, _, err := gh.client.Repositories.Create(ctx, "", &github.Repository{
		Name: repoName,
		Owner: &github.User{
			Name: github.Ptr(gh.owner),
		},
		Visibility: &visibility,
	})

	if err != nil {
		var ghErrResp *github.ErrorResponse
		if ok := errors.As(err, &ghErrResp); ok {
			log.Ctx(ctx).Debug().Any("github_err", ghErrResp).Msg("GitHub API error")
			for _, e := range ghErrResp.Errors {
				if e.Code == "custom" && e.Resource == "Repository" && e.Message == "name already exists on this account" {
					return fmt.Errorf("%w: %s", ErrRepoAlreadyExists, *repoName)
				}
			}
		}

		return err
	}

	// TODO: figure out how to trigger check for this
	// dont want to list/search for it
	// createdAt==nil is NOT the condition
	if repo.CreatedAt == nil {
		tries := 0
		multiplier := 1
		var createdAt *github.Timestamp
		for createdAt == nil {
			if tries > 3 {
				return fmt.Errorf("failed to check whether repo is created")
			}

			repo, _, err := gh.client.Repositories.Get(ctx, gh.owner, *repoName)
			if err != nil {
				return err
			}

			createdAt = repo.CreatedAt

			time.Sleep(time.Duration(multiplier) * time.Second)

			multiplier *= 2
		}

	}
	return nil
}

func (gh *GitHub) DeleteRepo(ctx context.Context, name string) error {
	repoName := util.RepoName(name)

	_, err := gh.client.Repositories.Delete(ctx, gh.owner, *repoName)
	if err != nil {
		var ghErrResp *github.ErrorResponse
		if ok := errors.As(err, &ghErrResp); ok {
			log.Ctx(ctx).Debug().Any("github_err", ghErrResp).Msg("GitHub API error")
			if ghErrResp.DocumentationURL == "https://docs.github.com/rest/repos/repos#delete-a-repository" && ghErrResp.Message == "Not Found" {
				return fmt.Errorf("%w: %s", ErrRepoNotFound, *repoName)
			}
		}

		return err
	}

	return nil
}

func (gh *GitHub) ListRepos(ctx context.Context, page, perPage int, prefix string) ([]*github.Repository, int, bool, error) {
	if page < 1 {
		return nil, 0, false, fmt.Errorf("page must be greater than 0")
	}

	search := fmt.Sprintf("user:%s ghs3-%s in:name", gh.owner, prefix)
	repos, _, err := gh.client.Search.Repositories(ctx, search, &github.SearchOptions{
		Sort:  "updated",
		Order: "desc",
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	})
	if err != nil {
		return nil, 0, false, err
	}

	return repos.Repositories, page + 1, repos.GetIncompleteResults(), nil
}

// TODO: implement HeadBucket
// func (gh *GitHub) HeadRepo(){}

// can be directory or file
func (gh *GitHub) Head(ctx context.Context, name, filepath, version string) (*github.RepositoryContent, []*github.RepositoryContent, *time.Time, error) {
	if name == "" {
		return nil, nil, nil, fmt.Errorf("name cannot be empty")
	}

	logger := log.Ctx(ctx).With().Str("repo", name).Str("path", filepath).Logger()

	getContentOptions := &github.RepositoryContentGetOptions{}

	if version != "" {
		getContentOptions.Ref = version
	}

	fileContent, directoryContent, _, err := gh.client.Repositories.GetContents(ctx, gh.owner, *util.RepoName(name), filepath, getContentOptions)
	if err != nil {
		var ghErrResp *github.ErrorResponse
		if ok := errors.As(err, &ghErrResp); ok {
			logger.Debug().Any("github_err", ghErrResp).Msg("GitHub API error")
		}
		return nil, nil, nil, err
	}

	lastModified := time.Now()

	// TODO: stop faking time
	// d, _, err := gh.client.Git.GetCommit(ctx, gh.owner, *util.RepoName(name), *fileContent.SHA)
	// if err != nil {
	// 	var ghErrResp *github.ErrorResponse
	// 	if ok := errors.As(err, &ghErrResp); ok {
	// 		logger.Debug().Any("github_err", ghErrResp).Msg("GitHub API error")
	// 	}

	// 	// its ok if this doesnt pass we fuzzy the date
	// 	logger.Debug().Msg("could not get commit - using time.Now()")
	// } else {
	// 	lastModified = d.Committer.Date.Time
	// 	logger.Debug().Time("last_modified", d.Committer.Date.Time).Msg("last modified")
	// }
	logger.Debug().Any("filecontent", fileContent).Any("directorycontent", directoryContent).Msg("content")

	return fileContent, directoryContent, &lastModified, nil
}

func (gh GitHub) GetOwner() string {
	return gh.owner
}

func checkMissingPermissions(permissions []string) []string {
	permissionMap := make(map[string]struct{})

	for _, rp := range requiredPermissions {
		permissionMap[rp] = struct{}{}
	}

	for _, p := range permissions {
		p = strings.TrimSpace(p)
		delete(permissionMap, p)
	}

	missing := make([]string, 0)
	for p := range permissionMap {
		missing = append(missing, p)
	}

	return missing
}
