package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/uploadcare/uploadcare-cli/internal/service"
)

func TestFileUpload_Human(t *testing.T) {
	tmpFile := createTestFile(t, "test.txt", "hello world")

	mock := &mockFileService{
		uploadFunc: func(ctx context.Context, params service.UploadParams) (*service.File, error) {
			if params.Name != "test.txt" {
				t.Errorf("name = %q, want %q", params.Name, "test.txt")
			}
			if params.Size != 11 {
				t.Errorf("size = %d, want 11", params.Size)
			}
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "upload", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
		t.Errorf("output missing UUID\ngot:\n%s", stdout)
	}
}

func TestFileUpload_JSON(t *testing.T) {
	tmpFile := createTestFile(t, "test.txt", "hello world")

	mock := &mockFileService{
		uploadFunc: func(ctx context.Context, params service.UploadParams) (*service.File, error) {
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "upload", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if result["uuid"] != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("uuid = %v", result["uuid"])
	}
}

func TestFileUpload_MultipleFiles(t *testing.T) {
	tmpFile1 := createTestFile(t, "a.txt", "aaa")
	tmpFile2 := createTestFile(t, "b.txt", "bbb")

	var uploadCount int
	mock := &mockFileService{
		uploadFunc: func(ctx context.Context, params service.UploadParams) (*service.File, error) {
			uploadCount++
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "upload", tmpFile1, tmpFile2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if uploadCount != 2 {
		t.Errorf("expected 2 uploads, got %d", uploadCount)
	}
	if !strings.Contains(stdout, "UUID") {
		t.Errorf("output missing table header\ngot:\n%s", stdout)
	}
}

func TestFileUpload_DryRun(t *testing.T) {
	tmpFile := createTestFile(t, "test.txt", "hello world")

	mock := &mockFileService{}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "upload", "--dry-run", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "test.txt") {
		t.Errorf("dry-run output missing filename\ngot:\n%s", stdout)
	}
	if !strings.Contains(stdout, "11") {
		t.Errorf("dry-run output missing size\ngot:\n%s", stdout)
	}
}

func TestFileUpload_NoFiles(t *testing.T) {
	mock := &mockFileService{}
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload")
	if err == nil {
		t.Fatal("expected error for no files")
	}
}

func TestFileUpload_NonexistentFile(t *testing.T) {
	mock := &mockFileService{}
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload", "/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestFileUpload_MutuallyExclusiveFlags(t *testing.T) {
	tmpFile := createTestFile(t, "test.txt", "hello")

	mock := &mockFileService{}
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload",
		"--force-multipart", "--force-direct", tmpFile)
	if err == nil {
		t.Fatal("expected error for mutually exclusive flags")
	}
}

func TestFileUpload_StoreFlag(t *testing.T) {
	tmpFile := createTestFile(t, "test.txt", "hello")

	var capturedStore string
	mock := &mockFileService{
		uploadFunc: func(ctx context.Context, params service.UploadParams) (*service.File, error) {
			capturedStore = params.Store
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload", "--store", "true", tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStore != "true" {
		t.Errorf("store = %q, want %q", capturedStore, "true")
	}
}

func TestFileUpload_StoreFlag_Invalid(t *testing.T) {
	tmpFile := createTestFile(t, "test.txt", "hello")

	mock := &mockFileService{
		uploadFunc: func(ctx context.Context, params service.UploadParams) (*service.File, error) {
			t.Fatal("upload should not be called with invalid --store value")
			return nil, nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload", "--store", "ture", tmpFile)
	if err == nil {
		t.Fatal("expected error for invalid --store value")
	}
	if !strings.Contains(err.Error(), "invalid --store value") {
		t.Errorf("error should mention invalid store value, got: %v", err)
	}
}

func TestFileUpload_MetadataFlag(t *testing.T) {
	tmpFile := createTestFile(t, "test.txt", "hello")

	var capturedMeta map[string]string
	mock := &mockFileService{
		uploadFunc: func(ctx context.Context, params service.UploadParams) (*service.File, error) {
			capturedMeta = params.Metadata
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload",
		"--metadata", "key1=val1",
		"--metadata", "key2=val2",
		tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedMeta["key1"] != "val1" || capturedMeta["key2"] != "val2" {
		t.Errorf("metadata = %v", capturedMeta)
	}
}

func TestParseMetadata(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    map[string]string
		wantErr bool
	}{
		{"empty", nil, nil, false},
		{"single", []string{"k=v"}, map[string]string{"k": "v"}, false},
		{"multiple", []string{"a=1", "b=2"}, map[string]string{"a": "1", "b": "2"}, false},
		{"value with equals", []string{"k=a=b"}, map[string]string{"k": "a=b"}, false},
		{"no equals", []string{"invalid"}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMetadata(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				for k, v := range tt.want {
					if got[k] != v {
						t.Errorf("got[%q] = %q, want %q", k, got[k], v)
					}
				}
			}
		})
	}
}

func TestBaseName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/path/to/file.txt", "file.txt"},
		{"file.txt", "file.txt"},
		{"/file.txt", "file.txt"},
		{"path\\to\\file.txt", "file.txt"},
	}
	for _, tt := range tests {
		got := baseName(tt.input)
		if got != tt.want {
			t.Errorf("baseName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func createTestFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
