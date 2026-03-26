package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
)

func newProjectSecretCmd(secretSvc service.SecretService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secret",
		Short: "Manage API secrets",
		Long: `Manage API secrets for a project.

Secrets are used for HMAC-signed REST API authentication.
Requires --project-api-token.

Subcommands: list, create, delete.`,
	}

	cmd.AddCommand(newSecretListCmd(secretSvc))
	cmd.AddCommand(newSecretCreateCmd(secretSvc))
	cmd.AddCommand(newSecretDeleteCmd(secretSvc))

	return cmd
}

func newSecretListCmd(secretSvc service.SecretService) *cobra.Command {
	return &cobra.Command{
		Use:   "list <project>",
		Short: "List API secrets",
		Long: `List API secrets for a project (by name or pub_key).

Returns secret IDs and 4-character hints (not full keys).
Requires --project-api-token.

JSON fields: id, hint, last_used_at.`,
		Example: `  # List secrets by pub_key
  uploadcare project secret list abc123

  # List secrets by project name
  uploadcare project secret list "My App"

  # As JSON
  uploadcare project secret list abc123 --json all`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pubKey := resolveProjectPubKey(cmd, args[0])

			svc := secretSvc
			if svc == nil {
				var err error
				svc, err = secretServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			secrets, err := svc.List(cmd.Context(), pubKey)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), secrets)
			}

			table := output.NewTableData("ID", "HINT", "LAST_USED_AT")
			for _, s := range secrets {
				lastUsed := ""
				if s.LastUsedAt != nil {
					lastUsed = *s.LastUsedAt
				}
				table.AddRow(s.ID, s.Hint, lastUsed)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}
}

func newSecretCreateCmd(secretSvc service.SecretService) *cobra.Command {
	return &cobra.Command{
		Use:   "create <project>",
		Short: "Create a new API secret",
		Long: `Create a new API secret for a project (by name or pub_key).

Returns the full secret key. This is the only time the full secret
is visible — store it securely.

Requires --project-api-token.

JSON fields: id, secret.`,
		Example: `  # Create a secret
  uploadcare project secret create abc123

  # Create and capture in a variable
  uploadcare project secret create "My App" --json all --jq '.secret'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pubKey := resolveProjectPubKey(cmd, args[0])

			svc := secretSvc
			if svc == nil {
				var err error
				svc, err = secretServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			result, err := svc.Create(cmd.Context(), pubKey)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), result)
			}

			table := &output.TableData{}
			table.AddRow("ID:", result.ID)
			table.AddRow("Secret:", result.Secret)
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}
}

func newSecretDeleteCmd(secretSvc service.SecretService) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "delete <project> <secret-id>",
		Short: "Delete an API secret",
		Long: `Delete an API secret by project (name or pub_key) and secret ID.

Fails if it is the last secret and signed uploads are enabled.
Requires --project-api-token.`,
		Example: `  # Delete a secret
  uploadcare project secret delete abc123 sec_abc

  # Dry run
  uploadcare project secret delete "My App" sec_abc --dry-run`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pubKey := resolveProjectPubKey(cmd, args[0])
			secretID := args[1]

			svc := secretSvc
			if svc == nil {
				var err error
				svc, err = secretServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]string{
						"pub_key":   pubKey,
						"secret_id": secretID,
						"status":    "would delete",
					})
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would delete secret %s from project %s\n", secretID, pubKey)
				return nil
			}

			if err := svc.Delete(cmd.Context(), pubKey, secretID); err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]string{
					"pub_key":   pubKey,
					"secret_id": secretID,
					"status":    "deleted",
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted secret %s from project %s\n", secretID, pubKey)
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Verify without deleting")

	return cmd
}
