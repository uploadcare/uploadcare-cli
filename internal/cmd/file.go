package cmd

import (
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
)

func newFileCmd(fileSvc service.FileService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "Manage files",
	}

	cmd.AddCommand(newFileInfoCmd(fileSvc))

	return cmd
}

func newFileInfoCmd(fileSvc service.FileService) *cobra.Command {
	var includeAppData bool

	cmd := &cobra.Command{
		Use:   "info <uuid>",
		Short: "Get file details",
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

			file, err := svc.Info(cmd.Context(), uuid, includeAppData)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), file)
			}

			table := &output.TableData{}
			table.AddRow("UUID:", file.UUID)
			table.AddRow("Filename:", file.Filename)
			table.AddRow("Size:", strconv.FormatInt(file.Size, 10))
			table.AddRow("MIME Type:", file.MimeType)
			table.AddRow("Image:", strconv.FormatBool(file.IsImage))
			table.AddRow("Stored:", strconv.FormatBool(file.IsStored))
			table.AddRow("Ready:", strconv.FormatBool(file.IsReady))
			table.AddRow("Uploaded:", formatTime(file.DatetimeUploaded))
			if file.DatetimeStored != nil {
				table.AddRow("Stored At:", formatTime(*file.DatetimeStored))
			}
			if file.DatetimeRemoved != nil {
				table.AddRow("Removed At:", formatTime(*file.DatetimeRemoved))
			}
			table.AddRow("URL:", file.OriginalFileURL)

			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	cmd.Flags().BoolVar(&includeAppData, "include-appdata", false, "Include application data in response")

	return cmd
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
