package util

import "fmt"

func GithubURL(owner, name string) string {
	return fmt.Sprintf("https://github.com/%s/%s", owner, *RepoName(name))
}
