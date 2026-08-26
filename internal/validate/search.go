package validate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/file"
)

// Limits and sort keys come from the SDK so an SDK upgrade cannot leave this
// pre-flight validation rejecting values the API accepts.
const (
	MinSearchTextLength = file.MinSearchQueryLength
	MaxSearchLimit      = file.MaxSearchLimit
	MaxSearchWindow     = file.MaxSearchOffsetLimit
	MaxSearchSortKeys   = file.MaxSearchSortKeys
)

var searchMetadataKeyPattern = regexp.MustCompile(`^metadata\[[\w.:-]{1,64}\]$`)

var validSearchSortKeys = map[string]struct{}{
	string(file.SortByScore): {}, string(file.SortByScoreDesc): {},
	string(file.SortByUploadedAt): {}, string(file.SortByUploadedAtDesc): {},
	string(file.SortBySize): {}, string(file.SortBySizeDesc): {},
	string(file.SortByOriginalFilename): {}, string(file.SortByOriginalFilenameDesc): {},
}

// FileSearch validates search options before credentials are resolved or a
// request is sent. Tag filters are expected to be already normalized via
// NormalizeTags. The SDK performs the same validation defensively.
func FileSearch(opts service.FileSearchOptions) error {
	if !hasSearchCondition(opts) {
		return fmt.Errorf("search requires at least one query or filter")
	}
	if opts.Query != "" && utf8.RuneCountInString(opts.Query) < MinSearchTextLength {
		return fmt.Errorf("search query must be at least %d characters", MinSearchTextLength)
	}
	if opts.Phrase != nil {
		phrases := map[string]string{
			"original_filename":  opts.Phrase.OriginalFilename,
			"metadata":           opts.Phrase.Metadata,
			"detected_mime_type": opts.Phrase.DetectedMimeType,
		}
		for field, value := range phrases {
			if value != "" && utf8.RuneCountInString(value) < MinSearchTextLength {
				return fmt.Errorf("search phrase %q must be at least %d characters", field, MinSearchTextLength)
			}
		}
		if opts.Phrase.OriginalFilename != "" && len(opts.Exact["original_filename"]) > 0 {
			return fmt.Errorf("search field %q cannot appear in both --phrase and --exact", "original_filename")
		}
		if opts.Phrase.DetectedMimeType != "" && len(opts.Exact["detected_mime_type"]) > 0 {
			return fmt.Errorf("search field %q cannot appear in both --phrase and --exact", "detected_mime_type")
		}
		if opts.Phrase.Metadata != "" {
			for key := range opts.Exact {
				if strings.HasPrefix(key, "metadata[") {
					return fmt.Errorf("search field %q cannot appear in both --phrase and --exact", key)
				}
			}
		}
	}
	for key, values := range opts.Exact {
		if !validSearchExactKey(key) {
			return fmt.Errorf("unsupported --exact field %q", key)
		}
		if len(values) == 0 {
			return fmt.Errorf("--exact %q requires a non-empty value", key)
		}
		for _, value := range values {
			if value == "" {
				return fmt.Errorf("--exact %q requires a non-empty value", key)
			}
		}
	}
	if opts.Limit < 1 || opts.Limit > MaxSearchLimit {
		return fmt.Errorf("--limit must be between 1 and %d", MaxSearchLimit)
	}
	if opts.Offset < 0 || opts.Offset > MaxSearchWindow-opts.Limit {
		return fmt.Errorf("--offset plus --limit must not exceed %d", MaxSearchWindow)
	}
	if len(opts.Sort) > MaxSearchSortKeys {
		return fmt.Errorf("--sort accepts at most %d values", MaxSearchSortKeys)
	}
	seenSort := make(map[string]struct{}, len(opts.Sort))
	for _, key := range opts.Sort {
		if _, ok := validSearchSortKeys[key]; !ok {
			return fmt.Errorf("unsupported --sort value %q", key)
		}
		base := strings.TrimPrefix(key, "-")
		if _, ok := seenSort[base]; ok {
			return fmt.Errorf("--sort values must be unique and cannot include both directions of %q", base)
		}
		seenSort[base] = struct{}{}
	}
	return nil
}

func hasSearchCondition(opts service.FileSearchOptions) bool {
	return opts.Query != "" ||
		(opts.Phrase != nil && (opts.Phrase.OriginalFilename != "" || opts.Phrase.Metadata != "" || opts.Phrase.DetectedMimeType != "")) ||
		len(opts.Exact) > 0 || opts.DatetimeUploaded != nil || opts.Size != nil || opts.IsImage != nil ||
		(opts.Tags != nil && (len(opts.Tags.Any) > 0 || len(opts.Tags.All) > 0 || len(opts.Tags.None) > 0))
}

func validSearchExactKey(key string) bool {
	switch key {
	case "uuid", "detected_mime_type", "original_filename":
		return true
	default:
		return searchMetadataKeyPattern.MatchString(key)
	}
}
