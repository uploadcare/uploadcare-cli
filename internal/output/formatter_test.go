package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatOptions_Validate_OK(t *testing.T) {
	cases := []FormatOptions{
		{},
		{Verbose: true},
		{Quiet: true},
		{Verbose: true, JSON: true},
	}
	for _, opts := range cases {
		if err := opts.Validate(); err != nil {
			t.Errorf("Validate(%+v) = %v, want nil", opts, err)
		}
	}
}

func TestFormatOptions_Validate_VerboseQuietConflict(t *testing.T) {
	opts := FormatOptions{Verbose: true, Quiet: true}
	err := opts.Validate()
	if err == nil {
		t.Fatal("Validate(verbose+quiet) should return error")
	}
	if err != ErrVerboseQuietConflict {
		t.Errorf("err = %v, want ErrVerboseQuietConflict", err)
	}
}

func TestNew_Default_ReturnsTable(t *testing.T) {
	f := New(FormatOptions{})
	if _, ok := f.(*TableFormatter); !ok {
		t.Errorf("New(default) returned %T, want *TableFormatter", f)
	}
}

func TestNew_JSON_ReturnsJSON(t *testing.T) {
	f := New(FormatOptions{JSON: true})
	if _, ok := f.(*JSONFormatter); !ok {
		t.Errorf("New(JSON) returned %T, want *JSONFormatter", f)
	}
}

func TestNew_JQ_ReturnsJSON(t *testing.T) {
	f := New(FormatOptions{JQ: ".uuid"})
	if _, ok := f.(*JSONFormatter); !ok {
		t.Errorf("New(JQ) returned %T, want *JSONFormatter", f)
	}
}

func TestNew_Quiet_ReturnsQuiet(t *testing.T) {
	f := New(FormatOptions{Quiet: true})
	if _, ok := f.(*quietFormatter); !ok {
		t.Errorf("New(Quiet) returned %T, want *quietFormatter", f)
	}
}

func TestNew_JSONWithFields(t *testing.T) {
	f := New(FormatOptions{JSON: true, Fields: []string{"uuid", "size"}})
	jf, ok := f.(*JSONFormatter)
	if !ok {
		t.Fatalf("New(JSON+Fields) returned %T, want *JSONFormatter", f)
	}
	if len(jf.Fields) != 2 {
		t.Errorf("Fields len = %d, want 2", len(jf.Fields))
	}
}

