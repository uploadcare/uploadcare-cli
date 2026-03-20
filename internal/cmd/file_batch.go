package cmd

import (
	"errors"
	"fmt"
	"net/http"
	"os"

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
		Args:  cobra.ArbitraryArgs,
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
		Args:  cobra.ArbitraryArgs,
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
		for uuid, problem := range merged.Problems {
			fmt.Fprintf(cmd.ErrOrStderr(), "problem: %s: %s\n", uuid, problem)
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

