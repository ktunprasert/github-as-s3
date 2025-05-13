package main

import (
	"context"
	"fmt"
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
