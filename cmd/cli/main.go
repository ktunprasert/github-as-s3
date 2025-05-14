package main

import (
	"context"
	"fmt"
	"github-as-s3/internal/application"
	"github-as-s3/internal/github"
	"log"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	_ = godotenv.Load()
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	ctx := context.Background()

	app := application.NewApplication()

	gh := github.NewGitHub(app.Token, app.Owner)

	{
		err := gh.CreateRepo(ctx, "test-repo")
		if err != nil {
			log.Err(err).Str("repo", "test-repo").Msg("failed to create repo")
		}
	}

	{
		repos, nextPage, haveMore, err := gh.ListRepos(ctx, 1)
		if err != nil {
			log.Fatalf("failed to list repos: %v", err)
		}

		fmt.Printf("repos: %v, nextPage: %d, haveMore: %v\n", repos, nextPage, haveMore)
	}

	{
		err := gh.DeleteRepo(ctx, "test-repo")
		if err != nil {
			log.Fatalf("failed to delete repo: %v", err)
		}
	}

	// if err := app.Start(); err != http.ErrServerClosed {
	// 	log.Fatal(err)
	// }
}
