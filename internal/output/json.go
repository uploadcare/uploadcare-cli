package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/itchyny/gojq"
)

// JSONFormatter writes data as minified JSON to stdout.
// When Fields is non-empty, only those top-level keys are included.
// When JQ is non-empty, the jq expression is applied after field filtering.
type JSONFormatter struct {
	Fields []string
	JQ     string
}

// Format writes data as JSON. If Fields is set, filters top-level keys.
// If JQ is set, applies the jq expression to the output.
func (f *JSONFormatter) Format(w io.Writer, data any) error {
	if len(f.Fields) > 0 {
		data = filterFields(data, f.Fields)
	}

	if f.JQ != "" {
		return applyJQ(w, data, f.JQ)
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

// NDJSONLine writes a single object as one line of NDJSON.
// Fields filtering is applied if fields is non-empty.
// If jq is non-empty, the jq expression is applied.
func NDJSONLine(w io.Writer, data any, fields []string, jq string) error {
	if len(fields) > 0 {
		data = filterFields(data, fields)
	}

	if jq != "" {
		return applyJQ(w, data, jq)
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

// applyJQ applies a jq expression to the given data and writes the results.
func applyJQ(w io.Writer, data any, expr string) error {
	query, err := gojq.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid jq expression: %w", err)
	}

	// Convert data to interface{} that gojq can process
	// by round-tripping through JSON
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var input any
	if err := json.Unmarshal(raw, &input); err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	iter := query.Run(input)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			return fmt.Errorf("jq error: %w", err)
		}

		// For strings, output raw (unquoted) text
		if s, ok := v.(string); ok {
			_, _ = fmt.Fprintln(w, s)
			continue
		}

		if err := enc.Encode(v); err != nil {
			return err
		}
	}

	return nil
}
