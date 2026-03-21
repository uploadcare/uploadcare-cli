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
		Long: `Create a copy of a file within Uploadcare's own storage.

The new copy gets its own UUID and is independent of the original.
Use --store to immediately mark the copy as permanently stored.

Use --dry-run to verify the source file exists without copying.

JSON fields: uuid, size, filename, mime_type, is_image, is_stored,
is_ready, datetime_uploaded, original_file_url.`,
		Example: `  # Copy a file locally
  uploadcare file local-copy 740e1b8c-1ad8-4324-b7ec-112345678900

  # Copy and store immediately
  uploadcare file local-copy 740e1b8c-1ad8-4324-b7ec-112345678900 --store

  # Dry run: verify the source file exists
  uploadcare file local-copy 740e1b8c-1ad8-4324-b7ec-112345678900 --dry-run --json all`,
		Args: cobra.ExactArgs(1),
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
		Long: `Copy a file from Uploadcare to an external (remote) storage bucket.

The --target flag is required and must be the name of a remote storage
configured in your Uploadcare project settings (e.g. an S3 bucket name).

Use --make-public to set public ACL on the remote copy.
Use --pattern to customize the filename on the remote storage.

Use --dry-run to verify the source file exists without copying.

JSON fields: result (remote URL or identifier), already_exists (bool).`,
		Example: `  # Copy a file to remote storage
  uploadcare file remote-copy 740e1b8c-1ad8-4324-b7ec-112345678900 --target my-s3-bucket

  # Copy and make public
  uploadcare file remote-copy 740e1b8c-1ad8-4324-b7ec-112345678900 \
    --target my-s3-bucket --make-public

  # Copy with custom filename pattern
  uploadcare file remote-copy 740e1b8c-1ad8-4324-b7ec-112345678900 \
    --target my-s3-bucket --pattern "${uuid}/${filename}"

  # Dry run: verify the source file exists
  uploadcare file remote-copy 740e1b8c-1ad8-4324-b7ec-112345678900 \
    --target my-s3-bucket --dry-run --json all`,
		Args: cobra.ExactArgs(1),
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
