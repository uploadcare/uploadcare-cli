package validate

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Sentinel errors for validation failures.
var (
	ErrControlChars   = errors.New("input contains control characters")
	ErrPathTraversal  = errors.New("input contains path traversal")
	ErrDoubleEncoding = errors.New("input appears double-URL-encoded")
	ErrInvalidUUID    = errors.New("invalid UUID format")
	ErrInvalidURL     = errors.New("invalid URL")
	ErrStringTooLong  = errors.New("input exceeds maximum length")
)

// Length limits.
const (
	MaxMetadataKeyLen   = 255
	MaxMetadataValueLen = 50 * 1024 // 50 KB
)

// uuidRe matches a canonical lowercase UUID.
var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// groupIDRe matches a group ID in the form UUID~N.
var groupIDRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}~\d+$`)

// doubleEncodedRe detects common double-URL-encoding patterns like %25XX.
var doubleEncodedRe = regexp.MustCompile(`%25[0-9A-Fa-f]{2}`)

// ValidateString checks a generic string input for control characters,
// path traversal sequences, and double encoding.
func ValidateString(s string) error {
	if err := rejectControlChars(s); err != nil {
		return err
	}
	if err := rejectPathTraversal(s); err != nil {
		return err
	}
	if err := rejectDoubleEncoding(s); err != nil {
		return err
	}
	return nil
}

// UUID validates that s is a well-formed lowercase UUID.
func UUID(s string) error {
	if !uuidRe.MatchString(s) {
		return fmt.Errorf("%w: %q", ErrInvalidUUID, s)
	}
	return nil
}

// GroupID validates that s is a well-formed group ID (UUID~N format).
func GroupID(s string) error {
	if !groupIDRe.MatchString(s) {
		return fmt.Errorf("invalid group ID format %q (expected UUID~N, e.g. d52d7136-a2e5-4338-9f45-affbf83b857d~3)", s)
	}
	return nil
}

// URL validates that s is a well-formed HTTP or HTTPS URL.
func URL(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%w: scheme must be http or https, got %q", ErrInvalidURL, u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("%w: missing host", ErrInvalidURL)
	}
	return nil
}

// MetadataKey validates a metadata key: sanitized string within length limit.
func MetadataKey(s string) error {
	if err := ValidateString(s); err != nil {
		return err
	}
	if len(s) > MaxMetadataKeyLen {
		return fmt.Errorf("%w: metadata key length %d exceeds limit %d", ErrStringTooLong, len(s), MaxMetadataKeyLen)
	}
	return nil
}

// MetadataValue validates a metadata value: sanitized string within length limit.
func MetadataValue(s string) error {
	if err := ValidateString(s); err != nil {
		return err
	}
	if len(s) > MaxMetadataValueLen {
		return fmt.Errorf("%w: metadata value length %d exceeds limit %d", ErrStringTooLong, len(s), MaxMetadataValueLen)
	}
	return nil
}

// StringWithMaxLen validates a string with a custom max length.
func StringWithMaxLen(s string, maxLen int) error {
	if err := ValidateString(s); err != nil {
		return err
	}
	if len(s) > maxLen {
		return fmt.Errorf("%w: length %d exceeds limit %d", ErrStringTooLong, len(s), maxLen)
	}
	return nil
}

// rejectControlChars rejects strings containing ASCII control characters
// (0x00-0x1F, 0x7F) except tab (\t) and newline (\n).
func rejectControlChars(s string) error {
	for i, r := range s {
		if r == '\t' || r == '\n' {
			continue
		}
		if (r >= 0x00 && r <= 0x1F) || r == 0x7F {
			return fmt.Errorf("%w: byte 0x%02X at position %d", ErrControlChars, r, i)
		}
	}
	return nil
}

// rejectPathTraversal rejects strings containing ../ or ..\ sequences.
func rejectPathTraversal(s string) error {
	if strings.Contains(s, "../") || strings.Contains(s, `..\\`) || strings.Contains(s, `..\`) {
		return fmt.Errorf("%w: %q", ErrPathTraversal, s)
	}
	return nil
}

// rejectDoubleEncoding detects strings that appear to be double-URL-encoded.
func rejectDoubleEncoding(s string) error {
	if doubleEncodedRe.MatchString(s) {
		return fmt.Errorf("%w: %q", ErrDoubleEncoding, s)
	}
	return nil
}
