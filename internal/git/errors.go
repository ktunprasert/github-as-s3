package git

import "errors"

var (
	ErrRepoNil       = errors.New("repository is nil")
	ErrPathEmpty     = errors.New("path is empty")
	ErrPathNotExists = errors.New("path does not exist")
	ErrFileNil       = errors.New("file is nil")
)
