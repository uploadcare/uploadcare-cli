package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/service"
)

type mockConvertService struct {
	documentFunc       func(ctx context.Context, params service.DocConvertParams) (*service.ConvertResult, error)
	documentStatusFunc func(ctx context.Context, token string) (*service.ConvertStatus, error)
	videoFunc          func(ctx context.Context, params service.VideoConvertParams) (*service.ConvertResult, error)
	videoStatusFunc    func(ctx context.Context, token string) (*service.ConvertStatus, error)
}

func (m *mockConvertService) Document(ctx context.Context, params service.DocConvertParams) (*service.ConvertResult, error) {
	if m.documentFunc != nil {
		return m.documentFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockConvertService) DocumentStatus(ctx context.Context, token string) (*service.ConvertStatus, error) {
	if m.documentStatusFunc != nil {
		return m.documentStatusFunc(ctx, token)
	}
	return nil, errors.New("not implemented")
}

func (m *mockConvertService) Video(ctx context.Context, params service.VideoConvertParams) (*service.ConvertResult, error) {
	if m.videoFunc != nil {
		return m.videoFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockConvertService) VideoStatus(ctx context.Context, token string) (*service.ConvertStatus, error) {
	if m.videoStatusFunc != nil {
		return m.videoStatusFunc(ctx, token)
	}
	return nil, errors.New("not implemented")
}

func newTestRootWithConvert(mock service.ConvertService) *cobra.Command {
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

	root.AddCommand(newConvertCmd(mock))
	return root
}

func TestConvertDocument_NoWait_Human(t *testing.T) {
	mock := &mockConvertService{
		documentFunc: func(ctx context.Context, params service.DocConvertParams) (*service.ConvertResult, error) {
			if params.Format != "pdf" {
				t.Errorf("format = %q, want pdf", params.Format)
			}
			return &service.ConvertResult{
				Token:  "12345",
				UUID:   "b1b2c3d4-e5f6-7890-abcd-ef1234567890",
				Status: "pending",
			}, nil
		},
	}

	root := newTestRootWithConvert(mock)
	stdout, _, err := executeCommand(t, root, "convert", "document", testUUID, "--format", "pdf", "--no-wait")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Token: 12345") {
		t.Errorf("output missing token\ngot: %s", stdout)
	}
}

func TestConvertDocument_NoWait_JSON(t *testing.T) {
	mock := &mockConvertService{
		documentFunc: func(ctx context.Context, params service.DocConvertParams) (*service.ConvertResult, error) {
			return &service.ConvertResult{
				Token:  "12345",
				UUID:   "b1b2c3d4-e5f6-7890-abcd-ef1234567890",
				Status: "pending",
			}, nil
		},
	}

	root := newTestRootWithConvert(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "convert", "document", testUUID, "--format", "pdf", "--no-wait")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if result["token"] != "12345" {
		t.Errorf("token = %v", result["token"])
	}
}

func TestConvertDocument_Wait(t *testing.T) {
	callCount := 0
	mock := &mockConvertService{
		documentFunc: func(ctx context.Context, params service.DocConvertParams) (*service.ConvertResult, error) {
			return &service.ConvertResult{Token: "12345", UUID: "b1b2c3d4-e5f6-7890-abcd-ef1234567890"}, nil
		},
		documentStatusFunc: func(ctx context.Context, token string) (*service.ConvertStatus, error) {
			callCount++
			return &service.ConvertStatus{
				Status:    "finished",
				ResultURL: "c1b2c3d4-e5f6-7890-abcd-ef1234567890",
			}, nil
		},
	}

	root := newTestRootWithConvert(mock)
	stdout, _, err := executeCommand(t, root, "convert", "document", testUUID, "--format", "pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Status: finished") {
		t.Errorf("output missing finished status\ngot: %s", stdout)
	}
}

func TestConvertDocument_MissingFormat(t *testing.T) {
	mock := &mockConvertService{}
	root := newTestRootWithConvert(mock)
	_, _, err := executeCommand(t, root, "convert", "document", testUUID)
	if err == nil {
		t.Fatal("expected error for missing --format")
	}
}

func TestConvertDocument_InvalidUUID(t *testing.T) {
	mock := &mockConvertService{}
	root := newTestRootWithConvert(mock)
	_, _, err := executeCommand(t, root, "convert", "document", "bad-uuid", "--format", "pdf")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}

func TestConvertDocument_DryRun(t *testing.T) {
	mock := &mockConvertService{}
	root := newTestRootWithConvert(mock)
	stdout, _, err := executeCommand(t, root, "convert", "document", testUUID, "--format", "pdf", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would convert") {
		t.Errorf("dry-run output missing 'Would convert'\ngot: %s", stdout)
	}
}

func TestConvertDocument_WithPage(t *testing.T) {
	var capturedParams service.DocConvertParams
	mock := &mockConvertService{
		documentFunc: func(ctx context.Context, params service.DocConvertParams) (*service.ConvertResult, error) {
			capturedParams = params
			return &service.ConvertResult{Token: "12345", UUID: "b1b2c3d4-e5f6-7890-abcd-ef1234567890"}, nil
		},
	}

	root := newTestRootWithConvert(mock)
	_, _, err := executeCommand(t, root, "convert", "document", testUUID, "--format", "pdf", "--page", "3", "--no-wait")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedParams.Page == nil || *capturedParams.Page != 3 {
		t.Errorf("page should be 3, got %v", capturedParams.Page)
	}
}

func TestConvertVideo_NoWait_Human(t *testing.T) {
	mock := &mockConvertService{
		videoFunc: func(ctx context.Context, params service.VideoConvertParams) (*service.ConvertResult, error) {
			return &service.ConvertResult{
				Token:  "67890",
				UUID:   "b1b2c3d4-e5f6-7890-abcd-ef1234567890",
				Status: "pending",
			}, nil
		},
	}

	root := newTestRootWithConvert(mock)
	stdout, _, err := executeCommand(t, root, "convert", "video", testUUID, "--format", "mp4", "--no-wait")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Token: 67890") {
		t.Errorf("output missing token\ngot: %s", stdout)
	}
}

func TestConvertVideo_DryRun(t *testing.T) {
	mock := &mockConvertService{}
	root := newTestRootWithConvert(mock)
	stdout, _, err := executeCommand(t, root, "convert", "video", testUUID, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would convert") {
		t.Errorf("dry-run output missing 'Would convert'\ngot: %s", stdout)
	}
}

func TestConvertVideo_Wait(t *testing.T) {
	mock := &mockConvertService{
		videoFunc: func(ctx context.Context, params service.VideoConvertParams) (*service.ConvertResult, error) {
			return &service.ConvertResult{Token: "67890", UUID: "b1b2c3d4-e5f6-7890-abcd-ef1234567890"}, nil
		},
		videoStatusFunc: func(ctx context.Context, token string) (*service.ConvertStatus, error) {
			return &service.ConvertStatus{Status: "finished", ResultURL: "c1b2c3d4-e5f6-7890-abcd-ef1234567890"}, nil
		},
	}

	root := newTestRootWithConvert(mock)
	stdout, _, err := executeCommand(t, root, "convert", "video", testUUID, "--format", "mp4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Status: finished") {
		t.Errorf("output missing finished status\ngot: %s", stdout)
	}
}
