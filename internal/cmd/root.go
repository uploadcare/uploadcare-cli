package cmd

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// NewRootCmd creates the top-level "uploadcare" command with all global flags.
func NewRootCmd(version, commit, date string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "uploadcare",
		Short:         "Uploadcare CLI — manage files, projects, and more",
		Long: `Uploadcare CLI — manage files, projects, and more from the command line.

Authenticate using API keys (flags, env vars, or config file):
  --public-key / UPLOADCARE_PUBLIC_KEY
  --secret-key / UPLOADCARE_SECRET_KEY
Or use a named project from ~/.config/uploadcare/cli.yaml:
  --project <name> / UPLOADCARE_PROJECT

Output modes:
  (default)          Human-readable tables
  --json all         Full JSON objects (one per result)
  --json f1,f2,...   JSON with only the listed fields
  --jq <expr>        Apply a jq expression to JSON output (implies --json)
  --quiet      Suppress all non-error output
  --verbose    Print HTTP request/response details to stderr

Exit codes:
  0  Success
  1  API/runtime error
  2  Usage error (bad arguments, invalid flags)
  3  Auth/config error (missing or invalid credentials)

URL API (on-the-fly CDN image transformations):
  Use "uploadcare url-api" for reference and examples.

Use "uploadcare <command> --help" for details on any command.
Use "uploadcare api-schema" for machine-readable command metadata.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			f := cmd.Root().PersistentFlags()
			verbose, _ := f.GetBool("verbose")
			quiet, _ := f.GetBool("quiet")
			if verbose && quiet {
				return fmt.Errorf("--verbose and --quiet are mutually exclusive")
			}

			// Disable color globally if --no-color flag is set.
			// fatih/color already reads NO_COLOR env on init.
			noColor, _ := f.GetBool("no-color")
			if noColor {
				color.NoColor = true
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
	flags.String("json", "", "Output as JSON: 'all' for every field, or field1,field2 to select")
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
	rootCmd.AddCommand(newAPISchemaCmd(version))
	rootCmd.AddCommand(newFileCmd(nil))
	rootCmd.AddCommand(newMetadataCmd(nil))
	rootCmd.AddCommand(newGroupCmd(nil))
	rootCmd.AddCommand(newWebhookCmd(nil))
	rootCmd.AddCommand(newConvertCmd(nil))
	rootCmd.AddCommand(newAddonCmd(nil))
	rootCmd.AddCommand(newProjectCmd(nil, nil, nil, nil))
	rootCmd.AddCommand(newURLAPICmd())

	return rootCmd
}
