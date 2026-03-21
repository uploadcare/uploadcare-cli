package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
)

func newGroupCmd(groupSvc service.GroupService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manage file groups",
		Long: `Manage file groups in the current Uploadcare project.

A file group is an ordered collection of files referenced by a single
group ID (format: <uuid>~<count>). Groups are useful for multi-file
uploads and batch operations.

Subcommands: list, info, create, delete.`,
	}

	cmd.AddCommand(newGroupListCmd(groupSvc))
	cmd.AddCommand(newGroupInfoCmd(groupSvc))
	cmd.AddCommand(newGroupCreateCmd(groupSvc))
	cmd.AddCommand(newGroupDeleteCmd(groupSvc))

	return cmd
}

func newGroupListCmd(groupSvc service.GroupService) *cobra.Command {
	var (
		ordering      string
		limit         int
		startingPoint string
		pageAll       bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List file groups",
		Long: `List file groups in the current Uploadcare project with pagination.

Returns groups sorted by --ordering (default: datetime_created ascending).
Prefix the ordering field with - for descending (e.g., -datetime_created).

By default returns one page of up to --limit groups (default: 100).
Use --page-all to stream ALL groups as NDJSON (one JSON object per line).

JSON fields: id, files_count, datetime_created, datetime_stored, cdn_url, url.`,
		Example: `  # List groups as JSON
  uploadcare group list --json all

  # List groups, newest first
  uploadcare group list --ordering -datetime_created --json all

  # Stream all group IDs
  uploadcare group list --page-all --json id`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := groupSvc
			if svc == nil {
				var err error
				svc, err = groupServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			listOpts := service.GroupListOptions{
				Ordering:      ordering,
				Limit:         limit,
				StartingPoint: startingPoint,
			}

			if pageAll {
				return runGroupListAll(cmd, svc, listOpts, opts)
			}

			result, err := svc.List(cmd.Context(), listOpts)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), result.Groups)
			}

			table := output.NewTableData("ID", "FILES", "CREATED", "STORED")
			for _, g := range result.Groups {
				stored := ""
				if g.DatetimeStored != nil {
					stored = formatTime(*g.DatetimeStored)
				}
				table.AddRow(
					g.ID,
					strconv.Itoa(g.FilesCount),
					formatTime(g.DatetimeCreated),
					stored,
				)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	f := cmd.Flags()
	f.StringVar(&ordering, "ordering", "datetime_created", "Sort order (prefix - for descending)")
	f.IntVar(&limit, "limit", 100, "Number of groups per page")
	f.StringVar(&startingPoint, "starting-point", "", "Starting point (RFC3339 datetime)")
	f.BoolVar(&pageAll, "page-all", false, "Stream all pages as NDJSON")

	return cmd
}

func runGroupListAll(cmd *cobra.Command, svc service.GroupService, listOpts service.GroupListOptions, opts output.FormatOptions) error {
	if opts.Quiet {
		return svc.Iterate(cmd.Context(), listOpts, func(g service.Group) error {
			return nil
		})
	}

	w := cmd.OutOrStdout()
	return svc.Iterate(cmd.Context(), listOpts, func(g service.Group) error {
		if opts.JSON {
			return output.NDJSONLine(w, &g, opts.Fields, opts.JQ)
		}
		stored := ""
		if g.DatetimeStored != nil {
			stored = formatTime(*g.DatetimeStored)
		}
		_, err := fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
			g.ID, g.FilesCount, formatTime(g.DatetimeCreated), stored)
		return err
	})
}

