package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/service"
)

type mockWebhookService struct {
	listFunc   func(ctx context.Context) ([]service.Webhook, error)
	createFunc func(ctx context.Context, params service.WebhookCreateParams) (*service.Webhook, error)
	updateFunc func(ctx context.Context, id string, params service.WebhookUpdateParams) (*service.Webhook, error)
	deleteFunc func(ctx context.Context, id string) error
}

func (m *mockWebhookService) List(ctx context.Context) ([]service.Webhook, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockWebhookService) Create(ctx context.Context, params service.WebhookCreateParams) (*service.Webhook, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockWebhookService) Update(ctx context.Context, id string, params service.WebhookUpdateParams) (*service.Webhook, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockWebhookService) Delete(ctx context.Context, id string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return errors.New("not implemented")
}

func newTestRootWithWebhook(mock service.WebhookService) *cobra.Command {
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

	root.AddCommand(newWebhookCmd(mock))
	return root
}

func testWebhook() *service.Webhook {
	created := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 3, 1, 10, 0, 1, 0, time.UTC)
	return &service.Webhook{
		ID:              123,
		TargetURL:       "https://example.com/webhook",
		Event:           "file.uploaded",
		IsActive:        true,
		DatetimeCreated: created,
		DatetimeUpdated: updated,
	}
}

func TestWebhookList_Human(t *testing.T) {
	mock := &mockWebhookService{
		listFunc: func(ctx context.Context) ([]service.Webhook, error) {
			return []service.Webhook{*testWebhook()}, nil
		},
	}

	root := newTestRootWithWebhook(mock)
	stdout, _, err := executeCommand(t, root, "webhook", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"ID", "TARGET_URL", "EVENT", "ACTIVE", "CREATED", "123", "example.com"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestWebhookList_JSON(t *testing.T) {
	mock := &mockWebhookService{
		listFunc: func(ctx context.Context) ([]service.Webhook, error) {
			return []service.Webhook{*testWebhook()}, nil
		},
	}

	root := newTestRootWithWebhook(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "webhook", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\ngot: %s", err, stdout)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0]["target_url"] != "https://example.com/webhook" {
		t.Errorf("target_url = %v", result[0]["target_url"])
	}
}

func TestWebhookCreate_Human(t *testing.T) {
	mock := &mockWebhookService{
		createFunc: func(ctx context.Context, params service.WebhookCreateParams) (*service.Webhook, error) {
			return testWebhook(), nil
		},
	}

	root := newTestRootWithWebhook(mock)
	stdout, _, err := executeCommand(t, root, "webhook", "create", "https://example.com/webhook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"ID:", "123", "Target URL:", "Event:"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestWebhookCreate_JSON(t *testing.T) {
	mock := &mockWebhookService{
		createFunc: func(ctx context.Context, params service.WebhookCreateParams) (*service.Webhook, error) {
			return testWebhook(), nil
		},
	}

	root := newTestRootWithWebhook(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "webhook", "create", "https://example.com/webhook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["target_url"] != "https://example.com/webhook" {
		t.Errorf("target_url = %v", result["target_url"])
	}
}

func TestWebhookCreate_InvalidURL(t *testing.T) {
	mock := &mockWebhookService{}
	root := newTestRootWithWebhook(mock)
	_, _, err := executeCommand(t, root, "webhook", "create", "not-a-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}

func TestWebhookCreate_InvalidEvent(t *testing.T) {
	mock := &mockWebhookService{}
	root := newTestRootWithWebhook(mock)
	_, _, err := executeCommand(t, root, "webhook", "create", "https://example.com/webhook", "--event", "bad.event")
	if err == nil {
		t.Fatal("expected error for invalid event")
	}
}

func TestWebhookCreate_DryRun(t *testing.T) {
	mock := &mockWebhookService{}
	root := newTestRootWithWebhook(mock)
	stdout, _, err := executeCommand(t, root, "webhook", "create", "--dry-run", "https://example.com/webhook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would create") {
		t.Errorf("dry-run output missing 'Would create'\ngot: %s", stdout)
	}
}

func TestWebhookUpdate_Human(t *testing.T) {
	mock := &mockWebhookService{
		updateFunc: func(ctx context.Context, id string, params service.WebhookUpdateParams) (*service.Webhook, error) {
			return testWebhook(), nil
		},
	}

	root := newTestRootWithWebhook(mock)
	stdout, _, err := executeCommand(t, root, "webhook", "update", "123", "--target-url", "https://new.example.com/webhook")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "ID:") {
		t.Errorf("output missing webhook info\ngot: %s", stdout)
	}
}

func TestWebhookUpdate_InvalidID(t *testing.T) {
	mock := &mockWebhookService{}
	root := newTestRootWithWebhook(mock)
	_, _, err := executeCommand(t, root, "webhook", "update", "not-a-number")
	if err == nil {
		t.Fatal("expected error for invalid webhook ID")
	}
}

func TestWebhookDelete_Human(t *testing.T) {
	var capturedID string
	mock := &mockWebhookService{
		deleteFunc: func(ctx context.Context, id string) error {
			capturedID = id
			return nil
		},
	}

	root := newTestRootWithWebhook(mock)
	stdout, _, err := executeCommand(t, root, "webhook", "delete", "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "123" {
		t.Errorf("expected delete(123), got delete(%s)", capturedID)
	}
	if !strings.Contains(stdout, "Deleted webhook") {
		t.Errorf("output missing confirmation\ngot: %s", stdout)
	}
}

func TestWebhookDelete_JSON(t *testing.T) {
	mock := &mockWebhookService{
		deleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}

	root := newTestRootWithWebhook(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "webhook", "delete", "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["status"] != "deleted" {
		t.Errorf("status = %v, want deleted", result["status"])
	}
}

func TestWebhookDelete_DryRun(t *testing.T) {
	mock := &mockWebhookService{}
	root := newTestRootWithWebhook(mock)
	stdout, _, err := executeCommand(t, root, "webhook", "delete", "--dry-run", "123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would delete") {
		t.Errorf("dry-run output missing 'Would delete'\ngot: %s", stdout)
	}
}
