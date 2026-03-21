package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	root := NewRootCmd("v0.1.0", "abc1234", "2026-03-08")

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()

	expected := []string{
		"uploadcare-cli v0.1.0",
		"commit: abc1234",
		"built:  2026-03-08",
		"go:     go",
		"os/arch:",
	}

	for _, s := range expected {
		if !strings.Contains(out, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, out)
		}
	}
}

func TestRootCmd_JSONFlagNoArg(t *testing.T) {
	root := NewRootCmd("dev", "none", "unknown")
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"--json", "version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("--json without argument should not error, got: %v", err)
	}

	val, err := root.PersistentFlags().GetString("json")
	if err != nil {
		t.Fatalf("GetString(json): %v", err)
	}
	if val != "true" {
		t.Errorf("--json without arg should be %q, got %q", "true", val)
	}
}

func TestRootCmd_JSONFlagWithFields(t *testing.T) {
	root := NewRootCmd("dev", "none", "unknown")
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"--json=uuid,size", "version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("--json=uuid,size should not error, got: %v", err)
	}

	val, err := root.PersistentFlags().GetString("json")
	if err != nil {
		t.Fatalf("GetString(json): %v", err)
	}
	if val != "uuid,size" {
		t.Errorf("--json=uuid,size should be %q, got %q", "uuid,size", val)
	}
}

func TestRootCmd_VerboseQuietConflict(t *testing.T) {
	root := NewRootCmd("dev", "none", "unknown")
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"--verbose", "--quiet", "version"})

	err := root.Execute()
	if err == nil {
		t.Fatal("--verbose --quiet should produce an error")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error should mention mutual exclusivity, got: %v", err)
	}
}

func TestRootCmd_JQImpliesJSON(t *testing.T) {
	root := NewRootCmd("dev", "none", "unknown")

	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"--jq", ".version", "version"})

	if err := root.Execute(); err != nil {
		t.Fatalf("--jq should not error, got: %v", err)
	}

	// The output should be JSON (jq applied), not the human-readable table
	out := strings.TrimSpace(buf.String())
	// jq '.version' on the version JSON should produce a quoted string
	if !json.Valid([]byte(out)) && out != "" {
		// At minimum, it should not contain the human "uploadcare-cli" header
		if strings.Contains(out, "uploadcare-cli") {
			t.Errorf("--jq without --json should still produce JSON output, got:\n%s", out)
		}
	}

	// Also verify via formatOptionsFromCmd that JSON is set
	opts := formatOptionsFromCmd(root)
	if !opts.JSON {
		t.Error("formatOptionsFromCmd should set JSON=true when --jq is specified")
	}
	if opts.JQ != ".version" {
		t.Errorf("formatOptionsFromCmd JQ = %q, want %q", opts.JQ, ".version")
	}
}

func TestRootCmd_HasGlobalFlags(t *testing.T) {
	root := NewRootCmd("dev", "none", "unknown")

	flags := []string{
		"public-key", "secret-key", "project-api-token", "project",
		"json", "jq", "quiet", "verbose", "no-color",
		"rest-api-base", "upload-api-base", "cdn-base", "project-api-base",
	}

	for _, name := range flags {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("missing global flag: --%s", name)
		}
	}
}
