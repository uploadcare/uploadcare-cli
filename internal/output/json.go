package output

import (
	"encoding/json"
	"io"
)

// JSONFormatter writes data as minified JSON to stdout.
// When Fields is non-empty, only those top-level keys are included.
type JSONFormatter struct {
	Fields []string
}

// Format writes data as JSON. If Fields is set, filters top-level keys.
func (f *JSONFormatter) Format(w io.Writer, data any) error {
	if len(f.Fields) > 0 {
		data = filterFields(data, f.Fields)
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}

// NDJSONLine writes a single object as one line of NDJSON.
// Fields filtering is applied if fields is non-empty.
func NDJSONLine(w io.Writer, data any, fields []string) error {
	if len(fields) > 0 {
		data = filterFields(data, fields)
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}
