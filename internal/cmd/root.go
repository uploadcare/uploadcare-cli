package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCmd creates the top-level "uploadcare" command with all global flags.
func NewRootCmd(version, commit, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "uploadcare",
		Short:         "Uploadcare CLI — manage files, projects, and more",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Global flags
	flags := rootCmd.PersistentFlags()

	// Authentication
	flags.String("public-key", "", "API public key (env: UPLOADCARE_PUBLIC_KEY)")
	flags.String("secret-key", "", "API secret key (env: UPLOADCARE_SECRET_KEY)")
	flags.String("token", "", "Account-level bearer token (env: UPLOADCARE_TOKEN)")
	flags.String("project", "", "Project name from config (env: UPLOADCARE_PROJECT)")

	// Output control
	flags.String("json", "", "Output as JSON; optional comma-separated field list")
	flags.String("jq", "", "Apply jq expression to JSON output (implies --json)")
	flags.BoolP("quiet", "q", false, "Suppress all non-error output")
	flags.Bool("no-color", false, "Disable colored output (env: NO_COLOR)")

	// Base URL overrides
	flags.String("rest-api-base", "", "Override REST API base URL (env: UPLOADCARE_REST_API_BASE)")
	flags.String("upload-api-base", "", "Override Upload API base URL (env: UPLOADCARE_UPLOAD_API_BASE)")
	flags.String("cdn-base", "", "Override CDN base URL (env: UPLOADCARE_CDN_BASE)")
	flags.String("project-api-base", "", "Override Project API base URL (env: UPLOADCARE_PROJECT_API_BASE)")

	// Subcommands
	rootCmd.AddCommand(newVersionCmd(version, commit, date))

	return rootCmd
}
