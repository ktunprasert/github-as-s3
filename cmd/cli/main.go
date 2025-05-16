package main

import (
	"github-as-s3/internal/application"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setup() {
	_ = godotenv.Load()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

}

func main() {
	setup()

	app := application.NewApplicationWithOpts(
		application.WithDefaultGit(),
		application.WithDefaultGithub(),
	)

	if err := app.Start(); err != http.ErrServerClosed {
		log.Fatal().Err(err)
	}
}
