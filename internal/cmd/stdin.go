package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ReadLinesOrNDJSON reads lines from r and returns extracted values.
// It auto-detects NDJSON (first non-empty line starts with '{') vs plain text.
// In NDJSON mode, it extracts the value of fieldName from each JSON object.
// In plain mode, it returns trimmed non-empty lines.
func ReadLinesOrNDJSON(r io.Reader, fieldName string) ([]string, error) {
	scanner := bufio.NewScanner(r)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading input: %w", err)
	}

	if len(lines) == 0 {
		return nil, nil
	}

	// Auto-detect NDJSON by checking if first line starts with '{'
	if strings.HasPrefix(lines[0], "{") {
		return parseNDJSON(lines, fieldName)
	}

	return lines, nil
}

func parseNDJSON(lines []string, fieldName string) ([]string, error) {
	var values []string
	for i, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			return nil, fmt.Errorf("line %d: invalid JSON: %w", i+1, err)
		}
		val, ok := obj[fieldName]
		if !ok {
			return nil, fmt.Errorf("line %d: missing field %q", i+1, fieldName)
		}
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("line %d: field %q is not a string", i+1, fieldName)
		}
		values = append(values, s)
	}
	return values, nil
}
