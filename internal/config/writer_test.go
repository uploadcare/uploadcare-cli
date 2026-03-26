package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// overrideConfigDir temporarily overrides the config directory for tests.
// Returns a cleanup function that restores the original.
func overrideConfigDir(t *testing.T, dir string) {
	t.Helper()

	// Override the ConfigDir/ConfigPath functions by setting $HOME.
	// ConfigDir() uses os.UserHomeDir() which reads $HOME.
	t.Setenv("HOME", dir)

	// Create the .uploadcare directory.
	if err := os.MkdirAll(filepath.Join(dir, ".uploadcare"), 0o700); err != nil {
		t.Fatal(err)
	}
}

func TestSaveProjectEntry_EmptyConfig(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDir(t, dir)

	if err := SaveProjectEntry("My App", "pub123", "sec456"); err != nil {
		t.Fatalf("SaveProjectEntry failed: %v", err)
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "My App") {
		t.Errorf("config missing project name\ngot:\n%s", content)
	}
	if !strings.Contains(content, "pub123") {
		t.Errorf("config missing public key\ngot:\n%s", content)
	}
	if !strings.Contains(content, "sec456") {
		t.Errorf("config missing secret key\ngot:\n%s", content)
	}
}

func TestSaveProjectEntry_PreservesExistingKeys(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDir(t, dir)

	// Write existing config.
	existing := "project_api_token: bearer-token-123\n"
	if err := os.WriteFile(ConfigPath(), []byte(existing), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := SaveProjectEntry("My App", "pub123", "sec456"); err != nil {
		t.Fatalf("SaveProjectEntry failed: %v", err)
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "bearer-token-123") {
		t.Errorf("existing key was lost\ngot:\n%s", content)
	}
	if !strings.Contains(content, "pub123") {
		t.Errorf("new project missing\ngot:\n%s", content)
	}
}

func TestRemoveProjectEntry(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDir(t, dir)

	// Seed with a project.
	if err := SaveProjectEntry("My App", "pub123", "sec456"); err != nil {
		t.Fatal(err)
	}

	if err := RemoveProjectEntry("My App"); err != nil {
		t.Fatalf("RemoveProjectEntry failed: %v", err)
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if strings.Contains(content, "pub123") {
		t.Errorf("project should have been removed\ngot:\n%s", content)
	}
}

func TestRemoveProjectEntry_ClearsDefaultProject(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDir(t, dir)

	if err := SaveProjectEntry("My App", "pub123", "sec456"); err != nil {
		t.Fatal(err)
	}
	if err := SetDefaultProject("My App"); err != nil {
		t.Fatal(err)
	}

	if err := RemoveProjectEntry("My App"); err != nil {
		t.Fatalf("RemoveProjectEntry failed: %v", err)
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if strings.Contains(content, "default_project") {
		t.Errorf("default_project should have been cleared\ngot:\n%s", content)
	}
}

func TestRemoveProjectEntry_NotFound(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDir(t, dir)

	// Should not error on missing project.
	if err := RemoveProjectEntry("nonexistent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRemoveProjectByPubKey(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDir(t, dir)

	if err := SaveProjectEntry("My App", "pub123", "sec456"); err != nil {
		t.Fatal(err)
	}
	if err := SetDefaultProject("My App"); err != nil {
		t.Fatal(err)
	}

	// Remove by pub_key, not by name.
	if err := RemoveProjectByPubKey("pub123"); err != nil {
		t.Fatalf("RemoveProjectByPubKey failed: %v", err)
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if strings.Contains(content, "pub123") {
		t.Errorf("project should have been removed by pub_key\ngot:\n%s", content)
	}
	if strings.Contains(content, "default_project") {
		t.Errorf("default_project should have been cleared\ngot:\n%s", content)
	}
}

func TestRemoveProjectByPubKey_NotFound(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDir(t, dir)

	if err := RemoveProjectByPubKey("nonexistent"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetDefaultProject(t *testing.T) {
	dir := t.TempDir()
	overrideConfigDir(t, dir)

	if err := SetDefaultProject("My App"); err != nil {
		t.Fatalf("SetDefaultProject failed: %v", err)
	}

	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "default_project") || !strings.Contains(content, "My App") {
		t.Errorf("default_project not set\ngot:\n%s", content)
	}
}
