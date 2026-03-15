package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/service"
)

// mockFileService implements service.FileService for testing.
type mockFileService struct {
	infoFunc func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error)
}

func (m *mockFileService) Info(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
	return m.infoFunc(ctx, uuid, includeAppData)
}

func (m *mockFileService) List(ctx context.Context, opts service.FileListOptions) (*service.FileListResult, error) {
	return nil, errors.New("not implemented")
}

func (m *mockFileService) Upload(ctx context.Context, params service.UploadParams) (*service.File, error) {
	return nil, errors.New("not implemented")
}

func (m *mockFileService) UploadFromURL(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
	return nil, errors.New("not implemented")
}

func (m *mockFileService) Store(ctx context.Context, uuids []string) ([]service.File, error) {
	return nil, errors.New("not implemented")
}

func (m *mockFileService) Delete(ctx context.Context, uuids []string) ([]service.File, error) {
	return nil, errors.New("not implemented")
}

func (m *mockFileService) LocalCopy(ctx context.Context, params service.LocalCopyParams) (*service.File, error) {
	return nil, errors.New("not implemented")
}

func (m *mockFileService) RemoteCopy(ctx context.Context, params service.RemoteCopyParams) (*service.RemoteCopyResult, error) {
	return nil, errors.New("not implemented")
}

func testFile() *service.File {
	uploaded := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	stored := time.Date(2026, 3, 1, 10, 0, 1, 0, time.UTC)
	return &service.File{
		UUID:             "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		Filename:         "photo.jpg",
		Size:             1258000,
		MimeType:         "image/jpeg",
		IsImage:          true,
		IsStored:         true,
		IsReady:          true,
		DatetimeUploaded: uploaded,
		DatetimeStored:   &stored,
		URL:              "https://api.uploadcare.com/files/a1b2c3d4-e5f6-7890-abcd-ef1234567890/",
		OriginalFileURL:  "https://ucarecdn.com/a1b2c3d4-e5f6-7890-abcd-ef1234567890/",
		Metadata:         map[string]string{"key": "value"},
	}
}

// newTestRoot creates a root command with a mock file service wired in,
// bypassing config loader initialization.
func newTestRoot(mock service.FileService) *cobra.Command {
	root := &cobra.Command{
		Use:           "uploadcare",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	flags := root.PersistentFlags()
	flags.String("json", "", "Output as JSON")
	flags.Lookup("json").NoOptDefVal = "true"
	flags.String("jq", "", "jq expression")
	flags.BoolP("quiet", "q", false, "Suppress output")
	flags.BoolP("verbose", "v", false, "Verbose output")

	root.AddCommand(newFileCmd(mock))
	return root
}

// executeCommand sets up buffers, executes a cobra command, and captures output.
func executeCommand(t *testing.T, cmd *cobra.Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(outBuf)
	cmd.SetErr(errBuf)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestFileInfo_Human(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "info", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"UUID:",
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"Filename:",
		"photo.jpg",
		"Size:",
		"1258000",
		"MIME Type:",
		"image/jpeg",
		"Image:",
		"true",
		"Stored:",
		"true",
		"Ready:",
		"true",
		"Uploaded:",
		"2026-03-01T10:00:00Z",
		"URL:",
		"https://ucarecdn.com/a1b2c3d4-e5f6-7890-abcd-ef1234567890/",
	}

	for _, s := range expected {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestFileInfo_JSON(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "file", "info", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot: %s", err, stdout)
	}

	if result["uuid"] != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("uuid = %v, want a1b2c3d4-e5f6-7890-abcd-ef1234567890", result["uuid"])
	}
	if result["filename"] != "photo.jpg" {
		t.Errorf("filename = %v, want photo.jpg", result["filename"])
	}
	if result["size"] != float64(1258000) {
		t.Errorf("size = %v, want 1258000", result["size"])
	}
}

func TestFileInfo_JSONFields(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json=uuid,size", "file", "info", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot: %s", err, stdout)
	}

	if _, ok := result["uuid"]; !ok {
		t.Error("filtered output should contain 'uuid'")
	}
	if _, ok := result["size"]; !ok {
		t.Error("filtered output should contain 'size'")
	}
	if _, ok := result["filename"]; ok {
		t.Error("filtered output should not contain 'filename'")
	}
}

func TestFileInfo_MissingUUID(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			t.Fatal("service should not be called without UUID")
			return nil, nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "info")
	if err == nil {
		t.Fatal("expected error for missing UUID argument")
	}
}

func TestFileInfo_InvalidUUID(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			t.Fatal("service should not be called with invalid UUID")
			return nil, nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "info", "not-a-valid-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}

	var exitErr *ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 2 {
		t.Errorf("exit code = %d, want 2", exitErr.Code)
	}
	if !strings.Contains(err.Error(), "invalid UUID") {
		t.Errorf("error should mention UUID validation, got: %v", err)
	}
}

func TestFileInfo_ServiceError(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return nil, errors.New("file not found")
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "info", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err == nil {
		t.Fatal("expected error from service")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("error should propagate service error, got: %v", err)
	}
}

func TestFileInfo_IncludeAppdata(t *testing.T) {
	var capturedIncludeAppData bool

	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			capturedIncludeAppData = includeAppData
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "info", "--include-appdata", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !capturedIncludeAppData {
		t.Error("includeAppData should be true when --include-appdata flag is set")
	}
}
