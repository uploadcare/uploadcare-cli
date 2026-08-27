package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
)

func newFileSearchCmd(fileSvc service.FileService) *cobra.Command {
	var (
		phraseValues, exactValues        []string
		uploadedGt, uploadedGte          string
		uploadedLt, uploadedLte          string
		sizeGt, sizeGte, sizeLt, sizeLte uint64
		isImage                          string
		fuzziness                        bool
		tagAny, tagAll, tagNone          []string
		sortValues                       []string
		limit, offset                    int
		pageAll, includeAppData          bool
	)

	buildOptions := func(cmd *cobra.Command, args []string) (service.FileSearchOptions, error) {
		opts := service.FileSearchOptions{
			Limit: limit, Offset: offset, IncludeAppData: includeAppData,
			Fuzziness: fuzziness, Sort: sortValues,
		}
		if len(args) == 1 {
			opts.Query = args[0]
		}
		phrase, err := parseSearchPhrases(phraseValues)
		if err != nil {
			return opts, err
		}
		opts.Phrase = phrase
		opts.Exact, err = parseSearchExact(exactValues)
		if err != nil {
			return opts, err
		}

		dateRange := &service.FileSearchDatetime{}
		dateFlags := []struct {
			name, raw string
			dst       **time.Time
		}{
			{"uploaded-gt", uploadedGt, &dateRange.Gt},
			{"uploaded-gte", uploadedGte, &dateRange.Gte},
			{"uploaded-lt", uploadedLt, &dateRange.Lt},
			{"uploaded-lte", uploadedLte, &dateRange.Lte},
		}
		for _, value := range dateFlags {
			if !cmd.Flags().Changed(value.name) {
				continue
			}
			parsed, err := time.Parse(time.RFC3339, value.raw)
			if err != nil {
				return opts, fmt.Errorf("invalid --%s value %q: expected RFC3339 timestamp", value.name, value.raw)
			}
			*value.dst = &parsed
		}
		if dateRange.Gt != nil || dateRange.Gte != nil || dateRange.Lt != nil || dateRange.Lte != nil {
			opts.DatetimeUploaded = dateRange
		}

		sizeRange := &service.FileSearchSize{}
		if cmd.Flags().Changed("size-gt") {
			sizeRange.Gt = &sizeGt
		}
		if cmd.Flags().Changed("size-gte") {
			sizeRange.Gte = &sizeGte
		}
		if cmd.Flags().Changed("size-lt") {
			sizeRange.Lt = &sizeLt
		}
		if cmd.Flags().Changed("size-lte") {
			sizeRange.Lte = &sizeLte
		}
		if sizeRange.Gt != nil || sizeRange.Gte != nil || sizeRange.Lt != nil || sizeRange.Lte != nil {
			opts.Size = sizeRange
		}

		if cmd.Flags().Changed("is-image") {
			switch isImage {
			case "true":
				b := true
				opts.IsImage = &b
			case "false":
				b := false
				opts.IsImage = &b
			default:
				return opts, fmt.Errorf("invalid --is-image value: %q (must be \"true\" or \"false\")", isImage)
			}
		}

		if len(tagAny) > 0 || len(tagAll) > 0 || len(tagNone) > 0 {
			tags := &service.FileSearchTags{}
			if tags.Any, err = normalizeTagFilter("--tag-any", tagAny); err != nil {
				return opts, err
			}
			if tags.All, err = normalizeTagFilter("--tag-all", tagAll); err != nil {
				return opts, err
			}
			if tags.None, err = normalizeTagFilter("--tag-none", tagNone); err != nil {
				return opts, err
			}
			opts.Tags = tags
		}
		return opts, nil
	}

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search files by text, fields, ranges, and tags",
		Long: `Search the project's file index.

At least one query or filter is required. The optional query and each phrase
value must contain at least four characters. Repeat --exact to match any of
several exact values, and repeat --sort to define an ordered sort list.

The API serves at most the first 1000 matches of a search: --offset plus
--limit cannot exceed 1000, and --page-all stops after streaming the first
1000 matches.

The search index updates asynchronously, so recent file, metadata, and tag
changes may not be immediately visible. Removed files are excluded. Pages
are filled by following the API's next cursor, so offset-stepped pages may
occasionally overlap; prefer --page-all when completeness matters.`,
		Example: `  # Search PDFs that have approved but not archived tags
  uploadcare file search invoice \
    --exact detected_mime_type=application/pdf \
    --tag-all approved --tag-none archived \
    --sort score --sort=-datetime_uploaded

  # Search an exact metadata value
  uploadcare file search --exact 'metadata[camera]=Canon' --json uuid,filename,tags,highlight`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			searchOpts, err := buildOptions(cmd, args)
			if err != nil {
				return usageError(err)
			}
			if err := validate.FileSearch(searchOpts); err != nil {
				return usageError(err)
			}

			svc := fileSvc
			if svc == nil {
				svc, err = fileServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}
			formatOpts := formatOptionsFromCmd(cmd)
			if pageAll {
				// Page through the window with the largest page the API
				// allows unless the user picked an explicit page size.
				if !cmd.Flags().Changed("limit") {
					searchOpts.Limit = min(validate.MaxSearchLimit, validate.MaxSearchWindow-searchOpts.Offset)
				}
				return runFileSearchAll(cmd, svc, searchOpts, formatOpts, includeAppData)
			}

			result, err := svc.Search(cmd.Context(), searchOpts)
			if err != nil {
				return err
			}
			formatter := output.New(formatOpts)
			if formatOpts.JSON {
				err = formatter.Format(cmd.OutOrStdout(), result.Files)
			} else {
				err = formatter.Format(cmd.OutOrStdout(), fileSearchTable(result.Files, includeAppData))
			}
			if err != nil {
				return err
			}
			if !formatOpts.Quiet {
				_, err = fmt.Fprintf(cmd.ErrOrStderr(), "Found %d matches; showing %d.\n", result.Total, len(result.Files))
			}
			return err
		},
	}

	f := cmd.Flags()
	f.StringArrayVar(&phraseValues, "phrase", nil, "Phrase match as field=value (repeatable by field)")
	f.StringArrayVar(&exactValues, "exact", nil, "Exact match as field=value (repeatable)")
	f.StringVar(&uploadedGt, "uploaded-gt", "", "Uploaded after RFC3339 timestamp")
	f.StringVar(&uploadedGte, "uploaded-gte", "", "Uploaded at or after RFC3339 timestamp")
	f.StringVar(&uploadedLt, "uploaded-lt", "", "Uploaded before RFC3339 timestamp")
	f.StringVar(&uploadedLte, "uploaded-lte", "", "Uploaded at or before RFC3339 timestamp")
	f.Uint64Var(&sizeGt, "size-gt", 0, "Size greater than bytes")
	f.Uint64Var(&sizeGte, "size-gte", 0, "Size greater than or equal to bytes")
	f.Uint64Var(&sizeLt, "size-lt", 0, "Size less than bytes")
	f.Uint64Var(&sizeLte, "size-lte", 0, "Size less than or equal to bytes")
	f.StringVar(&isImage, "is-image", "", "Filter by image status (true/false)")
	f.BoolVar(&fuzziness, "fuzziness", false, "Enable fuzzy text and phrase matching")
	f.StringArrayVar(&tagAny, "tag-any", nil, "Match at least one tag (repeatable)")
	f.StringArrayVar(&tagAll, "tag-all", nil, "Match every tag (repeatable)")
	f.StringArrayVar(&tagNone, "tag-none", nil, "Exclude any matching tag (repeatable)")
	f.StringArrayVar(&sortValues, "sort", nil, "Sort field, optionally prefixed with - (repeatable, max 4)")
	f.IntVar(&limit, "limit", 20, "Number of matches per page (1-100)")
	f.IntVar(&offset, "offset", 0, "Result offset (offset + limit must not exceed 1000)")
	f.BoolVar(&pageAll, "page-all", false, "Stream all search result pages (at most the first 1000 matches)")
	f.BoolVar(&includeAppData, "include-appdata", false, "Include application data")
	return cmd
}

