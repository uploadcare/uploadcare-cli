package validate

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/uploadcare/uploadcare-go/v2/tag"
)

// Limits come from the SDK so an SDK upgrade cannot leave this pre-flight
// validation rejecting values the API accepts.
const (
	MaxTagLength = tag.MaxLength
	MaxTagCount  = tag.MaxCount
)

var tagPattern = regexp.MustCompile(`^[a-z0-9._-]+$`)

// NormalizeTags returns normalized, de-duplicated tags in first-seen order.
// A maxCount of zero disables count validation.
func NormalizeTags(tags []string, maxCount int) ([]string, error) {
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == "" {
			return nil, fmt.Errorf("tag must not be blank")
		}
		if utf8.RuneCountInString(value) > MaxTagLength {
			return nil, fmt.Errorf("tag %q exceeds the maximum length of %d characters", value, MaxTagLength)
		}
		if !tagPattern.MatchString(value) {
			return nil, fmt.Errorf("tag %q contains invalid characters (allowed: a-z, 0-9, dot, underscore, hyphen)", value)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if maxCount > 0 && len(normalized) > maxCount {
		return nil, fmt.Errorf("too many tags: %d (maximum %d)", len(normalized), maxCount)
	}
	return normalized, nil
}
