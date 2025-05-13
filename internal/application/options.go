package application

import "os"

type applicationOpts = func(*Application)

func WithToken(token string) applicationOpts {
	return func(a *Application) {
		a.token = token
	}
}

func WithEnvToken() applicationOpts {
	token := os.Getenv("GITHUB_TOKEN")

	return func(a *Application) {
		a.token = token
	}
}

func WithPort(port string) applicationOpts {
	return func(a *Application) {
		a.port = port
	}
}

func WithAddress(address string) applicationOpts {
	return func(a *Application) {
		a.address = address
	}
}
