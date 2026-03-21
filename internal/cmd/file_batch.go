package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

func newFileStoreCmd(fileSvc service.FileService) *cobra.Command {
	var (
		fromStdin bool
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "store <uuid>...",
		Short: "Store files",
		Long: `Mark one or more files as permanently stored.

Stored files are not auto-deleted and persist until explicitly removed.
Accepts UUIDs as positional arguments, from stdin (--from-stdin), or both.
Duplicates are automatically removed.

Files are processed in batches of 100. The result includes both
successfully stored files and any problems (e.g. not found).

Use --dry-run to look up each file and show what would happen
without actually storing anything.

JSON fields: files (array of file objects), problems (map of uuid to error).

Exit codes:
  0  All files stored successfully
  1  Some files had problems (partial success)
  2  Usage error (bad UUID format, no input)`,
		Example: `  # Store specific files
  uploadcare file store 740e1b8c-1ad8-4324-b7ec-112345678900

  # Store multiple files
  uploadcare file store UUID1 UUID2 UUID3

  # Pipe from file list: store all unstored files
  uploadcare file list --page-all --stored false --json uuid \
    | uploadcare file store --from-stdin

  # Dry run: check which files would be stored
  uploadcare file store UUID1 UUID2 --dry-run --json all`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchCommand(cmd, args, fileSvc, fromStdin, dryRun, "store")
		},
	}

	cmd.Flags().BoolVar(&fromStdin, "from-stdin", false, "Read UUIDs from stdin")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without executing")

	return cmd
}

func newFileDeleteCmd(fileSvc service.FileService) *cobra.Command {
	var (
		fromStdin bool
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "delete <uuid>...",
		Short: "Delete files",
		Long: `Delete one or more files from the current Uploadcare project.

Accepts UUIDs as positional arguments, from stdin (--from-stdin), or both.
Duplicates are automatically removed.

Files are processed in batches of 100. The result includes both
successfully deleted files and any problems (e.g. not found).

Use --dry-run to look up each file and show what would happen
without actually deleting anything.

JSON fields: files (array of file objects), problems (map of uuid to error).

Exit codes:
  0  All files deleted successfully
  1  Some files had problems (partial success)
  2  Usage error (bad UUID format, no input)`,
		Example: `  # Delete a specific file
  uploadcare file delete 740e1b8c-1ad8-4324-b7ec-112345678900

  # Delete multiple files
  uploadcare file delete UUID1 UUID2 UUID3

  # Pipe from file list: delete all unstored files
  uploadcare file list --page-all --stored false --json uuid \
    | uploadcare file delete --from-stdin

  # Dry run: check which files would be deleted
  uploadcare file delete UUID1 UUID2 --dry-run --json all`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBatchCommand(cmd, args, fileSvc, fromStdin, dryRun, "delete")
		},
	}

	cmd.Flags().BoolVar(&fromStdin, "from-stdin", false, "Read UUIDs from stdin")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without executing")

	return cmd
}

func runBatchCommand(
	cmd *cobra.Command,
	args []string,
	fileSvc service.FileService,
	fromStdin, dryRun bool,
	operation string,
) error {
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

	uuids, err := collectUUIDs(args, fromStdin)
	if err != nil {
		return err
	}

	if len(uuids) == 0 {
		return ExitErrorf(2, "no UUIDs provided")
	}

	// Validate all UUIDs
	for _, uuid := range uuids {
		if err := validate.UUID(uuid); err != nil {
			return &ExitError{Code: 2, Err: err}
		}
	}

	if dryRun {
		return runBatchDryRun(cmd, svc, uuids, operation, opts, formatter)
	}

	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())
	totalBatches := (len(uuids) + 99) / 100
	verbose.Infof("batch %s: %d files in %d batch(es)", operation, len(uuids), totalBatches)

	// Execute in chunks of 100
	merged := &service.BatchResult{
		Problems: make(map[string]string),
	}

	batchNum := 0
	for i := 0; i < len(uuids); i += 100 {
		end := i + 100
		if end > len(uuids) {
			end = len(uuids)
		}
		chunk := uuids[i:end]
		batchNum++
		verbose.Infof("batch %d/%d: %d files", batchNum, totalBatches, len(chunk))

		var result *service.BatchResult
		switch operation {
		case "store":
			result, err = svc.Store(cmd.Context(), chunk)
		case "delete":
			result, err = svc.Delete(cmd.Context(), chunk)
		}
		if err != nil {
			return err
		}

		merged.Files = append(merged.Files, result.Files...)
		for k, v := range result.Problems {
			merged.Problems[k] = v
		}
	}

	if opts.JSON {
		if err := formatter.Format(cmd.OutOrStdout(), merged); err != nil {
			return err
		}
	} else {
		table := output.NewTableData("UUID", "FILENAME", "STATUS")
		for _, f := range merged.Files {
			table.AddRow(f.UUID, f.Filename, "ok")
		}
		for uuid, problem := range merged.Problems {
			table.AddRow(uuid, "", problem)
		}
		if err := formatter.Format(cmd.OutOrStdout(), table); err != nil {
			return err
		}
	}

	if len(merged.Problems) > 0 {
		warn := color.New(color.FgYellow)
		for uuid, problem := range merged.Problems {
			fmt.Fprintf(cmd.ErrOrStderr(), "%s %s: %s\n", warn.Sprint("problem:"), uuid, problem)
		}
		return ExitErrorf(1, "%d problems encountered", len(merged.Problems))
	}

	return nil
}

func runBatchDryRun(
	cmd *cobra.Command,
	svc service.FileService,
	uuids []string,
	operation string,
	opts output.FormatOptions,
	formatter output.Formatter,
) error {
	ctx := cmd.Context()
	w := cmd.OutOrStdout()

	type dryRunEntry struct {
		UUID     string `json:"uuid"`
		Filename string `json:"filename,omitempty"`
		Status   string `json:"status"`
	}

	var entries []dryRunEntry
	var hasErrors bool

	for _, uuid := range uuids {
		entry := dryRunEntry{UUID: uuid}
		file, err := svc.Info(ctx, uuid, false)
		if err != nil {
			var apiErr ucare.APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
				entry.Status = "not found"
				hasErrors = true
			} else {
				return fmt.Errorf("looking up %s: %w", uuid, err)
			}
		} else {
			entry.Filename = file.Filename
			entry.Status = "would " + operation
		}
		entries = append(entries, entry)
	}

	if opts.JSON {
		if err := formatter.Format(w, entries); err != nil {
			return err
		}
	} else {
		table := output.NewTableData("UUID", "FILENAME", "STATUS")
		for _, e := range entries {
			table.AddRow(e.UUID, e.Filename, e.Status)
		}
		if err := formatter.Format(w, table); err != nil {
			return err
		}
	}

	if hasErrors {
		return ExitErrorf(1, "some files not found")
	}
	return nil
}

// collectUUIDs merges UUIDs from args and optionally from stdin, deduplicating.
func collectUUIDs(args []string, fromStdin bool) ([]string, error) {
	seen := make(map[string]struct{})
	var result []string

	add := func(uuids []string) {
		for _, u := range uuids {
			if _, exists := seen[u]; !exists {
				seen[u] = struct{}{}
				result = append(result, u)
			}
		}
	}

	add(args)

	if fromStdin {
		stdinUUIDs, err := ReadLinesOrNDJSON(os.Stdin, "uuid")
		if err != nil {
			return nil, fmt.Errorf("reading stdin: %w", err)
		}
		add(stdinUUIDs)
	}

	return result, nil
}

