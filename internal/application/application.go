package application

import (
	"fmt"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Application struct {
	token   string
	echo    *echo.Echo
	port    string
	address string
}

func NewApplication() *Application {
	return &Application{
		port:    "8888",
		address: "0.0.0.0",
	}
}

func NewApplicationWithOpts(opts ...applicationOpts) *Application {
	app := NewApplication()
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

	return app.echo.Start(fmt.Sprintf("%s:%s", app.address, app.port))
}

func (app *Application) Setup() error {
	// setup routes
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.AddTrailingSlash())
	e.Use(middleware.RequestID())

	app.echo = e

	return nil
}
