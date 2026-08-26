package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
)

func newTagCmd(tagSvc service.TagService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage file tags",
		Long: `Manage ordered per-file tags.

Tags are normalized to lowercase and may contain letters, digits, dots,
underscores, and hyphens. Use --dry-run on mutation commands to preview a
change without applying it.`,
	}
	cmd.AddCommand(newTagListCmd(tagSvc))
	cmd.AddCommand(newTagReplaceCmd(tagSvc))
	cmd.AddCommand(newTagUpdateCmd(tagSvc))
	cmd.AddCommand(newTagClearCmd(tagSvc))
	return cmd
}

func newTagListCmd(tagSvc service.TagService) *cobra.Command {
	return &cobra.Command{
		Use:   "list <file-uuid>",
		Short: "List a file's tags",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.UUID(args[0]); err != nil {
				return usageError(err)
			}
			svc, err := resolveTagService(cmd, tagSvc)
			if err != nil {
				return err
			}
			tags, err := svc.List(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)
			if opts.JSON {
				if tags == nil {
					tags = []string{}
				}
				return formatter.Format(cmd.OutOrStdout(), map[string][]string{"tags": tags})
			}
			if len(tags) == 0 {
				if opts.Quiet {
					return nil
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "No tags found")
				return err
			}
			table := output.NewTableData()
			for _, value := range tags {
				table.AddRow(value)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}
}

func newTagReplaceCmd(tagSvc service.TagService) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "replace <file-uuid> <tag>...",
		Short: "Replace a file's complete tag set",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.UUID(args[0]); err != nil {
				return usageError(err)
			}
			tags, err := validate.NormalizeTags(args[1:], validate.MaxTagCount)
			if err != nil {
				return usageError(err)
			}
			return runTagReplace(cmd, tagSvc, args[0], tags, dryRun)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without applying")
	return cmd
}

func newTagUpdateCmd(tagSvc service.TagService) *cobra.Command {
	var add, deleteTags []string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "update <file-uuid>",
		Short: "Atomically add and delete tags",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.UUID(args[0]); err != nil {
				return usageError(err)
			}
			if len(add) == 0 && len(deleteTags) == 0 {
				return usageError(fmt.Errorf("at least one --add or --delete is required"))
			}
			normalizedAdd, err := validate.NormalizeTags(add, validate.MaxTagCount)
			if err != nil {
				return usageError(fmt.Errorf("--add: %w", err))
			}
			normalizedDelete, err := validate.NormalizeTags(deleteTags, 0)
			if err != nil {
				return usageError(fmt.Errorf("--delete: %w", err))
			}
			svc, err := resolveTagService(cmd, tagSvc)
			if err != nil {
				return err
			}
			var result *service.TagChangeResult
			if dryRun {
				current, err := svc.List(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				result = updateDiff(current, normalizedAdd, normalizedDelete)
			} else {
				result, err = svc.Update(cmd.Context(), args[0], service.TagUpdateOptions{Add: normalizedAdd, Delete: normalizedDelete})
				if err != nil {
					return err
				}
			}
			return formatTagChange(cmd, result, dryRun)
		},
	}
	cmd.Flags().StringArrayVar(&add, "add", nil, "Tag to add (repeatable)")
	cmd.Flags().StringArrayVar(&deleteTags, "delete", nil, "Tag to delete (repeatable)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without applying")
	return cmd
}

func newTagClearCmd(tagSvc service.TagService) *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "clear <file-uuid>",
		Short: "Remove all tags from a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validate.UUID(args[0]); err != nil {
				return usageError(err)
			}
			return runTagReplace(cmd, tagSvc, args[0], []string{}, dryRun)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would change without applying")
	return cmd
}

func runTagReplace(cmd *cobra.Command, injected service.TagService, uuid string, tags []string, dryRun bool) error {
	svc, err := resolveTagService(cmd, injected)
	if err != nil {
		return err
	}
	var result *service.TagChangeResult
	if dryRun {
		current, err := svc.List(cmd.Context(), uuid)
		if err != nil {
			return err
		}
		result = replacementDiff(current, tags)
	} else {
		result, err = svc.Replace(cmd.Context(), uuid, tags)
		if err != nil {
			return err
		}
	}
	return formatTagChange(cmd, result, dryRun)
}

func resolveTagService(cmd *cobra.Command, injected service.TagService) (service.TagService, error) {
	if injected != nil {
		return injected, nil
	}
	return tagServiceFromCmd(cmd)
}

func formatTagChange(cmd *cobra.Command, result *service.TagChangeResult, dryRun bool) error {
	opts := formatOptionsFromCmd(cmd)
	formatter := output.New(opts)
	if opts.JSON {
		if dryRun {
			return formatter.Format(cmd.OutOrStdout(), struct {
				Status string `json:"status"`
				*service.TagChangeResult
			}{"would change", result})
		}
		return formatter.Format(cmd.OutOrStdout(), result)
	}
	table := output.NewTableData("FIELD", "VALUE")
	if dryRun {
		table.AddRow("Status", "Would change")
	}
	table.AddRow("Tags", displayTags(result.Tags))
	table.AddRow("Added", displayTags(result.Added))
	table.AddRow("Deleted", displayTags(result.Deleted))
	return formatter.Format(cmd.OutOrStdout(), table)
}

func displayTags(tags []string) string {
	if len(tags) == 0 {
		return "(none)"
	}
	return strings.Join(tags, ", ")
}

func replacementDiff(current, replacement []string) *service.TagChangeResult {
	currentSet := stringSet(current)
	replacementSet := stringSet(replacement)
	result := &service.TagChangeResult{
		Tags: append([]string{}, replacement...), Added: []string{}, Deleted: []string{},
	}
	for _, value := range replacement {
		if _, exists := currentSet[value]; !exists {
			result.Added = append(result.Added, value)
		}
	}
	for _, value := range current {
		if _, exists := replacementSet[value]; !exists {
			result.Deleted = append(result.Deleted, value)
		}
	}
	return result
}

func updateDiff(current, add, deleteTags []string) *service.TagChangeResult {
	deleteSet := stringSet(deleteTags)
	result := &service.TagChangeResult{Tags: []string{}, Added: []string{}, Deleted: []string{}}
	remaining := make(map[string]struct{}, len(current)+len(add))
	for _, value := range current {
		if _, deleted := deleteSet[value]; deleted {
			result.Deleted = append(result.Deleted, value)
			continue
		}
		if _, duplicate := remaining[value]; !duplicate {
			remaining[value] = struct{}{}
			result.Tags = append(result.Tags, value)
		}
	}
	for _, value := range add {
		if _, exists := remaining[value]; exists {
			continue
		}
		remaining[value] = struct{}{}
		result.Tags = append(result.Tags, value)
		result.Added = append(result.Added, value)
	}
	return result
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func usageError(err error) error {
	return &ExitError{Code: 2, Err: err}
}
