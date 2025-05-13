package main

import "github-as-s3/internal/application"

func main() {

	app := application.NewApplicationWithOpts(
		application.WithToken("my_token"),
	)

	if err := app.Start(); err != nil {
		panic(err)
	}
}
