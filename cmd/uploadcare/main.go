package main

import (
	"errors"
	"fmt"
	"os"

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
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)

		var exitErr *cmd.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
