package application

import (
	"github-as-s3/internal/git"
	"github-as-s3/internal/github"
	"os"
)

type applicationOpts = func(*Application)

func WithToken(token string) applicationOpts {
	return func(a *Application) {
		a.Token = token
	}
}

func WithEnvToken() applicationOpts {
	token := os.Getenv("GITHUB_TOKEN")

	return func(a *Application) {
		a.Token = token
	}
}

func WithPort(port string) applicationOpts {
	return func(a *Application) {
		a.Port = port
	}
}

func WithAddress(address string) applicationOpts {
	return func(a *Application) {
		a.Address = address
	}
}

func WithDefaultGithub() applicationOpts {
	return func(a *Application) {
		a.gh = github.NewGitHub(a.Token, a.Owner)
	}
}

func WithDefaultGit() applicationOpts {
	return func(a *Application) {
		a.git = git.NewGit(a.Token, a.Owner)
	}
}

func WithGitHub(gh *github.GitHub) applicationOpts {
	return func(a *Application) {
		a.gh = gh
	}
}

func WithGit(g *git.Git) applicationOpts {
	return func(a *Application) {
		a.git = g
	}
}
