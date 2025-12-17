package main

import (
	"context"
	"os"

	"github.com/steipete/blucli/internal/app"
)

var version = "dev"

func main() {
	app.Version = version
	ctx := context.Background()
	os.Exit(app.Run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