func TestQuietFormatter_ProducesNoOutput(t *testing.T) {
	f := &quietFormatter{}
	var buf bytes.Buffer
	err := f.Format(&buf, map[string]any{"uuid": "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() > 0 {
		t.Errorf("quiet formatter produced output: %q", buf.String())
	}
}

func TestJSONFormatter_FullObject(t *testing.T) {
	f := &JSONFormatter{}
	var buf bytes.Buffer
	data := map[string]any{
		"uuid":     "abc-123",
		"size":     1024,
		"filename": "test.jpg",
	}

	err := f.Format(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["uuid"] != "abc-123" {
		t.Errorf("uuid = %v, want abc-123", result["uuid"])
	}
	if result["size"] != float64(1024) {
		t.Errorf("size = %v, want 1024", result["size"])
	}
	if result["filename"] != "test.jpg" {
		t.Errorf("filename = %v, want test.jpg", result["filename"])
	}
}

func TestJSONFormatter_WithFields(t *testing.T) {
	f := &JSONFormatter{Fields: []string{"uuid", "size"}}
	var buf bytes.Buffer
	data := map[string]any{
		"uuid":     "abc-123",
		"size":     1024,
		"filename": "test.jpg",
	}

	err := f.Format(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["uuid"] != "abc-123" {
		t.Errorf("uuid = %v, want abc-123", result["uuid"])
	}
	if result["size"] != float64(1024) {
		t.Errorf("size = %v, want 1024", result["size"])
	}
	if _, ok := result["filename"]; ok {
		t.Error("filename should be filtered out")
	}
}

func TestJSONFormatter_Array(t *testing.T) {
	f := &JSONFormatter{Fields: []string{"uuid"}}
	var buf bytes.Buffer
	data := []any{
		map[string]any{"uuid": "a", "size": 1},
		map[string]any{"uuid": "b", "size": 2},
	}

	err := f.Format(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0]["uuid"] != "a" {
		t.Errorf("result[0].uuid = %v, want a", result[0]["uuid"])
	}
	if _, ok := result[0]["size"]; ok {
		t.Error("size should be filtered out")
	}
}

func TestJSONFormatter_Struct(t *testing.T) {
	type file struct {
		UUID     string `json:"uuid"`
		Size     int    `json:"size"`
		Filename string `json:"filename"`
	}

	f := &JSONFormatter{Fields: []string{"uuid", "size"}}
	var buf bytes.Buffer
	data := file{UUID: "abc", Size: 1024, Filename: "test.jpg"}

	err := f.Format(&buf, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["uuid"] != "abc" {
		t.Errorf("uuid = %v, want abc", result["uuid"])
	}
	if _, ok := result["filename"]; ok {
		t.Error("filename should be filtered out")
	}
}

func TestNDJSONLine(t *testing.T) {
	var buf bytes.Buffer

	data1 := map[string]any{"uuid": "a", "size": 1, "filename": "one.jpg"}
	data2 := map[string]any{"uuid": "b", "size": 2, "filename": "two.jpg"}

	fields := []string{"uuid"}

	if err := NDJSONLine(&buf, data1, fields, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := NDJSONLine(&buf, data2, fields, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line %d: invalid JSON: %v", i, err)
		}
		if _, ok := m["uuid"]; !ok {
			t.Errorf("line %d: missing uuid", i)
		}
		if _, ok := m["filename"]; ok {
			t.Errorf("line %d: filename should be filtered", i)
		}
	}
}

func TestNDJSONLine_NoFields(t *testing.T) {
	var buf bytes.Buffer
	data := map[string]any{"uuid": "a", "size": 1}

	if err := NDJSONLine(&buf, data, nil, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 fields, got %d", len(result))
	}
}

func TestTableFormatter_Basic(t *testing.T) {
	f := &TableFormatter{}
	var buf bytes.Buffer

	td := NewTableData("UUID", "SIZE", "FILENAME")
	td.AddRow("abc-123", "1.2 MB", "photo.jpg")
	td.AddRow("def-456", "340 KB", "doc.pdf")

	err := f.Format(&buf, td)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "UUID") {
		t.Error("output missing header UUID")
	}
	if !strings.Contains(out, "photo.jpg") {
		t.Error("output missing photo.jpg")
	}
	if !strings.Contains(out, "doc.pdf") {
		t.Error("output missing doc.pdf")
	}

	// Check alignment: each line should be separated by at least 2 spaces
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (header + 2 rows), got %d", len(lines))
	}
}

func TestTableFormatter_EmptyTable(t *testing.T) {
	f := &TableFormatter{}
	var buf bytes.Buffer

	td := NewTableData("A", "B")

	err := f.Format(&buf, td)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if !strings.Contains(out, "A") {
		t.Error("output should contain header even with no rows")
	}
}

func TestTableFormatter_NonTableData(t *testing.T) {
	f := &TableFormatter{}
	var buf bytes.Buffer

	err := f.Format(&buf, "plain string output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "plain string output") {
		t.Error("output should contain the plain string")
	}
}

func TestParseJSONFlag(t *testing.T) {
	cases := []struct {
		raw        string
		wantEnable bool
		wantFields []string
	}{
		{"", false, nil},
		{"all", true, nil},
		{"uuid", true, []string{"uuid"}},
		{"uuid,size", true, []string{"uuid", "size"}},
		{"uuid, size, filename", true, []string{"uuid", "size", "filename"}},
	}
	for _, tc := range cases {
		enabled, fields := ParseJSONFlag(tc.raw)
		if enabled != tc.wantEnable {
			t.Errorf("ParseJSONFlag(%q) enabled = %v, want %v", tc.raw, enabled, tc.wantEnable)
		}
		if tc.wantFields == nil {
			if fields != nil {
				t.Errorf("ParseJSONFlag(%q) fields = %v, want nil", tc.raw, fields)
			}
		} else {
			if len(fields) != len(tc.wantFields) {
				t.Errorf("ParseJSONFlag(%q) fields = %v, want %v", tc.raw, fields, tc.wantFields)
				continue
			}
			for i := range fields {
				if fields[i] != tc.wantFields[i] {
					t.Errorf("ParseJSONFlag(%q) fields[%d] = %q, want %q", tc.raw, i, fields[i], tc.wantFields[i])
				}
			}
		}
	}
}

func TestParseFields(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"uuid", []string{"uuid"}},
		{"uuid,size", []string{"uuid", "size"}},
		{"uuid, size, filename", []string{"uuid", "size", "filename"}},
		{" uuid , size ", []string{"uuid", "size"}},
		{",,,", nil},
		{"uuid,,size", []string{"uuid", "size"}},
	}

	for _, tc := range cases {
		got := ParseFields(tc.input)
		if tc.want == nil {
			if got != nil {
				t.Errorf("ParseFields(%q) = %v, want nil", tc.input, got)
			}
			continue
		}
		if len(got) != len(tc.want) {
			t.Errorf("ParseFields(%q) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("ParseFields(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

func TestFilterFields_Map(t *testing.T) {
	m := map[string]any{
		"uuid":     "abc",
		"size":     1024,
		"filename": "test.jpg",
	}
	result := filterFields(m, []string{"uuid", "size"})
	rm, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if rm["uuid"] != "abc" {
		t.Errorf("uuid = %v", rm["uuid"])
	}
	if _, ok := rm["filename"]; ok {
		t.Error("filename should be filtered")
	}
}

func TestFilterFields_EmptyFields(t *testing.T) {
	m := map[string]any{"a": 1, "b": 2}
	result := filterFields(m, nil)
	// Should return original data unchanged.
	rm, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if len(rm) != 2 {
		t.Errorf("expected 2 fields, got %d", len(rm))
	}
}

func TestFilterFields_Slice(t *testing.T) {
	data := []any{
		map[string]any{"uuid": "a", "size": 1},
		map[string]any{"uuid": "b", "size": 2},
	}
	result := filterFields(data, []string{"uuid"})
	arr, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("item %d: expected map, got %T", i, item)
		}
		if _, ok := m["size"]; ok {
			t.Errorf("item %d: size should be filtered", i)
		}
	}
}
