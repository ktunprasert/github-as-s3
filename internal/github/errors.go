package github

import "errors"

var (
	ErrRepoAlreadyExists = errors.New("repository already exists")
	ErrRepoNotFound      = errors.New("repository not found")
)
