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

type mockGroupService struct {
	listFunc    func(ctx context.Context, opts service.GroupListOptions) (*service.GroupListResult, error)
	iterateFunc func(ctx context.Context, opts service.GroupListOptions, fn func(service.Group) error) error
	infoFunc    func(ctx context.Context, groupID string) (*service.Group, error)
	createFunc  func(ctx context.Context, uuids []string) (*service.Group, error)
	deleteFunc  func(ctx context.Context, groupID string) error
}

func (m *mockGroupService) List(ctx context.Context, opts service.GroupListOptions) (*service.GroupListResult, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, opts)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGroupService) Iterate(ctx context.Context, opts service.GroupListOptions, fn func(service.Group) error) error {
	if m.iterateFunc != nil {
		return m.iterateFunc(ctx, opts, fn)
	}
	return errors.New("not implemented")
}

func (m *mockGroupService) Info(ctx context.Context, groupID string) (*service.Group, error) {
	if m.infoFunc != nil {
		return m.infoFunc(ctx, groupID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGroupService) Create(ctx context.Context, uuids []string) (*service.Group, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, uuids)
	}
	return nil, errors.New("not implemented")
}

func (m *mockGroupService) Delete(ctx context.Context, groupID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, groupID)
	}
	return errors.New("not implemented")
}

func newTestRootWithGroup(mock service.GroupService) *cobra.Command {
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

	root.AddCommand(newGroupCmd(mock))
	return root
}

func testGroup() *service.Group {
	created := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	stored := time.Date(2026, 3, 1, 10, 0, 1, 0, time.UTC)
	return &service.Group{
		ID:              "a1b2c3d4-e5f6-7890-abcd-ef1234567890~3",
		DatetimeCreated: created,
		DatetimeStored:  &stored,
		FilesCount:      3,
		CDNURL:          "https://ucarecdn.com/a1b2c3d4-e5f6-7890-abcd-ef1234567890~3/",
	}
}

const testGroupID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890~3"

func TestGroupList_Human(t *testing.T) {
	mock := &mockGroupService{
		listFunc: func(ctx context.Context, opts service.GroupListOptions) (*service.GroupListResult, error) {
			return &service.GroupListResult{
				Groups: []service.Group{*testGroup()},
			}, nil
		},
	}

	root := newTestRootWithGroup(mock)
	stdout, _, err := executeCommand(t, root, "group", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"ID", "FILES", "CREATED", "STORED", testGroupID} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestGroupList_JSON(t *testing.T) {
	mock := &mockGroupService{
		listFunc: func(ctx context.Context, opts service.GroupListOptions) (*service.GroupListResult, error) {
			return &service.GroupListResult{
				Groups: []service.Group{*testGroup()},
			}, nil
		},
	}

	root := newTestRootWithGroup(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "group", "list")
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
	if result[0]["id"] != testGroupID {
		t.Errorf("id = %v", result[0]["id"])
	}
}

func TestGroupList_PageAll(t *testing.T) {
	mock := &mockGroupService{
		iterateFunc: func(ctx context.Context, opts service.GroupListOptions, fn func(service.Group) error) error {
			return fn(*testGroup())
		},
	}

	root := newTestRootWithGroup(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "group", "list", "--page-all")
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
	if obj["id"] != testGroupID {
		t.Errorf("id = %v", obj["id"])
	}
}

func TestGroupInfo_Human(t *testing.T) {
	mock := &mockGroupService{
		infoFunc: func(ctx context.Context, groupID string) (*service.Group, error) {
			return testGroup(), nil
		},
	}

	root := newTestRootWithGroup(mock)
	stdout, _, err := executeCommand(t, root, "group", "info", testGroupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"ID:", testGroupID, "Files:", "3", "CDN URL:"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestGroupInfo_JSON(t *testing.T) {
	mock := &mockGroupService{
		infoFunc: func(ctx context.Context, groupID string) (*service.Group, error) {
			return testGroup(), nil
		},
	}

	root := newTestRootWithGroup(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "group", "info", testGroupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["id"] != testGroupID {
		t.Errorf("id = %v", result["id"])
	}
}

func TestGroupInfo_InvalidGroupID(t *testing.T) {
	mock := &mockGroupService{}
	root := newTestRootWithGroup(mock)
	_, _, err := executeCommand(t, root, "group", "info", "bad-id")
	if err == nil {
		t.Fatal("expected error for invalid group ID")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}

func TestGroupCreate_Human(t *testing.T) {
	mock := &mockGroupService{
		createFunc: func(ctx context.Context, uuids []string) (*service.Group, error) {
			return testGroup(), nil
		},
	}

	root := newTestRootWithGroup(mock)
	stdout, _, err := executeCommand(t, root, "group", "create", testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "ID:") {
		t.Errorf("output missing group info\ngot: %s", stdout)
	}
}

func TestGroupCreate_DryRun(t *testing.T) {
	mock := &mockGroupService{}
	root := newTestRootWithGroup(mock)
	stdout, _, err := executeCommand(t, root, "group", "create", "--dry-run", testUUID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would create") {
		t.Errorf("dry-run output missing 'Would create'\ngot: %s", stdout)
	}
}

func TestGroupCreate_NoUUIDs(t *testing.T) {
	mock := &mockGroupService{}
	root := newTestRootWithGroup(mock)
	_, _, err := executeCommand(t, root, "group", "create")
	if err == nil {
		t.Fatal("expected error for no UUIDs")
	}
}

func TestGroupDelete_Human(t *testing.T) {
	var capturedID string
	mock := &mockGroupService{
		deleteFunc: func(ctx context.Context, groupID string) error {
			capturedID = groupID
			return nil
		},
	}

	root := newTestRootWithGroup(mock)
	stdout, _, err := executeCommand(t, root, "group", "delete", testGroupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != testGroupID {
		t.Errorf("expected delete(%s), got delete(%s)", testGroupID, capturedID)
	}
	if !strings.Contains(stdout, "Deleted group") {
		t.Errorf("output missing confirmation\ngot: %s", stdout)
	}
}

func TestGroupDelete_JSON(t *testing.T) {
	mock := &mockGroupService{
		deleteFunc: func(ctx context.Context, groupID string) error {
			return nil
		},
	}

	root := newTestRootWithGroup(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "group", "delete", testGroupID)
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

func TestGroupDelete_DryRun(t *testing.T) {
	mock := &mockGroupService{
		infoFunc: func(ctx context.Context, groupID string) (*service.Group, error) {
			return testGroup(), nil
		},
	}

	root := newTestRootWithGroup(mock)
	stdout, _, err := executeCommand(t, root, "group", "delete", "--dry-run", testGroupID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would delete") {
		t.Errorf("dry-run output missing 'Would delete'\ngot: %s", stdout)
	}
}
