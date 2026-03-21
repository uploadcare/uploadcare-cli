package output

import (
	"encoding/json"
	"strings"
)

// ParseJSONFlag parses the raw --json flag value into a boolean and field list.
//
//   - "" → JSON disabled, nil fields
//   - "all" → JSON enabled, nil fields (all fields)
//   - "uuid,size" → JSON enabled, ["uuid","size"]
func ParseJSONFlag(raw string) (enabled bool, fields []string) {
	if raw == "" {
		return false, nil
	}
	if raw == "all" {
		return true, nil
	}
	return true, ParseFields(raw)
}

// ParseFields parses a comma-separated field list (e.g. "uuid,size,filename").
// Returns nil if the input is empty.
func ParseFields(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	fields := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			fields = append(fields, p)
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return fields
}

// filterFields filters data to include only the specified top-level fields.
// Works on single objects (map) and slices of objects.
func filterFields(data any, fields []string) any {
	if len(fields) == 0 {
		return data
	}

	// Build a set for fast lookup.
	allowed := make(map[string]bool, len(fields))
	for _, f := range fields {
		allowed[f] = true
	}

	switch v := data.(type) {
	case map[string]any:
		return filterMap(v, allowed)
	case []any:
		result := make([]any, len(v))
		for i, item := range v {
			if m, ok := item.(map[string]any); ok {
				result[i] = filterMap(m, allowed)
			} else {
				result[i] = item
			}
		}
		return result
	default:
		// Marshal to JSON and back to get a map, then filter.
		return filterViaJSON(data, allowed)
	}
}

// filterMap returns a new map containing only keys in allowed.
func filterMap(m map[string]any, allowed map[string]bool) map[string]any {
	result := make(map[string]any, len(allowed))
	for k, v := range m {
		if allowed[k] {
			result[k] = v
		}
	}
	return result
}

// filterViaJSON converts a struct to map[string]any via JSON round-trip,
// then applies field filtering.
func filterViaJSON(data any, allowed map[string]bool) any {
	b, err := json.Marshal(data)
	if err != nil {
		return data
	}

	// Try as a single object first.
	var m map[string]any
	if err := json.Unmarshal(b, &m); err == nil {
		return filterMap(m, allowed)
	}

	// Try as an array of objects.
	var arr []any
	if err := json.Unmarshal(b, &arr); err == nil {
		result := make([]any, len(arr))
		for i, item := range arr {
			if im, ok := item.(map[string]any); ok {
				result[i] = filterMap(im, allowed)
			} else {
				result[i] = item
			}
		}
		return result
	}

	return data
}
