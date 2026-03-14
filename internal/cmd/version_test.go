package cmd

import (
	"bytes"
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

func TestRootCmd_HasGlobalFlags(t *testing.T) {
	root := NewRootCmd("dev", "none", "unknown")

	flags := []string{
		"public-key", "secret-key", "token", "project",
		"json", "jq", "quiet", "no-color",
		"rest-api-base", "upload-api-base", "cdn-base", "project-api-base",
	}

	for _, name := range flags {
		if root.PersistentFlags().Lookup(name) == nil {
			t.Errorf("missing global flag: --%s", name)
		}
	}
}
