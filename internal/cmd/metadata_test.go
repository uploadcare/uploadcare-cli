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

type mockMetadataService struct {
	listFunc   func(ctx context.Context, fileUUID string) (map[string]string, error)
	getFunc    func(ctx context.Context, fileUUID, key string) (string, error)
	setFunc    func(ctx context.Context, fileUUID, key, value string) error
	deleteFunc func(ctx context.Context, fileUUID, key string) error
}

func (m *mockMetadataService) List(ctx context.Context, fileUUID string) (map[string]string, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, fileUUID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockMetadataService) Get(ctx context.Context, fileUUID, key string) (string, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, fileUUID, key)
	}
	return "", errors.New("not implemented")
}

func (m *mockMetadataService) Set(ctx context.Context, fileUUID, key, value string) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, fileUUID, key, value)
	}
	return errors.New("not implemented")
}

func (m *mockMetadataService) Delete(ctx context.Context, fileUUID, key string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, fileUUID, key)
	}
	return errors.New("not implemented")
}

func newTestRootWithMetadata(mock service.MetadataService) *cobra.Command {
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

	root.AddCommand(newMetadataCmd(mock))
	return root
}

const testUUID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"

func TestMetadataList_Human(t *testing.T) {
	mock := &mockMetadataService{
		listFunc: func(ctx context.Context, fileUUID string) (map[string]string, error) {
			return map[string]string{"env": "production", "team": "backend"}, nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "metadata", "list", testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"KEY", "VALUE"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestMetadataList_JSON(t *testing.T) {
	mock := &mockMetadataService{
		listFunc: func(ctx context.Context, fileUUID string) (map[string]string, error) {
			return map[string]string{"env": "production"}, nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "--json", "metadata", "list", testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot: %s", err, stdout)
	}
	if result["env"] != "production" {
		t.Errorf("env = %v, want production", result["env"])
	}
}

func TestMetadataList_InvalidUUID(t *testing.T) {
	mock := &mockMetadataService{}
	root := newTestRootWithMetadata(mock)
	_, _, err := executeCommand(t, root, "metadata", "list", "bad-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}

func TestMetadataGet_Human(t *testing.T) {
	mock := &mockMetadataService{
		getFunc: func(ctx context.Context, fileUUID, key string) (string, error) {
			return "production", nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "metadata", "get", testUUID, "env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "production") {
		t.Errorf("output missing 'production'\ngot: %s", stdout)
	}
}

func TestMetadataGet_JSON(t *testing.T) {
	mock := &mockMetadataService{
		getFunc: func(ctx context.Context, fileUUID, key string) (string, error) {
			return "production", nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "--json", "metadata", "get", testUUID, "env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["key"] != "env" || result["value"] != "production" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestMetadataGet_JSONFields(t *testing.T) {
	mock := &mockMetadataService{
		getFunc: func(ctx context.Context, fileUUID, key string) (string, error) {
			return "production", nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "--json=value", "metadata", "get", testUUID, "env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := result["value"]; !ok {
		t.Error("filtered output should contain 'value'")
	}
	if _, ok := result["key"]; ok {
		t.Error("filtered output should not contain 'key'")
	}
}

func TestMetadataSet_Human(t *testing.T) {
	var capturedKey, capturedValue string
	mock := &mockMetadataService{
		setFunc: func(ctx context.Context, fileUUID, key, value string) error {
			capturedKey = key
			capturedValue = value
			return nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "metadata", "set", testUUID, "env", "staging")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedKey != "env" || capturedValue != "staging" {
		t.Errorf("expected set(env, staging), got set(%s, %s)", capturedKey, capturedValue)
	}
	if !strings.Contains(stdout, "Set env") {
		t.Errorf("output missing confirmation\ngot: %s", stdout)
	}
}

func TestMetadataSet_JSON(t *testing.T) {
	mock := &mockMetadataService{
		setFunc: func(ctx context.Context, fileUUID, key, value string) error {
			return nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "--json", "metadata", "set", testUUID, "env", "staging")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["key"] != "env" || result["value"] != "staging" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestMetadataSet_DryRun(t *testing.T) {
	mock := &mockMetadataService{
		getFunc: func(ctx context.Context, fileUUID, key string) (string, error) {
			return "production", nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "metadata", "set", "--dry-run", testUUID, "env", "staging")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would set") {
		t.Errorf("dry-run output missing 'Would set'\ngot: %s", stdout)
	}
}

func TestMetadataDelete_Human(t *testing.T) {
	var capturedKey string
	mock := &mockMetadataService{
		deleteFunc: func(ctx context.Context, fileUUID, key string) error {
			capturedKey = key
			return nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "metadata", "delete", testUUID, "env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedKey != "env" {
		t.Errorf("expected delete(env), got delete(%s)", capturedKey)
	}
	if !strings.Contains(stdout, "Deleted env") {
		t.Errorf("output missing confirmation\ngot: %s", stdout)
	}
}

func TestMetadataDelete_JSON(t *testing.T) {
	mock := &mockMetadataService{
		deleteFunc: func(ctx context.Context, fileUUID, key string) error {
			return nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "--json", "metadata", "delete", testUUID, "env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["key"] != "env" || result["status"] != "deleted" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestMetadataDelete_DryRun(t *testing.T) {
	mock := &mockMetadataService{
		getFunc: func(ctx context.Context, fileUUID, key string) (string, error) {
			return "production", nil
		},
	}

	root := newTestRootWithMetadata(mock)
	stdout, _, err := executeCommand(t, root, "metadata", "delete", "--dry-run", testUUID, "env")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would delete") {
		t.Errorf("dry-run output missing 'Would delete'\ngot: %s", stdout)
	}
}

func TestMetadataDelete_DryRun_KeyNotFound(t *testing.T) {
	mock := &mockMetadataService{
		getFunc: func(ctx context.Context, fileUUID, key string) (string, error) {
			return "", errors.New("not found")
		},
	}

	root := newTestRootWithMetadata(mock)
	_, _, err := executeCommand(t, root, "metadata", "delete", "--dry-run", testUUID, "env")
	if err == nil {
		t.Fatal("expected error when key not found in dry-run")
	}
}

func TestMetadataSet_ValidationError(t *testing.T) {
	mock := &mockMetadataService{}
	root := newTestRootWithMetadata(mock)
	_, _, err := executeCommand(t, root, "metadata", "set", "bad-uuid", "key", "value")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}
