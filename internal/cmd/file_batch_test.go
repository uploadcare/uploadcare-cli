package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

func TestFileStore_Human(t *testing.T) {
	mock := &mockFileService{
		storeFunc: func(ctx context.Context, uuids []string) (*service.BatchResult, error) {
			return &service.BatchResult{
				Files: []service.File{*testFile()},
			}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "store", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
		t.Errorf("output missing UUID\ngot:\n%s", stdout)
	}
	if !strings.Contains(stdout, "ok") {
		t.Errorf("output missing status\ngot:\n%s", stdout)
	}
}

func TestFileStore_JSON(t *testing.T) {
	mock := &mockFileService{
		storeFunc: func(ctx context.Context, uuids []string) (*service.BatchResult, error) {
			return &service.BatchResult{
				Files: []service.File{*testFile()},
			}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "store", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot: %s", err, stdout)
	}
	results, ok := result["results"].([]interface{})
	if !ok || len(results) != 1 {
		t.Fatalf("expected 1 result, got: %v", result["results"])
	}
}

func TestFileStore_NoUUIDs(t *testing.T) {
	mock := &mockFileService{}
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "store")
	if err == nil {
		t.Fatal("expected error for no UUIDs")
	}
}

func TestFileStore_InvalidUUID(t *testing.T) {
	mock := &mockFileService{}
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "store", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestFileStore_DryRun(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "store", "--dry-run", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "would store") {
		t.Errorf("dry-run output missing 'would store'\ngot:\n%s", stdout)
	}
}

func TestFileStore_WithProblems(t *testing.T) {
	mock := &mockFileService{
		storeFunc: func(ctx context.Context, uuids []string) (*service.BatchResult, error) {
			return &service.BatchResult{
				Problems: map[string]string{
					"a1b2c3d4-e5f6-7890-abcd-ef1234567890": "file not found",
				},
			}, nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "store", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err == nil {
		t.Fatal("expected error when there are problems")
	}
}

func TestFileStore_WithProblems_JSON(t *testing.T) {
	mock := &mockFileService{
		storeFunc: func(ctx context.Context, uuids []string) (*service.BatchResult, error) {
			return &service.BatchResult{
				Problems: map[string]string{
					"a1b2c3d4-e5f6-7890-abcd-ef1234567890": "file not found",
				},
			}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "store", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err == nil {
		t.Fatal("expected error when there are problems in JSON mode")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	// JSON output should still be written before the error
	if !strings.Contains(stdout, "problems") {
		t.Errorf("JSON output should contain problems\ngot:\n%s", stdout)
	}
}

func TestFileStore_DryRun_NotFound_JSON(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return nil, ucare.APIError{StatusCode: 404, Detail: "not found"}
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "store", "--dry-run", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err == nil {
		t.Fatal("expected error when dry-run finds missing files in JSON mode")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 1 {
		t.Errorf("expected exit code 1, got: %v", err)
	}
	// JSON output should still be written before the error
	if !strings.Contains(stdout, "not found") {
		t.Errorf("JSON output should contain status\ngot:\n%s", stdout)
	}
}

func TestFileStore_DryRun_APIError_Propagates(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return nil, ucare.APIError{StatusCode: 401, Detail: "unauthorized"}
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "store", "--dry-run", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err == nil {
		t.Fatal("expected error for non-404 API error")
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("error should propagate API error detail, got: %v", err)
	}
	// Should NOT be an exit code 1 "some files not found" error
	var exitErr *ExitError
	if errors.As(err, &exitErr) && exitErr.Code == 1 {
		t.Error("non-404 errors should not produce 'some files not found' exit")
	}
}

func TestFileStore_DryRun_NonAPIError_Propagates(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return nil, errors.New("connection refused")
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "store", "--dry-run", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err == nil {
		t.Fatal("expected error for network failure")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error should propagate underlying error, got: %v", err)
	}
}

func TestFileDelete_Human(t *testing.T) {
	mock := &mockFileService{
		deleteFunc: func(ctx context.Context, uuids []string) (*service.BatchResult, error) {
			return &service.BatchResult{
				Files: []service.File{*testFile()},
			}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "delete", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
		t.Errorf("output missing UUID\ngot:\n%s", stdout)
	}
}

func TestFileDelete_DryRun(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "delete", "--dry-run", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "would delete") {
		t.Errorf("dry-run output missing 'would delete'\ngot:\n%s", stdout)
	}
}
