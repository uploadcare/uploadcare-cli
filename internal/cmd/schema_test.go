package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestAPISchema_ContainsAgentNotes(t *testing.T) {
	root := NewRootCmd("v0.1.0", "abc1234", "2026-03-08")

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"api-schema"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var schema struct {
		AgentNotes []string `json:"agent_notes"`
	}
	if err := json.Unmarshal(buf.Bytes(), &schema); err != nil {
		t.Fatalf("failed to parse schema JSON: %v", err)
	}

	if len(schema.AgentNotes) == 0 {
		t.Fatal("agent_notes should not be empty")
	}

	// Verify the --json= syntax warning is present (highest value note)
	found := false
	for _, note := range schema.AgentNotes {
		if containsSubstring(note, "--json=") {
			found = true
			break
		}
	}
	if !found {
		t.Error("agent_notes should contain the --json= syntax warning")
	}
}

func TestAPISchema_CommandsHaveJSONFields(t *testing.T) {
	root := NewRootCmd("v0.1.0", "abc1234", "2026-03-08")

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"api-schema"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var schema struct {
		Commands []struct {
			Path       string   `json:"path"`
			JSONFields []string `json:"json_fields"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(buf.Bytes(), &schema); err != nil {
		t.Fatalf("failed to parse schema JSON: %v", err)
	}

	// Commands that must have json_fields
	expectFields := map[string][]string{
		"file info":   {"uuid", "filename", "size"},
		"file list":   {"uuid", "filename", "size"},
		"file upload": {"uuid", "filename"},
		"file store":  {"results", "problems"},
		"version":     {"version", "commit"},
	}

	cmdMap := make(map[string][]string)
	for _, cmd := range schema.Commands {
		if cmd.JSONFields != nil {
			cmdMap[cmd.Path] = cmd.JSONFields
		}
	}

	for path, required := range expectFields {
		fields, ok := cmdMap[path]
		if !ok {
			t.Errorf("command %q should have json_fields", path)
			continue
		}
		for _, f := range required {
			if !containsString(fields, f) {
				t.Errorf("command %q json_fields missing %q, got %v", path, f, fields)
			}
		}
	}
}

func TestJSONFieldsForCommand(t *testing.T) {
	tests := []struct {
		path       string
		wantNil    bool
		wantFields []string // subset that must be present
	}{
		{"file info", false, []string{"uuid", "size", "filename", "original_file_url"}},
		{"file list", false, []string{"uuid", "size"}},
		{"file store", false, []string{"results", "problems"}},
		{"file delete", false, []string{"results", "problems"}},
		{"file remote-copy", false, []string{"type", "result", "already_exists"}},
		{"group info", false, []string{"id", "cdn_url", "files"}},
		{"webhook list", false, []string{"id", "target_url", "event"}},
		{"version", false, []string{"version", "commit", "go_version"}},
		{"addon execute", false, []string{"status", "result"}},
		{"convert document", false, []string{"token", "uuid", "status"}},
		{"unknown-command", true, nil},
		{"", true, nil},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			got := jsonFieldsForCommand(tc.path)
			if tc.wantNil {
				if got != nil {
					t.Errorf("jsonFieldsForCommand(%q) = %v, want nil", tc.path, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("jsonFieldsForCommand(%q) = nil, want fields", tc.path)
			}
			for _, f := range tc.wantFields {
				if !containsString(got, f) {
					t.Errorf("jsonFieldsForCommand(%q) missing %q, got %v", tc.path, f, got)
				}
			}
		})
	}
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && bytes.Contains([]byte(s), []byte(substr))
}
