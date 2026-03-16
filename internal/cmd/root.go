package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// NewRootCmd creates the top-level "uploadcare" command with all global flags.
func NewRootCmd(version, commit, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "uploadcare",
		Short:         "Uploadcare CLI — manage files, projects, and more",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Root().PersistentFlags()
			verbose, _ := f.GetBool("verbose")
			quiet, _ := f.GetBool("quiet")
			if verbose && quiet {
				return fmt.Errorf("--verbose and --quiet are mutually exclusive")
			}

			// Store the root command reference in context so helpers
			// can lazy-init the config loader on first access.
			ctx := context.WithValue(cmd.Context(), configCmdKey{}, cmd)
			cmd.SetContext(ctx)

			return nil
		},
	}

	// Global flags
	flags := rootCmd.PersistentFlags()

	// Authentication
	flags.String("public-key", "", "API public key (env: UPLOADCARE_PUBLIC_KEY)")
	flags.String("secret-key", "", "API secret key (env: UPLOADCARE_SECRET_KEY)")
	flags.String("project-api-token", "", "Account-level bearer token (env: UPLOADCARE_PROJECT_API_TOKEN)")
	flags.String("project", "", "Project name from config (env: UPLOADCARE_PROJECT)")

	// Output control
	flags.String("json", "", "Output as JSON; optional comma-separated field list")
	flags.Lookup("json").NoOptDefVal = "true"
	flags.String("jq", "", "Apply jq expression to JSON output (implies --json)")
	flags.BoolP("quiet", "q", false, "Suppress all non-error output")
	flags.BoolP("verbose", "v", false, "Print HTTP request/response details to stderr (env: UPLOADCARE_VERBOSE)")
	flags.Bool("no-color", false, "Disable colored output (env: NO_COLOR)")

	// Base URL overrides
	flags.String("rest-api-base", "", "Override REST API base URL (env: UPLOADCARE_REST_API_BASE)")
	flags.String("upload-api-base", "", "Override Upload API base URL (env: UPLOADCARE_UPLOAD_API_BASE)")
	flags.String("cdn-base", "", "Override CDN base URL; auto-computed from public key when not set (env: UPLOADCARE_CDN_BASE)")
	flags.String("project-api-base", "", "Override Project API base URL (env: UPLOADCARE_PROJECT_API_BASE)")

	// Subcommands
	rootCmd.AddCommand(newVersionCmd(version, commit, date))
	rootCmd.AddCommand(newFileCmd(nil))

	return rootCmd
}
