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
	infoFunc       func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error)
	listFunc       func(ctx context.Context, opts service.FileListOptions) (*service.FileListResult, error)
	iterateFunc    func(ctx context.Context, opts service.FileListOptions, fn func(service.File) error) error
	uploadFunc        func(ctx context.Context, params service.UploadParams) (*service.File, error)
	uploadFromURLFunc func(ctx context.Context, params service.URLUploadParams) (*service.File, error)
	storeFunc         func(ctx context.Context, uuids []string) (*service.BatchResult, error)
	deleteFunc     func(ctx context.Context, uuids []string) (*service.BatchResult, error)
	localCopyFunc  func(ctx context.Context, params service.LocalCopyParams) (*service.File, error)
	remoteCopyFunc func(ctx context.Context, params service.RemoteCopyParams) (*service.RemoteCopyResult, error)
}

func (m *mockFileService) Info(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
	if m.infoFunc != nil {
		return m.infoFunc(ctx, uuid, includeAppData)
	}
	return nil, errors.New("not implemented")
}

func (m *mockFileService) List(ctx context.Context, opts service.FileListOptions) (*service.FileListResult, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, opts)
	}
	return nil, errors.New("not implemented")
}

func (m *mockFileService) Iterate(ctx context.Context, opts service.FileListOptions, fn func(service.File) error) error {
	if m.iterateFunc != nil {
		return m.iterateFunc(ctx, opts, fn)
	}
	return errors.New("not implemented")
}

func (m *mockFileService) Upload(ctx context.Context, params service.UploadParams) (*service.File, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockFileService) UploadFromURL(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
	if m.uploadFromURLFunc != nil {
		return m.uploadFromURLFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockFileService) Store(ctx context.Context, uuids []string) (*service.BatchResult, error) {
	if m.storeFunc != nil {
		return m.storeFunc(ctx, uuids)
	}
	return nil, errors.New("not implemented")
}

func (m *mockFileService) Delete(ctx context.Context, uuids []string) (*service.BatchResult, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, uuids)
	}
	return nil, errors.New("not implemented")
}

