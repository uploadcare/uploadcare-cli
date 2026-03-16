package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/uploadcare/uploadcare-cli/internal/service"
)

func TestFileLocalCopy_Human(t *testing.T) {
	mock := &mockFileService{
		localCopyFunc: func(ctx context.Context, params service.LocalCopyParams) (*service.File, error) {
			if params.UUID != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
				t.Errorf("unexpected UUID: %s", params.UUID)
			}
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "local-copy", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "a1b2c3d4-e5f6-7890-abcd-ef1234567890") {
		t.Errorf("output missing UUID\ngot:\n%s", stdout)
	}
}

func TestFileLocalCopy_JSON(t *testing.T) {
	mock := &mockFileService{
		localCopyFunc: func(ctx context.Context, params service.LocalCopyParams) (*service.File, error) {
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "file", "local-copy", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
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

func TestFileLocalCopy_StoreFlag(t *testing.T) {
	var capturedStore bool
	mock := &mockFileService{
		localCopyFunc: func(ctx context.Context, params service.LocalCopyParams) (*service.File, error) {
			capturedStore = params.Store
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "local-copy", "--store", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !capturedStore {
		t.Error("store should be true")
	}
}

func TestFileLocalCopy_DryRun(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "local-copy", "--dry-run", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "would copy locally") {
		t.Errorf("dry-run output missing status\ngot:\n%s", stdout)
	}
}

func TestFileLocalCopy_InvalidUUID(t *testing.T) {
	mock := &mockFileService{}
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "local-copy", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected exit code 2, got: %v", err)
	}
}

func TestFileRemoteCopy_Human(t *testing.T) {
	mock := &mockFileService{
		remoteCopyFunc: func(ctx context.Context, params service.RemoteCopyParams) (*service.RemoteCopyResult, error) {
			if params.Target != "my-storage" {
				t.Errorf("target = %q, want %q", params.Target, "my-storage")
			}
			return &service.RemoteCopyResult{
				Result:        "s3://my-bucket/file.jpg",
				AlreadyExists: false,
			}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "remote-copy",
		"--target", "my-storage",
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "s3://my-bucket/file.jpg") {
		t.Errorf("output missing result URL\ngot:\n%s", stdout)
	}
}

func TestFileRemoteCopy_JSON(t *testing.T) {
	mock := &mockFileService{
		remoteCopyFunc: func(ctx context.Context, params service.RemoteCopyParams) (*service.RemoteCopyResult, error) {
			return &service.RemoteCopyResult{
				Result:        "s3://bucket/file.jpg",
				AlreadyExists: true,
			}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "file", "remote-copy",
		"--target", "my-storage",
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if result["already_exists"] != true {
		t.Errorf("already_exists = %v, want true", result["already_exists"])
	}
}

func TestFileRemoteCopy_MissingTarget(t *testing.T) {
	mock := &mockFileService{}
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "remote-copy", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err == nil {
		t.Fatal("expected error for missing target")
	}
}

func TestFileRemoteCopy_DryRun(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
			return testFile(), nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "remote-copy",
		"--target", "my-storage",
		"--dry-run",
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "would copy to remote storage") {
		t.Errorf("dry-run output missing status\ngot:\n%s", stdout)
	}
}

func TestFileRemoteCopy_AllFlags(t *testing.T) {
	var capturedParams service.RemoteCopyParams
	mock := &mockFileService{
		remoteCopyFunc: func(ctx context.Context, params service.RemoteCopyParams) (*service.RemoteCopyResult, error) {
			capturedParams = params
			return &service.RemoteCopyResult{Result: "ok"}, nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "remote-copy",
		"--target", "storage-name",
		"--make-public",
		"--pattern", "${uuid}/${filename}",
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedParams.Target != "storage-name" {
		t.Errorf("target = %q", capturedParams.Target)
	}
	if !capturedParams.MakePublic {
		t.Error("make-public should be true")
	}
	if capturedParams.Pattern != "${uuid}/${filename}" {
		t.Errorf("pattern = %q", capturedParams.Pattern)
	}
}

func TestFileRemoteCopy_ServiceError(t *testing.T) {
	mock := &mockFileService{
		remoteCopyFunc: func(ctx context.Context, params service.RemoteCopyParams) (*service.RemoteCopyResult, error) {
			return nil, errors.New("remote storage not found")
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "remote-copy",
		"--target", "my-storage",
		"a1b2c3d4-e5f6-7890-abcd-ef1234567890",
	)
	if err == nil {
		t.Fatal("expected error from service")
	}
	if !strings.Contains(err.Error(), "remote storage not found") {
		t.Errorf("error = %v", err)
	}
}