func normalizeTagFilter(flag string, values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	normalized, err := validate.NormalizeTags(values, validate.MaxTagCount)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", flag, err)
	}
	return normalized, nil
}

func parseSearchPhrases(values []string) (*service.FileSearchPhrase, error) {
	if len(values) == 0 {
		return nil, nil
	}
	phrase := &service.FileSearchPhrase{}
	seen := make(map[string]struct{}, len(values))
	for _, entry := range values {
		field, value, err := splitSearchFieldValue("--phrase", entry)
		if err != nil {
			return nil, err
		}
		if _, duplicate := seen[field]; duplicate {
			return nil, fmt.Errorf("--phrase field %q may appear only once", field)
		}
		seen[field] = struct{}{}
		switch field {
		case "original_filename":
			phrase.OriginalFilename = value
		case "metadata":
			phrase.Metadata = value
		case "detected_mime_type":
			phrase.DetectedMimeType = value
		default:
			return nil, fmt.Errorf("unsupported --phrase field %q", field)
		}
	}
	return phrase, nil
}

func parseSearchExact(values []string) (map[string][]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	exact := make(map[string][]string)
	for _, entry := range values {
		field, value, err := splitSearchFieldValue("--exact", entry)
		if err != nil {
			return nil, err
		}
		exact[field] = append(exact[field], value)
	}
	return exact, nil
}

