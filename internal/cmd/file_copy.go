package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
)

func newFileLocalCopyCmd(fileSvc service.FileService) *cobra.Command {
	var (
		store  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "local-copy <uuid>",
		Short: "Copy file within Uploadcare storage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			if err := validate.UUID(uuid); err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			svc := fileSvc
			if svc == nil {
				var err error
				svc, err = fileServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if dryRun {
				file, err := svc.Info(cmd.Context(), uuid, false)
				if err != nil {
					return err
				}
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]any{
						"uuid":     uuid,
						"filename": file.Filename,
						"status":   "would copy locally",
						"store":    store,
					})
				}
				table := &output.TableData{}
				table.AddRow("UUID:", uuid)
				table.AddRow("Filename:", file.Filename)
				table.AddRow("Status:", "would copy locally")
				table.AddRow("Store:", strconv.FormatBool(store))
				return formatter.Format(cmd.OutOrStdout(), table)
			}

			file, err := svc.LocalCopy(cmd.Context(), service.LocalCopyParams{
				UUID:  uuid,
				Store: store,
			})
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), file)
			}

			return formatter.Format(cmd.OutOrStdout(), fileInfoTable(file))
		},
	}

	cmd.Flags().BoolVar(&store, "store", false, "Store the copied file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without executing")

	return cmd
}

func newFileRemoteCopyCmd(fileSvc service.FileService) *cobra.Command {
	var (
		target     string
		makePublic bool
		pattern    string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "remote-copy <uuid>",
		Short: "Copy file to remote storage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			if err := validate.UUID(uuid); err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			if target == "" {
				return ExitErrorf(2, "--target is required")
			}

			svc := fileSvc
			if svc == nil {
				var err error
				svc, err = fileServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if dryRun {
				file, err := svc.Info(cmd.Context(), uuid, false)
				if err != nil {
					return err
				}
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]any{
						"uuid":     uuid,
						"filename": file.Filename,
						"target":   target,
						"status":   "would copy to remote storage",
					})
				}
				table := &output.TableData{}
				table.AddRow("UUID:", uuid)
				table.AddRow("Filename:", file.Filename)
				table.AddRow("Target:", target)
				table.AddRow("Status:", "would copy to remote storage")
				return formatter.Format(cmd.OutOrStdout(), table)
			}

			result, err := svc.RemoteCopy(cmd.Context(), service.RemoteCopyParams{
				UUID:       uuid,
				Target:     target,
				MakePublic: makePublic,
				Pattern:    pattern,
			})
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), result)
			}

			table := &output.TableData{}
			table.AddRow("Result:", result.Result)
			table.AddRow("Already Exists:", fmt.Sprintf("%v", result.AlreadyExists))
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Remote storage name (required)")
	cmd.Flags().BoolVar(&makePublic, "make-public", false, "Make file public on remote storage")
	cmd.Flags().StringVar(&pattern, "pattern", "", "Filename pattern for remote storage")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without executing")

	return cmd
}
