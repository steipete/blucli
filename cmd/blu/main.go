package main

import (
	"context"
	"io"
	"os"

	"github.com/steipete/blucli/internal/app"
)

var version = "dev"
var exit = os.Exit

func runMain(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	app.Version = version
	return app.Run(ctx, args, stdout, stderr)
}

func main() {
	exit(runMain(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}
