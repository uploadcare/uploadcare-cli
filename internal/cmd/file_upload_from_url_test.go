package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/uploadcare/uploadcare-cli/internal/service"
)

func TestFileUploadFromURL_Single(t *testing.T) {
	var capturedParams service.URLUploadParams

	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			capturedParams = params
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "upload-from-url", "https://example.com/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedParams.URL != "https://example.com/image.jpg" {
		t.Errorf("URL = %q, want %q", capturedParams.URL, "https://example.com/image.jpg")
	}
	if capturedParams.Store != "auto" {
		t.Errorf("Store = %q, want %q", capturedParams.Store, "auto")
	}

	if !strings.Contains(stdout, "a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
		t.Errorf("output missing UUID\ngot:\n%s", stdout)
	}
	if !strings.Contains(stdout, "photo.jpg") {
		t.Errorf("output missing filename\ngot:\n%s", stdout)
	}
}

func TestFileUploadFromURL_Multiple(t *testing.T) {
	callCount := 0
	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			callCount++
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "upload-from-url",
		"https://example.com/a.jpg", "https://example.com/b.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}

	if !strings.Contains(stdout, "UUID") {
		t.Errorf("output should contain summary table header\ngot:\n%s", stdout)
	}
}

func TestFileUploadFromURL_JSON(t *testing.T) {
	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "upload-from-url", "https://example.com/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot: %s", err, stdout)
	}
	if result["uuid"] != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("uuid = %v", result["uuid"])
	}
}

func TestFileUploadFromURL_StoreTrue(t *testing.T) {
	var capturedParams service.URLUploadParams

	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			capturedParams = params
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload-from-url", "--store", "true", "https://example.com/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedParams.Store != "true" {
		t.Errorf("Store = %q, want %q", capturedParams.Store, "true")
	}
}

func TestFileUploadFromURL_Metadata(t *testing.T) {
	var capturedParams service.URLUploadParams

	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			capturedParams = params
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload-from-url", "--metadata", "key=val", "https://example.com/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedParams.Metadata == nil || capturedParams.Metadata["key"] != "val" {
		t.Errorf("Metadata = %v, want map[key:val]", capturedParams.Metadata)
	}
}

func TestFileUploadFromURL_CheckDuplicates(t *testing.T) {
	var capturedParams service.URLUploadParams

	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			capturedParams = params
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload-from-url", "--check-duplicates", "https://example.com/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !capturedParams.CheckDuplicates {
		t.Error("CheckDuplicates should be true")
	}
}

func TestFileUploadFromURL_SaveDuplicates(t *testing.T) {
	var capturedParams service.URLUploadParams

	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			capturedParams = params
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload-from-url", "--save-duplicates", "https://example.com/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !capturedParams.SaveDuplicates {
		t.Error("SaveDuplicates should be true")
	}
}

func TestFileUploadFromURL_Timeout(t *testing.T) {
	var capturedParams service.URLUploadParams

	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			capturedParams = params
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload-from-url", "--timeout", "10s", "https://example.com/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedParams.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", capturedParams.Timeout)
	}
}

func TestFileUploadFromURL_DryRun(t *testing.T) {
	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			t.Fatal("service should not be called in dry-run mode")
			return nil, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "upload-from-url", "--dry-run", "https://example.com/image.jpg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "https://example.com/image.jpg") {
		t.Errorf("output should contain the URL\ngot:\n%s", stdout)
	}
	if !strings.Contains(stdout, "ok") {
		t.Errorf("output should contain status 'ok'\ngot:\n%s", stdout)
	}
}

func TestFileUploadFromURL_DryRunInvalidURL(t *testing.T) {
	mock := &mockFileService{}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload-from-url", "--dry-run", "not-a-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestFileUploadFromURL_NoURLs(t *testing.T) {
	mock := &mockFileService{}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload-from-url")
	if err == nil {
		t.Fatal("expected error for no URLs")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 2 {
		t.Errorf("exit code = %d, want 2", exitErr.Code)
	}
}

func TestFileUploadFromURL_InvalidURL(t *testing.T) {
	mock := &mockFileService{}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload-from-url", "ftp://example.com/file")
	if err == nil {
		t.Fatal("expected error for invalid URL scheme")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 2 {
		t.Errorf("exit code = %d, want 2", exitErr.Code)
	}
}

func TestFileUploadFromURL_InvalidStoreValue(t *testing.T) {
	mock := &mockFileService{}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload-from-url", "--store", "invalid", "https://example.com/image.jpg")
	if err == nil {
		t.Fatal("expected error for invalid store value")
	}
}

func TestFileUploadFromURL_ServiceError(t *testing.T) {
	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			return nil, errors.New("upload failed")
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "upload-from-url", "https://example.com/image.jpg")
	if err == nil {
		t.Fatal("expected error from service")
	}
	if !strings.Contains(err.Error(), "upload failed") {
		t.Errorf("error should propagate service error, got: %v", err)
	}
}

func TestFileUploadFromURL_FromStdin(t *testing.T) {
	callCount := 0
	mock := &mockFileService{
		uploadFromURLFunc: func(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
			callCount++
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	root.SetIn(strings.NewReader("https://example.com/a.jpg\nhttps://example.com/b.jpg\n"))
	stdout, _, err := executeCommand(t, root, "file", "upload-from-url", "--from-stdin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}

	if !strings.Contains(stdout, "UUID") {
		t.Errorf("output should contain summary table\ngot:\n%s", stdout)
	}
}
