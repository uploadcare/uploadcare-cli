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

func newFileUploadFromURLCmd(fileSvc service.FileService) *cobra.Command {
	var (
		store           string
		metadata        []string
		timeout         time.Duration
		checkDuplicates bool
		saveDuplicates  bool
		dryRun          bool
		fromStdin       bool
	)

	cmd := &cobra.Command{
		Use:   "upload-from-url <url>...",
		Short: "Upload file from URL",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch store {
			case "auto", "true", "false":
			default:
				return ExitErrorf(2, "invalid --store value: %q (must be \"auto\", \"true\", or \"false\")", store)
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

			meta, err := parseMetadata(metadata)
			if err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			urls := append([]string{}, args...)
			if fromStdin {
				stdinURLs, err := ReadLinesOrNDJSON(cmd.InOrStdin(), "url")
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				urls = append(urls, stdinURLs...)
			}

			if len(urls) == 0 {
				return ExitErrorf(2, "no URLs provided")
			}

			for _, u := range urls {
				if err := validate.URL(u); err != nil {
					return &ExitError{Code: 2, Err: fmt.Errorf("invalid URL %q: %w", u, err)}
				}
			}

			if dryRun {
				return runUploadFromURLDryRun(cmd, urls, opts, formatter)
			}

			var results []*service.File
			for _, u := range urls {
				result, err := svc.UploadFromURL(cmd.Context(), service.URLUploadParams{
					URL:             u,
					Store:           store,
					Metadata:        meta,
					Timeout:         timeout,
					CheckDuplicates: checkDuplicates,
					SaveDuplicates:  saveDuplicates,
				})
				if err != nil {
					return fmt.Errorf("uploading %q: %w", u, err)
				}
				results = append(results, result)
			}

			if len(results) == 1 {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), results[0])
				}
				return formatter.Format(cmd.OutOrStdout(), fileInfoTable(results[0]))
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), results)
			}

			table := output.NewTableData("UUID", "SIZE", "FILENAME")
			for _, r := range results {
				table.AddRow(r.UUID, strconv.FormatInt(r.Size, 10), r.Filename)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	f := cmd.Flags()
	f.StringVar(&store, "store", "auto", "File storage behavior (auto, true, false)")
	f.StringSliceVar(&metadata, "metadata", nil, "Metadata key=value pairs (repeatable)")
	f.DurationVar(&timeout, "timeout", 5*time.Minute, "Max wait time for upload to complete")
	f.BoolVar(&checkDuplicates, "check-duplicates", false, "Check for duplicate URLs")
	f.BoolVar(&saveDuplicates, "save-duplicates", false, "Save duplicate URL information")
	f.BoolVar(&dryRun, "dry-run", false, "Validate URLs without uploading")
	f.BoolVar(&fromStdin, "from-stdin", false, "Read URLs from stdin")

	return cmd
}

func runUploadFromURLDryRun(cmd *cobra.Command, urls []string, opts output.FormatOptions, formatter output.Formatter) error {
	type dryRunEntry struct {
		URL    string `json:"url"`
		Status string `json:"status"`
	}
	var entries []dryRunEntry
	for _, u := range urls {
		status := "ok"
		if err := validate.URL(u); err != nil {
			status = err.Error()
		}
		entries = append(entries, dryRunEntry{URL: u, Status: status})
	}

	if opts.JSON {
		return formatter.Format(cmd.OutOrStdout(), entries)
	}

	table := output.NewTableData("URL", "STATUS")
	for _, e := range entries {
		table.AddRow(e.URL, e.Status)
	}
	return formatter.Format(cmd.OutOrStdout(), table)
}
