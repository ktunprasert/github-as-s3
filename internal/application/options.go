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
		a.GH = github.NewGitHub(a.Token, a.Owner)
	}
}

func WithDefaultGit() applicationOpts {
	return func(a *Application) {
		a.Git = git.NewGit(a.Token, a.Owner, a.GitUsername, a.GitEmail)
	}
}

func WithGitHub(gh *github.GitHub) applicationOpts {
	return func(a *Application) {
		a.GH = gh
	}
}

func WithGit(g *git.Git) applicationOpts {
	return func(a *Application) {
		a.Git = g
	}
}
