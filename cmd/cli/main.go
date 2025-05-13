package main

import (
	"github-as-s3/internal/application"
	"log"
	"net/http"
)

func main() {

	app := application.NewApplicationWithOpts(
		application.WithToken("my_token"),
	)

	if err := app.Start(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
