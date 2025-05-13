package main

import (
	"context"
	"github-as-s3/internal/application"
	"github-as-s3/internal/github"
	"log"
)

func main() {

	ctx := context.Background()

	app := application.NewApplication()

	gh := github.NewGitHub(app.Token, app.Owner)

	{
		err := gh.CreateRepo(ctx, "test-repo")
		if err != nil {
			log.Fatalf("failed to create repo: %v", err)
		}
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
