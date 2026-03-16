package cmd

import (
	"fmt"
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
	cmd.AddCommand(newFileListCmd(fileSvc))
	cmd.AddCommand(newFileUploadCmd(fileSvc))
	cmd.AddCommand(newFileStoreCmd(fileSvc))
	cmd.AddCommand(newFileDeleteCmd(fileSvc))
	cmd.AddCommand(newFileLocalCopyCmd(fileSvc))
	cmd.AddCommand(newFileRemoteCopyCmd(fileSvc))

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

			return formatter.Format(cmd.OutOrStdout(), fileInfoTable(file))
		},
	}

	cmd.Flags().BoolVar(&includeAppData, "include-appdata", false, "Include application data in response")

	return cmd
}

func newFileListCmd(fileSvc service.FileService) *cobra.Command {
	var (
		ordering       string
		limit          int
		startingPoint  string
		stored         string
		removed        bool
		pageAll        bool
		includeAppData bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List files in project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			listOpts := service.FileListOptions{
				Ordering:       ordering,
				Limit:          limit,
				StartingPoint:  startingPoint,
				Removed:        removed,
				IncludeAppData: includeAppData,
			}

			// Handle tristate --stored flag
			if cmd.Flags().Changed("stored") {
				switch stored {
				case "true":
					b := true
					listOpts.Stored = &b
				case "false":
					b := false
					listOpts.Stored = &b
				default:
					return ExitErrorf(2, "invalid --stored value: %q (must be \"true\" or \"false\")", stored)
				}
			}

			if pageAll {
				return runFileListAll(cmd, svc, listOpts, opts)
			}

			result, err := svc.List(cmd.Context(), listOpts)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), result.Files)
			}

			table := output.NewTableData("UUID", "SIZE", "FILENAME", "STORED", "UPLOADED")
			for _, f := range result.Files {
				table.AddRow(
					f.UUID,
					strconv.FormatInt(f.Size, 10),
					f.Filename,
					strconv.FormatBool(f.IsStored),
					formatTime(f.DatetimeUploaded),
				)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	f := cmd.Flags()
	f.StringVar(&ordering, "ordering", "datetime_uploaded", "Sort order (prefix - for descending)")
	f.IntVar(&limit, "limit", 100, "Number of files per page")
	f.StringVar(&startingPoint, "starting-point", "", "Starting point (RFC3339 datetime)")
	f.StringVar(&stored, "stored", "", "Filter by stored status (true/false)")
	f.BoolVar(&removed, "removed", false, "Include removed files")
	f.BoolVar(&pageAll, "page-all", false, "Stream all pages as NDJSON")
	f.BoolVar(&includeAppData, "include-appdata", false, "Include application data")

	return cmd
}

func runFileListAll(cmd *cobra.Command, svc service.FileService, listOpts service.FileListOptions, opts output.FormatOptions) error {
	if opts.Quiet {
		return svc.Iterate(cmd.Context(), listOpts, func(f service.File) error {
			return nil
		})
	}

	w := cmd.OutOrStdout()
	return svc.Iterate(cmd.Context(), listOpts, func(f service.File) error {
		if opts.JSON {
			return output.NDJSONLine(w, &f, opts.Fields)
		}
		_, err := fmt.Fprintf(w, "%s\t%d\t%s\t%v\t%s\n",
			f.UUID, f.Size, f.Filename, f.IsStored, formatTime(f.DatetimeUploaded))
		return err
	})
}

func fileInfoTable(file *service.File) *output.TableData {
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
	return table
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
