package application

import "os"

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
