package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/uploadcare/uploadcare-cli/internal/cmd"
)

// Set via -ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := cmd.NewRootCmd(version, commit, date)
	if err := root.Execute(); err != nil {
		prefix := color.New(color.FgRed, color.Bold).Sprint("Error:")
		fmt.Fprintf(os.Stderr, "%s %s\n", prefix, err)

		var exitErr *cmd.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
