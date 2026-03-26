package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
)

var validUsageMetrics = []string{"traffic", "storage", "operations"}

func isValidUsageMetric(m string) bool {
	for _, v := range validUsageMetrics {
		if v == m {
			return true
		}
	}
	return false
}

func newProjectUsageCmd(usageSvc service.UsageService) *cobra.Command {
	var (
		from   string
		to     string
		metric string
	)

	cmd := &cobra.Command{
		Use:   "usage <project>",
		Short: "Get usage metrics",
		Long: `Get usage metrics for a project (by name or pub_key).

Dates must be in YYYY-MM-DD format. Maximum range is 90 days.

Without --metric, returns all metrics (traffic, storage, operations).
With --metric, returns daily data for a single metric.

Requires --project-api-token.`,
		Example: `  # Combined usage for last month
  uploadcare project usage abc123 --from 2026-02-01 --to 2026-03-01

  # Traffic only
  uploadcare project usage "My App" --from 2026-02-01 --to 2026-03-01 --metric traffic

  # As JSON
  uploadcare project usage abc123 --from 2026-02-01 --to 2026-03-01 --json all`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pubKey := resolveProjectPubKey(cmd, args[0])

			// Validate date format.
			if _, err := time.Parse("2006-01-02", from); err != nil {
				return ExitErrorf(2, "invalid --from date %q: use YYYY-MM-DD format", from)
			}
			if _, err := time.Parse("2006-01-02", to); err != nil {
				return ExitErrorf(2, "invalid --to date %q: use YYYY-MM-DD format", to)
			}

			if metric != "" && !isValidUsageMetric(metric) {
				return ExitErrorf(2, "invalid --metric %q: must be one of: %v", metric, validUsageMetrics)
			}

			svc := usageSvc
			if svc == nil {
				var err error
				svc, err = usageServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if metric != "" {
				return runMetricUsage(cmd, svc, formatter, opts, pubKey, metric, from, to)
			}
			return runCombinedUsage(cmd, svc, formatter, opts, pubKey, from, to)
		},
	}

	f := cmd.Flags()
	f.StringVar(&from, "from", "", "Start date (YYYY-MM-DD, required)")
	f.StringVar(&to, "to", "", "End date (YYYY-MM-DD, required)")
	f.StringVar(&metric, "metric", "", "Single metric: traffic, storage, or operations")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runCombinedUsage(
	cmd *cobra.Command,
	svc service.UsageService,
	formatter output.Formatter,
	opts output.FormatOptions,
	pubKey, from, to string,
) error {
	result, err := svc.Combined(cmd.Context(), pubKey, from, to)
	if err != nil {
		return err
	}

	if opts.JSON {
		return formatter.Format(cmd.OutOrStdout(), result)
	}

	table := output.NewTableData("DATE", "TRAFFIC", "STORAGE", "OPERATIONS")
	for _, d := range result.Data {
		table.AddRow(
			d.Date,
			strconv.FormatInt(d.Traffic, 10),
			strconv.FormatInt(d.Storage, 10),
			strconv.FormatInt(d.Operations, 10),
		)
	}

	if !opts.Quiet && len(result.Units) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Units: traffic=%s, storage=%s, operations=%s\n",
			result.Units["traffic"], result.Units["storage"], result.Units["operations"])
	}

	return formatter.Format(cmd.OutOrStdout(), table)
}

func runMetricUsage(
	cmd *cobra.Command,
	svc service.UsageService,
	formatter output.Formatter,
	opts output.FormatOptions,
	pubKey, metric, from, to string,
) error {
	result, err := svc.Metric(cmd.Context(), pubKey, metric, from, to)
	if err != nil {
		return err
	}

	if opts.JSON {
		return formatter.Format(cmd.OutOrStdout(), result)
	}

	if !opts.Quiet {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Metric: %s (unit: %s)\n", result.Metric, result.Unit)
	}

	table := output.NewTableData("DATE", "VALUE")
	for _, d := range result.Data {
		table.AddRow(d.Date, strconv.FormatInt(d.Value, 10))
	}
	return formatter.Format(cmd.OutOrStdout(), table)
}