func (m *mockFileService) LocalCopy(ctx context.Context, params service.LocalCopyParams) (*service.File, error) {
	if m.localCopyFunc != nil {
		return m.localCopyFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockFileService) RemoteCopy(ctx context.Context, params service.RemoteCopyParams) (*service.RemoteCopyResult, error) {
	if m.remoteCopyFunc != nil {
		return m.remoteCopyFunc(ctx, params)
	}
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
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "info", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
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

// --- File list tests ---

func TestFileList_Human(t *testing.T) {
	mock := &mockFileService{
		listFunc: func(ctx context.Context, opts service.FileListOptions) (*service.FileListResult, error) {
			return &service.FileListResult{
				Files: []service.File{*testFile()},
				Total: 1,
			}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"UUID", "SIZE", "FILENAME", "STORED", "UPLOADED", "a1b2c3d4-e5f6-7890-abcd-ef1234567890", "photo.jpg"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestFileList_JSON(t *testing.T) {
	mock := &mockFileService{
		listFunc: func(ctx context.Context, opts service.FileListOptions) (*service.FileListResult, error) {
			return &service.FileListResult{
				Files: []service.File{*testFile()},
				Total: 1,
			}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON array: %v\ngot: %s", err, stdout)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0]["uuid"] != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("uuid = %v", result[0]["uuid"])
	}
}

func TestFileList_PageAll(t *testing.T) {
	files := []service.File{*testFile()}
	mock := &mockFileService{
		iterateFunc: func(ctx context.Context, opts service.FileListOptions, fn func(service.File) error) error {
			for _, f := range files {
				if err := fn(f); err != nil {
					return err
				}
			}
			return nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "list", "--page-all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(lines))
	}

	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &obj); err != nil {
		t.Fatalf("invalid NDJSON: %v", err)
	}
	if obj["uuid"] != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("uuid = %v", obj["uuid"])
	}
}

func TestFileList_PageAll_Quiet(t *testing.T) {
	var iterated bool
	mock := &mockFileService{
		iterateFunc: func(ctx context.Context, opts service.FileListOptions, fn func(service.File) error) error {
			iterated = true
			return fn(*testFile())
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--quiet", "file", "list", "--page-all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !iterated {
		t.Error("iterate should still be called")
	}
	if stdout != "" {
		t.Errorf("--quiet should suppress all output, got:\n%s", stdout)
	}
}

func TestFileList_Options(t *testing.T) {
	var capturedOpts service.FileListOptions

	mock := &mockFileService{
		listFunc: func(ctx context.Context, opts service.FileListOptions) (*service.FileListResult, error) {
			capturedOpts = opts
			return &service.FileListResult{}, nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "list",
		"--ordering", "-datetime_uploaded",
		"--limit", "50",
		"--stored", "true",
		"--removed",
		"--include-appdata",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedOpts.Ordering != "-datetime_uploaded" {
		t.Errorf("ordering = %q, want %q", capturedOpts.Ordering, "-datetime_uploaded")
	}
	if capturedOpts.Limit != 50 {
		t.Errorf("limit = %d, want 50", capturedOpts.Limit)
	}
	if capturedOpts.Stored == nil || *capturedOpts.Stored != true {
		t.Error("stored should be true")
	}
	if !capturedOpts.Removed {
		t.Error("removed should be true")
	}
	if !capturedOpts.IncludeAppData {
		t.Error("include appdata should be true")
	}
}

func testFileWithAppData() *service.File {
	f := testFile()
	f.AppData = json.RawMessage(`{"uc_clamav_virus_scan":{"data":{"infected":false},"version":"1.0.0"}}`)
	return f
}

func TestFileInfo_TableWithAppData(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return testFileWithAppData(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "info", "--include-appdata", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "AppData:") {
		t.Errorf("output should contain AppData section\ngot:\n%s", stdout)
	}
	if !strings.Contains(stdout, "uc_clamav_virus_scan") {
		t.Errorf("output should contain appdata content\ngot:\n%s", stdout)
	}
	if !strings.Contains(stdout, `"infected": false`) {
		t.Errorf("appdata should be pretty-printed with indentation\ngot:\n%s", stdout)
	}
}

func TestFileInfo_TableWithAppData_Quiet(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return testFileWithAppData(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--quiet", "file", "info", "--include-appdata", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(stdout, "AppData:") {
		t.Errorf("--quiet should suppress AppData section\ngot:\n%s", stdout)
	}
}

func TestFileInfo_TableWithoutAppData(t *testing.T) {
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

	if strings.Contains(stdout, "AppData:") {
		t.Errorf("output should not contain AppData section when flag is not set\ngot:\n%s", stdout)
	}
}

func TestFileList_TableWithAppData(t *testing.T) {
	f := testFileWithAppData()
	mock := &mockFileService{
		listFunc: func(ctx context.Context, opts service.FileListOptions) (*service.FileListResult, error) {
			return &service.FileListResult{
				Files: []service.File{*f},
				Total: 1,
			}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "list", "--include-appdata")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "APPDATA") {
		t.Errorf("output should contain APPDATA header\ngot:\n%s", stdout)
	}
	if !strings.Contains(stdout, "uc_clamav_virus_scan") {
		t.Errorf("output should contain appdata content\ngot:\n%s", stdout)
	}
}

func TestFileList_TableWithoutAppData(t *testing.T) {
	mock := &mockFileService{
		listFunc: func(ctx context.Context, opts service.FileListOptions) (*service.FileListResult, error) {
			return &service.FileListResult{
				Files: []service.File{*testFile()},
				Total: 1,
			}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(stdout, "APPDATA") {
		t.Errorf("output should not contain APPDATA column when flag is not set\ngot:\n%s", stdout)
	}
}

func TestFileList_PageAll_TableWithAppData(t *testing.T) {
	f := testFileWithAppData()
	mock := &mockFileService{
		iterateFunc: func(ctx context.Context, opts service.FileListOptions, fn func(service.File) error) error {
			return fn(*f)
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "list", "--page-all", "--include-appdata")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "uc_clamav_virus_scan") {
		t.Errorf("streaming output should contain appdata\ngot:\n%s", stdout)
	}
}

func TestTruncateAppData(t *testing.T) {
	tests := []struct {
		name   string
		data   json.RawMessage
		maxLen int
		want   string
	}{
		{"empty", nil, 50, ""},
		{"short", json.RawMessage(`{"ok":true}`), 50, `{"ok":true}`},
		{"exact", json.RawMessage(`12345`), 5, `12345`},
		{"over", json.RawMessage(`{"uc_clamav_virus_scan":{"data":{"infected":false}}}`), 20, `{"uc_clamav_virus_sc...`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateAppData(tt.data, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateAppData(%q, %d) = %q, want %q", string(tt.data), tt.maxLen, got, tt.want)
			}
		})
	}
}
