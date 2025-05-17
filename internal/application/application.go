package application

import (
	"fmt"
	"github-as-s3/internal/git"
	"github-as-s3/internal/github"
	"github-as-s3/internal/server"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
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
	// e.Use(middleware.Logger())
	e.Use(zerologger())
	e.Use(middleware.Recover())
	e.Use(middleware.AddTrailingSlash())
	e.Use(middleware.RequestID())
	e.Use(middleware.BodyLimit("5M"))

	app.Echo = e
	server.RegisterRoutes(e, server.NewS3Handler(app.gh, app.git), true)

	return nil
}

func zerologger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			start := time.Now()

			loggerBuilder := log.Logger.With()
			clientRequestID := req.Header.Get(echo.HeaderXRequestID)
			if clientRequestID != "" {
				loggerBuilder = loggerBuilder.Str("request_id", clientRequestID)
			}
			requestLogger := loggerBuilder.Logger()

			newCtx := requestLogger.WithContext(req.Context())
			c.SetRequest(req.WithContext(newCtx))

			log.Info().Str("method", req.Method).
				Str("url", req.URL.String()).
				Int("status", res.Status).
				Str("remote_ip", req.RemoteAddr).
				Str("user_agent", req.UserAgent()).
				Msg("request received")

			err := next(c)
			latency := time.Since(start)

			logEvent := requestLogger.Info()
			if err != nil {
				logEvent = requestLogger.Error().Err(err)
			}

			logEvent.Str("method", req.Method).
				Dur("latency", latency).
				Str("request_id", res.Header().Get(echo.HeaderXRequestID)).
				Msg("request completed")

			return err
		}
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}
