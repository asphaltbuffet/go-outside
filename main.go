// Package main is the entrypoint of the go-outside application.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/asphaltbuffet/go-outside/cmd"
	"github.com/asphaltbuffet/go-outside/internal/cli"
)

func main() {
	err := cmd.Execute(context.Background())
	if err != nil {
		// nope (exit 1) already printed its own stdout line; only surface real
		// failures (exit >= 2) on stderr, since cobra's SilenceErrors is on.
		if code := cli.CodeFor(err); code >= cli.ExitCodeError {
			fmt.Fprintln(os.Stderr, "go-outside:", err)
		}
		os.Exit(cli.CodeFor(err))
	}
}
