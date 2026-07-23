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
		RunE: func(cmd *cobra.Command, args []string) error {
			return printVersion(cmd, version, commit, date)
		},
	}
}

func printVersion(cmd *cobra.Command, version, commit, date string) error {
	opts := formatOptionsFromCmd(cmd)
	if opts.Quiet {
		return nil
	}

	if opts.JSON || opts.JQ != "" {
		formatter := output.New(opts)
		return formatter.Format(cmd.OutOrStdout(), map[string]string{
			"version":    version,
			"commit":     commit,
			"date":       date,
			"go_version": runtime.Version(),
			"os":         runtime.GOOS,
			"arch":       runtime.GOARCH,
		})
	}

	_, err := fmt.Fprintf(cmd.OutOrStdout(),
		"uploadcare-cli %s\ncommit: %s\nbuilt:  %s\ngo:     %s\nos/arch: %s/%s\n",
		version, commit, date,
		runtime.Version(),
		runtime.GOOS, runtime.GOARCH,
	)
	return err
}
