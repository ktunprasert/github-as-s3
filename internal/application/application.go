package application

import (
	"fmt"
	"github-as-s3/internal/git"
	"github-as-s3/internal/github"
	"github-as-s3/internal/server"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Application struct {
	Token   string
	Echo    *echo.Echo
	Port    string
	Address string
	Owner   string

	gh  *github.GitHub
	git *git.Git
}

func newApplication() *Application {
	return &Application{
		Port:    getEnv("GHS3_PORT", "8080"),
		Address: getEnv("GHS3_ADDRESS", "0.0.0.0"),
		Token:   getEnv("GITHUB_TOKEN", ""),
		Owner:   getEnv("GITHUB_OWNER", ""),
	}
}

func NewApplicationWithOpts(opts ...applicationOpts) *Application {
	app := newApplication()
	for _, opt := range opts {
		opt(app)
	}

	return app
}

func (app *Application) Start() error {
	if err := app.Setup(); err != nil {
		return err
	}

	// should be a goroutine?

	return app.Echo.Start(fmt.Sprintf("%s:%s", app.Address, app.Port))
}

func (app *Application) Setup() error {
	// setup routes
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.AddTrailingSlash())
	e.Use(middleware.RequestID())

	app.Echo = e
	server.RegisterRoutes(e, server.NewS3Handler(app.gh, app.git))

	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
