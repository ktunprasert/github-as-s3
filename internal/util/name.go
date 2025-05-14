package util

import "fmt"

func RepoName(name string) *string {
	s := fmt.Sprintf("ghs3-%s", name)

	return &s
}