func splitSearchFieldValue(flag, entry string) (string, string, error) {
	field, value, ok := strings.Cut(entry, "=")
	if !ok || field == "" || value == "" {
		return "", "", fmt.Errorf("%s value %q must use non-empty field=value syntax", flag, entry)
	}
	return field, value, nil
}

func fileSearchTable(files []service.File, includeAppData bool) *output.TableData {
	headers := []string{"UUID", "SIZE", "FILENAME", "MIME TYPE", "UPLOADED"}
	if includeAppData {
		headers = append(headers, "APPDATA")
	}
	table := output.NewTableData(headers...).Flexible(2)
	for _, file := range files {
		row := []string{file.UUID, strconv.FormatInt(file.Size, 10), file.Filename, file.MimeType, formatTime(file.DatetimeUploaded)}
		if includeAppData {
			row = append(row, truncateAppData(file.AppData, 50))
		}
		table.AddRow(row...)
	}
	return table
}

func runFileSearchAll(cmd *cobra.Command, svc service.FileService, searchOpts service.FileSearchOptions, opts output.FormatOptions, includeAppData bool) error {
	count := 0
	total, err := svc.IterateSearch(cmd.Context(), searchOpts, func(file service.File) error {
		count++
		if opts.Quiet {
			return nil
		}
		if opts.JSON {
			return output.NDJSONLine(cmd.OutOrStdout(), &file, opts.Fields, opts.JQ)
		}
		if includeAppData {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\t%s\t%s\t%s\t%s\n",
				file.UUID, file.Size, output.SanitizeCell(file.Filename), file.MimeType, formatTime(file.DatetimeUploaded),
				truncateAppData(file.AppData, 50))
			return err
		}
		_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\t%s\t%s\t%s\n",
			file.UUID, file.Size, output.SanitizeCell(file.Filename), file.MimeType, formatTime(file.DatetimeUploaded))
		return err
	})
	if err != nil {
		return err
	}
	if !opts.Quiet {
		_, err = fmt.Fprintf(cmd.ErrOrStderr(), "Found %d matches; showing %d.\n", total, count)
	}
	return err
}