func newGroupInfoCmd(groupSvc service.GroupService) *cobra.Command {
	return &cobra.Command{
		Use:   "info <group-id>",
		Short: "Get group details",
		Long: `Get detailed information about a file group by its group ID.

The group ID format is <uuid>~<count> (e.g. 740e1b8c-...~3).

JSON fields: id, files_count, datetime_created, datetime_stored,
cdn_url, url, files (array of file objects).`,
		Example: `  # Get group info
  uploadcare group info "740e1b8c-1ad8-4324-b7ec-112345678900~3"

  # Get group info as JSON
  uploadcare group info "740e1b8c-1ad8-4324-b7ec-112345678900~3" --json all`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := args[0]
			if err := validate.GroupID(groupID); err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			svc := groupSvc
			if svc == nil {
				var err error
				svc, err = groupServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			g, err := svc.Info(cmd.Context(), groupID)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), g)
			}

			table := &output.TableData{}
			table.AddRow("ID:", g.ID)
			table.AddRow("Files:", strconv.Itoa(g.FilesCount))
			table.AddRow("Created:", formatTime(g.DatetimeCreated))
			if g.DatetimeStored != nil {
				table.AddRow("Stored:", formatTime(*g.DatetimeStored))
			}
			table.AddRow("CDN URL:", g.CDNURL)
			if g.URL != "" {
				table.AddRow("URL:", g.URL)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}
}

func newGroupCreateCmd(groupSvc service.GroupService) *cobra.Command {
	var (
		fromStdin bool
		dryRun    bool
	)

	cmd := &cobra.Command{
		Use:   "create <uuid>...",
		Short: "Create a file group",
		Long: `Create a new file group from one or more file UUIDs.

Accepts UUIDs as positional arguments, from stdin (--from-stdin), or both.
All UUIDs are validated before creating the group.

Use --dry-run to validate UUIDs and preview the operation without creating.

JSON fields: id, files_count, datetime_created, cdn_url.`,
		Example: `  # Create a group from specific files
  uploadcare group create UUID1 UUID2 UUID3

  # Create a group from piped UUIDs
  uploadcare file list --page-all --stored true --json uuid \
    | head -5 | uploadcare group create --from-stdin

  # Create a group and get JSON output
  uploadcare group create UUID1 UUID2 --json all

  # Dry run: validate without creating
  uploadcare group create UUID1 UUID2 --dry-run`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := groupSvc
			if svc == nil {
				var err error
				svc, err = groupServiceFromCmd(cmd)
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

			for _, uuid := range uuids {
				if err := validate.UUID(uuid); err != nil {
					return &ExitError{Code: 2, Err: err}
				}
			}

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]interface{}{
						"files":  uuids,
						"status": "would create",
					})
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would create group with %d file(s)\n", len(uuids))
				return nil
			}

			g, err := svc.Create(cmd.Context(), uuids)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), g)
			}

			table := &output.TableData{}
			table.AddRow("ID:", g.ID)
			table.AddRow("Files:", strconv.Itoa(g.FilesCount))
			table.AddRow("Created:", formatTime(g.DatetimeCreated))
			table.AddRow("CDN URL:", g.CDNURL)
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	cmd.Flags().BoolVar(&fromStdin, "from-stdin", false, "Read UUIDs from stdin")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate without creating")

	return cmd
}

func newGroupDeleteCmd(groupSvc service.GroupService) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "delete <group-id>",
		Short: "Delete a file group",
		Long: `Delete a file group by its group ID.

Deleting a group does NOT delete the files within it — they remain in the
project. Only the grouping is removed.

Use --dry-run to verify the group exists and see its details without deleting.

JSON fields: id, status.`,
		Example: `  # Delete a group
  uploadcare group delete "740e1b8c-1ad8-4324-b7ec-112345678900~3"

  # Delete and confirm with JSON
  uploadcare group delete "740e1b8c-1ad8-4324-b7ec-112345678900~3" --json all

  # Dry run: verify group exists
  uploadcare group delete "740e1b8c-1ad8-4324-b7ec-112345678900~3" --dry-run`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			groupID := args[0]
			if err := validate.GroupID(groupID); err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			svc := groupSvc
			if svc == nil {
				var err error
				svc, err = groupServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if dryRun {
				g, err := svc.Info(cmd.Context(), groupID)
				if err != nil {
					return err
				}
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]string{
						"id":     g.ID,
						"status": "would delete",
					})
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would delete group %s (%d files)\n", g.ID, g.FilesCount)
				return nil
			}

			if err := svc.Delete(cmd.Context(), groupID); err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]string{
					"id":     groupID,
					"status": "deleted",
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Deleted group %s\n", groupID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Verify group exists without deleting")

	return cmd
}
