package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v72/github"
)

var requiredPermissions = []string{"repo", "delete_repo"}

type GitHub struct {
	client *github.Client
}

func NewGitHub(token string) *GitHub {
	client := github.NewClient(nil)
	return &GitHub{
		client: client,
	}
}

func (gh *GitHub) CheckPermissions(ctx context.Context, token string) (bool, error) {
	var resp map[string]any

	req, err := http.NewRequest(http.MethodGet, gh.client.BaseURL.String(), nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	ghResp, err := gh.client.WithAuthToken(token).Do(ctx, req, &resp)
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
