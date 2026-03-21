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

type mockAddonService struct {
	executeFunc func(ctx context.Context, addonName, fileUUID string, params json.RawMessage) (*service.AddonResult, error)
	statusFunc  func(ctx context.Context, addonName, requestID string) (*service.AddonStatus, error)
}

func (m *mockAddonService) Execute(ctx context.Context, addonName, fileUUID string, params json.RawMessage) (*service.AddonResult, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, addonName, fileUUID, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAddonService) Status(ctx context.Context, addonName, requestID string) (*service.AddonStatus, error) {
	if m.statusFunc != nil {
		return m.statusFunc(ctx, addonName, requestID)
	}
	return nil, errors.New("not implemented")
}

func newTestRootWithAddon(mock service.AddonService) *cobra.Command {
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

	root.AddCommand(newAddonCmd(mock))
	return root
}

func TestAddonExecute_NoWait_Human(t *testing.T) {
	var capturedAddonName, capturedFileUUID string
	mock := &mockAddonService{
		executeFunc: func(ctx context.Context, addonName, fileUUID string, params json.RawMessage) (*service.AddonResult, error) {
			capturedAddonName = addonName
			capturedFileUUID = fileUUID
			return &service.AddonResult{
				RequestID: "req-123",
				Status:    "in_progress",
			}, nil
		},
	}

	root := newTestRootWithAddon(mock)
	stdout, _, err := executeCommand(t, root, "addon", "execute", "remove-bg", testUUID, "--no-wait")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedAddonName != "remove_bg" {
		t.Errorf("addon name = %q, want remove_bg", capturedAddonName)
	}
	if capturedFileUUID != testUUID {
		t.Errorf("file UUID = %q, want %s", capturedFileUUID, testUUID)
	}
	if !strings.Contains(stdout, "Request ID: req-123") {
		t.Errorf("output missing request ID\ngot: %s", stdout)
	}
}

func TestAddonExecute_NoWait_JSON(t *testing.T) {
	mock := &mockAddonService{
		executeFunc: func(ctx context.Context, addonName, fileUUID string, params json.RawMessage) (*service.AddonResult, error) {
			return &service.AddonResult{RequestID: "req-123", Status: "in_progress"}, nil
		},
	}

	root := newTestRootWithAddon(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "addon", "execute", "remove-bg", testUUID, "--no-wait")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if result["request_id"] != "req-123" {
		t.Errorf("request_id = %v", result["request_id"])
	}
}

func TestAddonExecute_Wait(t *testing.T) {
	mock := &mockAddonService{
		executeFunc: func(ctx context.Context, addonName, fileUUID string, params json.RawMessage) (*service.AddonResult, error) {
			return &service.AddonResult{RequestID: "req-123", Status: "in_progress"}, nil
		},
		statusFunc: func(ctx context.Context, addonName, requestID string) (*service.AddonStatus, error) {
			return &service.AddonStatus{
				Status: "done",
				Result: json.RawMessage(`{"labels":["cat","animal"]}`),
			}, nil
		},
	}

	root := newTestRootWithAddon(mock)
	stdout, _, err := executeCommand(t, root, "addon", "execute", "aws-rekognition-detect-labels", testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Status: done") {
		t.Errorf("output missing done status\ngot: %s", stdout)
	}
}

func TestAddonExecute_WithParams(t *testing.T) {
	var capturedParams json.RawMessage
	mock := &mockAddonService{
		executeFunc: func(ctx context.Context, addonName, fileUUID string, params json.RawMessage) (*service.AddonResult, error) {
			capturedParams = params
			return &service.AddonResult{RequestID: "req-123"}, nil
		},
	}

	root := newTestRootWithAddon(mock)
	_, _, err := executeCommand(t, root, "addon", "execute", "remove-bg", testUUID, "--no-wait", "--params", `{"crop":true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(capturedParams) != `{"crop":true}` {
		t.Errorf("params = %s, want {\"crop\":true}", string(capturedParams))
	}
}

func TestAddonExecute_InvalidAddonName(t *testing.T) {
	mock := &mockAddonService{}
	root := newTestRootWithAddon(mock)
	_, _, err := executeCommand(t, root, "addon", "execute", "unknown-addon", testUUID)
	if err == nil {
		t.Fatal("expected error for invalid addon name")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}

func TestAddonExecute_InvalidUUID(t *testing.T) {
	mock := &mockAddonService{}
	root := newTestRootWithAddon(mock)
	_, _, err := executeCommand(t, root, "addon", "execute", "remove-bg", "bad-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}

func TestAddonExecute_InvalidParams(t *testing.T) {
	mock := &mockAddonService{}
	root := newTestRootWithAddon(mock)
	_, _, err := executeCommand(t, root, "addon", "execute", "remove-bg", testUUID, "--no-wait", "--params", "not-json")
	if err == nil {
		t.Fatal("expected error for invalid params JSON")
	}
}

func TestAddonExecute_DryRun(t *testing.T) {
	mock := &mockAddonService{}
	root := newTestRootWithAddon(mock)
	stdout, _, err := executeCommand(t, root, "addon", "execute", "remove-bg", testUUID, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would execute") {
		t.Errorf("dry-run output missing 'Would execute'\ngot: %s", stdout)
	}
}

func TestAddonStatus_Human(t *testing.T) {
	mock := &mockAddonService{
		statusFunc: func(ctx context.Context, addonName, requestID string) (*service.AddonStatus, error) {
			return &service.AddonStatus{
				Status: "done",
				Result: json.RawMessage(`{"labels":["cat"]}`),
			}, nil
		},
	}

	root := newTestRootWithAddon(mock)
	stdout, _, err := executeCommand(t, root, "addon", "status", "remove-bg", "req-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Status: done") {
		t.Errorf("output missing status\ngot: %s", stdout)
	}
	if !strings.Contains(stdout, "Result:") {
		t.Errorf("output missing result\ngot: %s", stdout)
	}
}

func TestAddonStatus_JSON(t *testing.T) {
	mock := &mockAddonService{
		statusFunc: func(ctx context.Context, addonName, requestID string) (*service.AddonStatus, error) {
			return &service.AddonStatus{
				Status: "done",
				Result: json.RawMessage(`{"labels":["cat"]}`),
			}, nil
		},
	}

	root := newTestRootWithAddon(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "addon", "status", "remove-bg", "req-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if result["status"] != "done" {
		t.Errorf("status = %v", result["status"])
	}
}

func TestAddonStatus_InvalidAddonName(t *testing.T) {
	mock := &mockAddonService{}
	root := newTestRootWithAddon(mock)
	_, _, err := executeCommand(t, root, "addon", "status", "bad-addon", "req-123")
	if err == nil {
		t.Fatal("expected error for invalid addon name")
	}
}

func TestAddonExecute_SDKNameAccepted(t *testing.T) {
	mock := &mockAddonService{
		executeFunc: func(ctx context.Context, addonName, fileUUID string, params json.RawMessage) (*service.AddonResult, error) {
			if addonName != "remove_bg" {
				t.Errorf("addon name = %q, want remove_bg", addonName)
			}
			return &service.AddonResult{RequestID: "req-123"}, nil
		},
	}

	root := newTestRootWithAddon(mock)
	_, _, err := executeCommand(t, root, "addon", "execute", "remove_bg", testUUID, "--no-wait")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
