package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
)

func newVersionCmd(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version",
		Long: `Print the CLI version, build commit, build date, Go version, and OS/arch.

Use --json all for machine-readable output.`,
		Example: `  # Print version info
  uploadcare version

  # Print version as JSON
  uploadcare version --json all`,
		Args: cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			opts := formatOptionsFromCmd(cmd)

			if opts.JSON || opts.JQ != "" {
				formatter := output.New(opts)
				formatter.Format(cmd.OutOrStdout(), map[string]string{
					"version": version,
					"commit":  commit,
					"date":    date,
					"go":      runtime.Version(),
					"os":      runtime.GOOS,
					"arch":    runtime.GOARCH,
				})
				return
			}

			fmt.Fprintf(cmd.OutOrStdout(),
				"uploadcare-cli %s\ncommit: %s\nbuilt:  %s\ngo:     %s\nos/arch: %s/%s\n",
				version, commit, date,
				runtime.Version(),
				runtime.GOOS, runtime.GOARCH,
			)
		},
	}
}
