package validate

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateString_Clean(t *testing.T) {
	clean := []string{
		"normal-string",
		"hello world",
		"file.jpg",
		"path/to/file",
		"with\ttab",
		"with\nnewline",
		"unicode: ñ ü ö 日本語",
		"https://example.com/path?q=1&r=2",
		"",
	}
	for _, s := range clean {
		if err := ValidateString(s); err != nil {
			t.Errorf("ValidateString(%q) = %v, want nil", s, err)
		}
	}
}

func TestRejectControlChars(t *testing.T) {
	cases := []struct {
		input string
		want  error
	}{
		{"normal", nil},
		{"with\ttab", nil},
		{"with\nnewline", nil},
		{"null\x00byte", ErrControlChars},
		{"bell\x07char", ErrControlChars},
		{"escape\x1bseq", ErrControlChars},
		{"del\x7fchar", ErrControlChars},
		{"\x01start", ErrControlChars},
		{"end\x0d", ErrControlChars},
	}
	for _, tc := range cases {
		err := ValidateString(tc.input)
		if !errors.Is(err, tc.want) {
			t.Errorf("ValidateString(%q) = %v, want %v", tc.input, err, tc.want)
		}
	}
}

func TestRejectPathTraversal(t *testing.T) {
	cases := []struct {
		input string
		want  error
	}{
		{"normal-string", nil},
		{"path/to/file", nil},
		{"../../etc/passwd", ErrPathTraversal},
		{"foo/../bar", ErrPathTraversal},
		{`..\\windows\\system32`, ErrPathTraversal},
		{"a/b/../c", ErrPathTraversal},
		{`a\..\\b`, ErrPathTraversal},
	}
	for _, tc := range cases {
		err := ValidateString(tc.input)
		if !errors.Is(err, tc.want) {
			t.Errorf("ValidateString(%q) = %v, want %v", tc.input, err, tc.want)
		}
	}
}

func TestRejectDoubleEncoding(t *testing.T) {
	cases := []struct {
		input string
		want  error
	}{
		{"normal", nil},
		{"hello%20world", nil},                 // single encoding is fine
		{"hello%2520world", ErrDoubleEncoding}, // %25 = %, so %2520 = double-encoded space
		{"%252F", ErrDoubleEncoding},           // double-encoded /
		{"%2541", ErrDoubleEncoding},           // double-encoded A
	}
	for _, tc := range cases {
		err := ValidateString(tc.input)
		if !errors.Is(err, tc.want) {
			t.Errorf("ValidateString(%q) = %v, want %v", tc.input, err, tc.want)
		}
	}
}

func TestUUID(t *testing.T) {
	cases := []struct {
		input string
		want  error
	}{
		{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", nil},
		{"00000000-0000-0000-0000-000000000000", nil},
		{"ffffffff-ffff-ffff-ffff-ffffffffffff", nil},
		// invalid
		{"", ErrInvalidUUID},
		{"not-a-uuid", ErrInvalidUUID},
		{"A1B2C3D4-E5F6-7890-ABCD-EF1234567890", ErrInvalidUUID},  // uppercase
		{"a1b2c3d4e5f67890abcdef1234567890", ErrInvalidUUID},      // no dashes
		{"a1b2c3d4-e5f6-7890-abcd-ef123456789", ErrInvalidUUID},   // too short
		{"a1b2c3d4-e5f6-7890-abcd-ef12345678901", ErrInvalidUUID}, // too long
		{"g1b2c3d4-e5f6-7890-abcd-ef1234567890", ErrInvalidUUID},  // invalid char 'g'
	}
	for _, tc := range cases {
		err := UUID(tc.input)
		if !errors.Is(err, tc.want) {
			t.Errorf("UUID(%q) = %v, want %v", tc.input, err, tc.want)
		}
	}
}

func TestURL(t *testing.T) {
	cases := []struct {
		input string
		want  error
	}{
		{"https://example.com", nil},
		{"http://example.com/path?q=1", nil},
		{"https://sub.example.com:8080/path", nil},
		// invalid
		{"", ErrInvalidURL},
		{"ftp://example.com", ErrInvalidURL},
		{"just-a-string", ErrInvalidURL},
		{"://missing-scheme", ErrInvalidURL},
		{"https://", ErrInvalidURL},
	}
	for _, tc := range cases {
		err := URL(tc.input)
		if !errors.Is(err, tc.want) {
			t.Errorf("URL(%q) = %v, want %v", tc.input, err, tc.want)
		}
	}
}

func TestMetadataKey(t *testing.T) {
	// Valid key
	if err := MetadataKey("my-key"); err != nil {
		t.Errorf("MetadataKey(%q) = %v, want nil", "my-key", err)
	}

	// Too long
	long := strings.Repeat("a", MaxMetadataKeyLen+1)
	err := MetadataKey(long)
	if !errors.Is(err, ErrStringTooLong) {
		t.Errorf("MetadataKey(len=%d) = %v, want ErrStringTooLong", len(long), err)
	}

	// Exactly at limit
	exact := strings.Repeat("a", MaxMetadataKeyLen)
	if err := MetadataKey(exact); err != nil {
		t.Errorf("MetadataKey(len=%d) = %v, want nil", len(exact), err)
	}

	// Control chars
	err = MetadataKey("key\x00value")
	if !errors.Is(err, ErrControlChars) {
		t.Errorf("MetadataKey with null = %v, want ErrControlChars", err)
	}
}

func TestMetadataValue(t *testing.T) {
	// Valid value
	if err := MetadataValue("my-value"); err != nil {
		t.Errorf("MetadataValue(%q) = %v, want nil", "my-value", err)
	}

	// Too long
	long := strings.Repeat("a", MaxMetadataValueLen+1)
	err := MetadataValue(long)
	if !errors.Is(err, ErrStringTooLong) {
		t.Errorf("MetadataValue(len=%d) = %v, want ErrStringTooLong", len(long), err)
	}

	// Exactly at limit
	exact := strings.Repeat("a", MaxMetadataValueLen)
	if err := MetadataValue(exact); err != nil {
		t.Errorf("MetadataValue(len=%d) = %v, want nil", len(exact), err)
	}
}

func TestStringWithMaxLen(t *testing.T) {
	if err := StringWithMaxLen("short", 10); err != nil {
		t.Errorf("StringWithMaxLen(%q, 10) = %v, want nil", "short", err)
	}

	err := StringWithMaxLen("toolong", 3)
	if !errors.Is(err, ErrStringTooLong) {
		t.Errorf("StringWithMaxLen(%q, 3) = %v, want ErrStringTooLong", "toolong", err)
	}

	// Sanitization still applies
	err = StringWithMaxLen("has\x00null", 100)
	if !errors.Is(err, ErrControlChars) {
		t.Errorf("StringWithMaxLen with null = %v, want ErrControlChars", err)
	}
}
