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
			for _, e := range ghErrResp.Errors {
				if e.Code == "already_exists" {
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
		return err
	}

	return nil
}

func (gh *GitHub) ListRepos(ctx context.Context, page int) ([]*github.Repository, int, bool, error) {
	if page < 1 {
		return nil, 0, false, fmt.Errorf("page must be greater than 0")
	}

	search := fmt.Sprintf("user:%s ghs3- in:name", gh.owner)
	repos, _, err := gh.client.Search.Repositories(ctx, search, &github.SearchOptions{
		Sort:  "updated",
		Order: "desc",
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: 25,
		},
	})
	if err != nil {
		return nil, 0, false, err
	}

	return repos.Repositories, page + 1, repos.GetIncompleteResults(), nil
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
