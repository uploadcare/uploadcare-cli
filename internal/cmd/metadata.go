package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
)

func newMetadataCmd(metaSvc service.MetadataService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metadata",
		Short: "Manage file metadata",
	}

	cmd.AddCommand(newMetadataListCmd(metaSvc))
	cmd.AddCommand(newMetadataGetCmd(metaSvc))
	cmd.AddCommand(newMetadataSetCmd(metaSvc))
	cmd.AddCommand(newMetadataDeleteCmd(metaSvc))

	return cmd
}

func newMetadataListCmd(metaSvc service.MetadataService) *cobra.Command {
	return &cobra.Command{
		Use:   "list <file-uuid>",
		Short: "List all metadata for a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			if err := validate.UUID(uuid); err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			svc := metaSvc
			if svc == nil {
				var err error
				svc, err = metadataServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			meta, err := svc.List(cmd.Context(), uuid)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), meta)
			}

			if len(meta) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No metadata found")
				return nil
			}

			table := output.NewTableData("KEY", "VALUE")
			for k, v := range meta {
				table.AddRow(k, v)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}
}

func newMetadataGetCmd(metaSvc service.MetadataService) *cobra.Command {
	return &cobra.Command{
		Use:   "get <file-uuid> <key>",
		Short: "Get a metadata value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid, key := args[0], args[1]
			if err := validate.UUID(uuid); err != nil {
				return &ExitError{Code: 2, Err: err}
			}
			if err := validate.MetadataKey(key); err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			svc := metaSvc
			if svc == nil {
				var err error
				svc, err = metadataServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			value, err := svc.Get(cmd.Context(), uuid, key)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]string{
					"key":   key,
					"value": value,
				})
			}

			fmt.Fprintln(cmd.OutOrStdout(), value)
			return nil
		},
	}
}

func newMetadataSetCmd(metaSvc service.MetadataService) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "set <file-uuid> <key> <value>",
		Short: "Set a metadata key-value pair",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid, key, value := args[0], args[1], args[2]
			if err := validate.UUID(uuid); err != nil {
				return &ExitError{Code: 2, Err: err}
			}
			if err := validate.MetadataKey(key); err != nil {
				return &ExitError{Code: 2, Err: err}
			}
			if err := validate.MetadataValue(value); err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			svc := metaSvc
			if svc == nil {
				var err error
				svc, err = metadataServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if dryRun {
				current, err := svc.Get(cmd.Context(), uuid, key)
				if err != nil {
					current = "(not set)"
				}
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]string{
						"key":           key,
						"current_value": current,
						"new_value":     value,
						"status":        "would set",
					})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Would set %s: %q -> %q\n", key, current, value)
				return nil
			}

			if err := svc.Set(cmd.Context(), uuid, key, value); err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]string{
					"key":   key,
					"value": value,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Set %s = %q\n", key, value)
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without applying")

	return cmd
}

func newMetadataDeleteCmd(metaSvc service.MetadataService) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "delete <file-uuid> <key>",
		Short: "Delete a metadata key",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid, key := args[0], args[1]
			if err := validate.UUID(uuid); err != nil {
				return &ExitError{Code: 2, Err: err}
			}
			if err := validate.MetadataKey(key); err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			svc := metaSvc
			if svc == nil {
				var err error
				svc, err = metadataServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if dryRun {
				value, err := svc.Get(cmd.Context(), uuid, key)
				if err != nil {
					return fmt.Errorf("key %q not found: %w", key, err)
				}
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]string{
						"key":    key,
						"value":  value,
						"status": "would delete",
					})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Would delete %s (current value: %q)\n", key, value)
				return nil
			}

			if err := svc.Delete(cmd.Context(), uuid, key); err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]string{
					"key":    key,
					"status": "deleted",
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Deleted %s\n", key)
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Verify key exists without deleting")

	return cmd
}
