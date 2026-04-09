package cmd

import (
	"bytes"
	"encoding/json"
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
		Long: `Manage files in the current Uploadcare project.

Subcommands cover the full file lifecycle: upload, list, inspect,
store, delete, and copy. Most subcommands support --json for
structured output and --dry-run for safe previews.

Batch operations (store, delete) accept UUIDs from arguments or
stdin (--from-stdin), and can be piped from "file list --page-all".`,
	}

	cmd.AddCommand(newFileInfoCmd(fileSvc))
	cmd.AddCommand(newFileListCmd(fileSvc))
	cmd.AddCommand(newFileUploadCmd(fileSvc))
	cmd.AddCommand(newFileUploadFromURLCmd(fileSvc))
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
		Long: `Get detailed information about a single file by its UUID.

Returns metadata including size, MIME type, stored/ready status,
upload and storage timestamps, and the original file URL.
Use --include-appdata to also return application-specific data
(e.g. add-on results attached to the file).

JSON fields: uuid, size, filename, mime_type, is_image, is_stored,
is_ready, datetime_uploaded, datetime_stored, datetime_removed,
original_file_url, metadata, appdata (with --include-appdata).`,
		Example: `  # Get file info as a table
  uploadcare file info 740e1b8c-1ad8-4324-b7ec-112345678900

  # Get file info as JSON
  uploadcare file info 740e1b8c-1ad8-4324-b7ec-112345678900 --json all

  # Get only the URL and size
  uploadcare file info 740e1b8c-1ad8-4324-b7ec-112345678900 --json original_file_url,size

  # Include appdata (e.g. virus scan results)
  uploadcare file info 740e1b8c-1ad8-4324-b7ec-112345678900 --include-appdata --json all`,
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

			file, err := svc.Info(cmd.Context(), uuid, includeAppData)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), file)
			}

			if err := formatter.Format(cmd.OutOrStdout(), fileInfoTable(file)); err != nil {
				return err
			}
			if !opts.Quiet && len(file.AppData) > 0 {
				w := cmd.OutOrStdout()
				var buf bytes.Buffer
				if err := json.Indent(&buf, file.AppData, "", "  "); err != nil {
					if _, err := fmt.Fprintf(w, "\nAppData:\n%s\n", string(file.AppData)); err != nil {
						return err
					}
				} else {
					if _, err := fmt.Fprintf(w, "\nAppData:\n%s\n", buf.String()); err != nil {
						return err
					}
				}
			}
			return nil
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
		Long: `List files in the current Uploadcare project with pagination and filtering.

Returns files sorted by --ordering (default: datetime_uploaded descending).
Prefix the ordering field with - for descending (e.g., -datetime_uploaded).

By default returns one page of up to --limit files (default: 100).
Use --page-all to stream ALL files as NDJSON (one JSON object per line).

The --stored flag is a tristate filter:
  --stored true    Only stored files
  --stored false   Only unstored files
  (omitted)        All files regardless of stored status

JSON fields: uuid, size, filename, mime_type, is_image, is_stored,
is_ready, datetime_uploaded, datetime_stored, datetime_removed,
original_file_url, metadata, appdata (with --include-appdata).`,
		Example: `  # List first 100 files as JSON
  uploadcare file list --json all

  # List only stored files, newest first, specific fields
  uploadcare file list --stored true --ordering -datetime_uploaded --json uuid,size,filename

  # Stream ALL file UUIDs (for piping to other commands)
  uploadcare file list --page-all --json uuid

  # Delete all unstored files
  uploadcare file list --page-all --stored false --json uuid \
    | uploadcare file delete --from-stdin

  # Count all files in the project
  uploadcare file list --page-all --json uuid | wc -l`,
		Args: cobra.NoArgs,
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
				return runFileListAll(cmd, svc, listOpts, opts, includeAppData)
			}

			result, err := svc.List(cmd.Context(), listOpts)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), result.Files)
			}

			var table *output.TableData
			if includeAppData {
				table = output.NewTableData("UUID", "SIZE", "FILENAME", "STORED", "UPLOADED", "APPDATA")
			} else {
				table = output.NewTableData("UUID", "SIZE", "FILENAME", "STORED", "UPLOADED")
			}
			for _, f := range result.Files {
				row := []string{
					f.UUID,
					strconv.FormatInt(f.Size, 10),
					f.Filename,
					strconv.FormatBool(f.IsStored),
					formatTime(f.DatetimeUploaded),
				}
				if includeAppData {
					row = append(row, truncateAppData(f.AppData, 50))
				}
				table.AddRow(row...)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	f := cmd.Flags()
	f.StringVar(&ordering, "ordering", "-datetime_uploaded", "Sort order (prefix - for descending)")
	f.IntVar(&limit, "limit", 100, "Number of files per page")
	f.StringVar(&startingPoint, "starting-point", "", "Starting point (RFC3339 datetime)")
	f.StringVar(&stored, "stored", "", "Filter by stored status (true/false)")
	f.BoolVar(&removed, "removed", false, "Include removed files")
	f.BoolVar(&pageAll, "page-all", false, "Stream all pages as NDJSON")
	f.BoolVar(&includeAppData, "include-appdata", false, "Include application data")

	return cmd
}

func runFileListAll(cmd *cobra.Command, svc service.FileService, listOpts service.FileListOptions, opts output.FormatOptions, includeAppData bool) error {
	if opts.Quiet {
		return svc.Iterate(cmd.Context(), listOpts, func(f service.File) error {
			return nil
		})
	}

	w := cmd.OutOrStdout()
	return svc.Iterate(cmd.Context(), listOpts, func(f service.File) error {
		if opts.JSON {
			return output.NDJSONLine(w, &f, opts.Fields, opts.JQ)
		}
		if includeAppData {
			_, err := fmt.Fprintf(w, "%s\t%d\t%s\t%v\t%s\t%s\n",
				f.UUID, f.Size, f.Filename, f.IsStored, formatTime(f.DatetimeUploaded),
				truncateAppData(f.AppData, 50))
			return err
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

func truncateAppData(data json.RawMessage, maxLen int) string {
	if len(data) == 0 {
		return ""
	}
	s := string(data)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
