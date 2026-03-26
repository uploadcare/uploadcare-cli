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

// --- Mock services ---

type mockProjectService struct {
	infoFunc func(ctx context.Context) (*service.Project, error)
}

func (m *mockProjectService) Info(ctx context.Context) (*service.Project, error) {
	if m.infoFunc != nil {
		return m.infoFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

type mockProjectManagementService struct {
	listFunc   func(ctx context.Context, opts service.ProjectListOptions) (*service.ProjectListResult, error)
	getFunc    func(ctx context.Context, pubKey string) (*service.ManagedProject, error)
	createFunc func(ctx context.Context, params service.ProjectCreateParams) (*service.ManagedProject, error)
	updateFunc func(ctx context.Context, pubKey string, params service.ProjectUpdateParams) (*service.ManagedProject, error)
	deleteFunc func(ctx context.Context, pubKey string) error
}

func (m *mockProjectManagementService) List(ctx context.Context, opts service.ProjectListOptions) (*service.ProjectListResult, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, opts)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProjectManagementService) Get(ctx context.Context, pubKey string) (*service.ManagedProject, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, pubKey)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProjectManagementService) Create(ctx context.Context, params service.ProjectCreateParams) (*service.ManagedProject, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProjectManagementService) Update(ctx context.Context, pubKey string, params service.ProjectUpdateParams) (*service.ManagedProject, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, pubKey, params)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProjectManagementService) Delete(ctx context.Context, pubKey string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, pubKey)
	}
	return errors.New("not implemented")
}

type mockSecretService struct {
	listFunc   func(ctx context.Context, pubKey string) ([]service.Secret, error)
	createFunc func(ctx context.Context, pubKey string) (*service.SecretCreateResult, error)
	deleteFunc func(ctx context.Context, pubKey, secretID string) error
}

func (m *mockSecretService) List(ctx context.Context, pubKey string) ([]service.Secret, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, pubKey)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSecretService) Create(ctx context.Context, pubKey string) (*service.SecretCreateResult, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, pubKey)
	}
	return nil, errors.New("not implemented")
}

func (m *mockSecretService) Delete(ctx context.Context, pubKey, secretID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, pubKey, secretID)
	}
	return errors.New("not implemented")
}

type mockUsageService struct {
	combinedFunc func(ctx context.Context, pubKey string, from, to string) (*service.UsageResult, error)
	metricFunc   func(ctx context.Context, pubKey, metric string, from, to string) (*service.MetricResult, error)
}

func (m *mockUsageService) Combined(ctx context.Context, pubKey string, from, to string) (*service.UsageResult, error) {
	if m.combinedFunc != nil {
		return m.combinedFunc(ctx, pubKey, from, to)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUsageService) Metric(ctx context.Context, pubKey, metric string, from, to string) (*service.MetricResult, error) {
	if m.metricFunc != nil {
		return m.metricFunc(ctx, pubKey, metric, from, to)
	}
	return nil, errors.New("not implemented")
}

// --- Test helpers ---

func newTestRootWithProject(
	projSvc service.ProjectService,
	mgmtSvc service.ProjectManagementService,
	secretSvc service.SecretService,
	usageSvc service.UsageService,
) *cobra.Command {
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

	root.AddCommand(newProjectCmd(projSvc, mgmtSvc, secretSvc, usageSvc))
	return root
}

func testManagedProject() *service.ManagedProject {
	autostore := true
	filesizeLimit := int64(10485760)
	return &service.ManagedProject{
		PubKey:           "abc123",
		Name:             "My App",
		AutostoreEnabled: &autostore,
		FilesizeLimit:    &filesizeLimit,
		IsSharedProject:  false,
	}
}

func testRESTProject() *service.Project {
	return &service.Project{
		Name:             "My App",
		PubKey:           "abc123",
		AutostoreEnabled: true,
		Collaborators: []service.Collaborator{
			{Name: "Alice", Email: "alice@example.com"},
		},
	}
}

// --- Project Info tests ---

func TestProjectInfo_REST_Human(t *testing.T) {
	mock := &mockProjectService{
		infoFunc: func(ctx context.Context) (*service.Project, error) {
			return testRESTProject(), nil
		},
	}

	root := newTestRootWithProject(mock, nil, nil, nil)
	stdout, _, err := executeCommand(t, root, "project", "info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"Name:", "My App", "Public Key:", "abc123", "Autostore:", "Alice"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestProjectInfo_REST_JSON(t *testing.T) {
	mock := &mockProjectService{
		infoFunc: func(ctx context.Context) (*service.Project, error) {
			return testRESTProject(), nil
		},
	}

	root := newTestRootWithProject(mock, nil, nil, nil)
	stdout, _, err := executeCommand(t, root, "--json", "all", "project", "info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if result["pub_key"] != "abc123" {
		t.Errorf("pub_key = %v", result["pub_key"])
	}
}

func TestProjectInfo_ProjectAPI_Human(t *testing.T) {
	mock := &mockProjectManagementService{
		getFunc: func(ctx context.Context, pubKey string) (*service.ManagedProject, error) {
			if pubKey != "abc123" {
				t.Errorf("expected pubKey abc123, got %s", pubKey)
			}
			return testManagedProject(), nil
		},
	}

	root := newTestRootWithProject(nil, mock, nil, nil)
	stdout, _, err := executeCommand(t, root, "project", "info", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"Name:", "My App", "Public Key:", "abc123"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

// --- Project List tests ---

func TestProjectList_Human(t *testing.T) {
	mock := &mockProjectManagementService{
		listFunc: func(ctx context.Context, opts service.ProjectListOptions) (*service.ProjectListResult, error) {
			return &service.ProjectListResult{
				Projects: []service.ManagedProject{*testManagedProject()},
				Total:    1,
			}, nil
		},
	}

	root := newTestRootWithProject(nil, mock, nil, nil)
	stdout, _, err := executeCommand(t, root, "project", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"PUB_KEY", "NAME", "abc123", "My App"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestProjectList_JSON(t *testing.T) {
	mock := &mockProjectManagementService{
		listFunc: func(ctx context.Context, opts service.ProjectListOptions) (*service.ProjectListResult, error) {
			return &service.ProjectListResult{
				Projects: []service.ManagedProject{*testManagedProject()},
				Total:    1,
			}, nil
		},
	}

	root := newTestRootWithProject(nil, mock, nil, nil)
	stdout, _, err := executeCommand(t, root, "--json", "all", "project", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}
	if result[0]["pub_key"] != "abc123" {
		t.Errorf("pub_key = %v", result[0]["pub_key"])
	}
}

// --- Project Create tests ---

func TestProjectCreate_Human(t *testing.T) {
	mgmt := &mockProjectManagementService{
		createFunc: func(ctx context.Context, params service.ProjectCreateParams) (*service.ManagedProject, error) {
			if params.Name != "My App" {
				t.Errorf("expected name 'My App', got %q", params.Name)
			}
			return testManagedProject(), nil
		},
	}
	sec := &mockSecretService{
		createFunc: func(ctx context.Context, pubKey string) (*service.SecretCreateResult, error) {
			return &service.SecretCreateResult{ID: "sec_1", Secret: "secret123"}, nil
		},
	}

	root := newTestRootWithProject(nil, mgmt, sec, nil)
	stdout, _, err := executeCommand(t, root, "project", "create", "My App", "--no-save")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"Created project", "My App", "abc123", "secret123"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestProjectCreate_NoSaveAndUseMutuallyExclusive(t *testing.T) {
	mgmt := &mockProjectManagementService{}
	sec := &mockSecretService{}

	root := newTestRootWithProject(nil, mgmt, sec, nil)
	_, _, err := executeCommand(t, root, "project", "create", "My App", "--no-save", "--use")
	if err == nil {
		t.Fatal("expected error when both --no-save and --use are provided")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}

func TestProjectCreate_DryRun(t *testing.T) {
	mgmt := &mockProjectManagementService{}
	sec := &mockSecretService{}

	root := newTestRootWithProject(nil, mgmt, sec, nil)
	stdout, _, err := executeCommand(t, root, "project", "create", "My App", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would create") {
		t.Errorf("dry-run output missing 'Would create'\ngot: %s", stdout)
	}
}

func TestProjectCreate_JSON(t *testing.T) {
	mgmt := &mockProjectManagementService{
		createFunc: func(ctx context.Context, params service.ProjectCreateParams) (*service.ManagedProject, error) {
			return testManagedProject(), nil
		},
	}
	sec := &mockSecretService{
		createFunc: func(ctx context.Context, pubKey string) (*service.SecretCreateResult, error) {
			return &service.SecretCreateResult{ID: "sec_1", Secret: "secret123"}, nil
		},
	}

	root := newTestRootWithProject(nil, mgmt, sec, nil)
	stdout, _, err := executeCommand(t, root, "--json", "all", "project", "create", "My App", "--no-save")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if result["pub_key"] != "abc123" {
		t.Errorf("pub_key = %v", result["pub_key"])
	}
	if result["secret_key"] != "secret123" {
		t.Errorf("secret_key = %v", result["secret_key"])
	}
}

// --- Project Update tests ---

func TestProjectUpdate_Human(t *testing.T) {
	mock := &mockProjectManagementService{
		updateFunc: func(ctx context.Context, pubKey string, params service.ProjectUpdateParams) (*service.ManagedProject, error) {
			if pubKey != "abc123" {
				t.Errorf("expected pubKey abc123, got %s", pubKey)
			}
			if params.Name == nil || *params.Name != "New Name" {
				t.Errorf("expected name 'New Name', got %v", params.Name)
			}
			return testManagedProject(), nil
		},
	}

	root := newTestRootWithProject(nil, mock, nil, nil)
	stdout, _, err := executeCommand(t, root, "project", "update", "abc123", "--name", "New Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout, "Name:") {
		t.Errorf("output missing project info\ngot: %s", stdout)
	}
}

func TestProjectUpdate_DryRun(t *testing.T) {
	mock := &mockProjectManagementService{}
	root := newTestRootWithProject(nil, mock, nil, nil)
	stdout, _, err := executeCommand(t, root, "project", "update", "abc123", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would update") {
		t.Errorf("dry-run output missing 'Would update'\ngot: %s", stdout)
	}
}

// --- Project Delete tests ---

func TestProjectDelete_Human(t *testing.T) {
	var capturedKey string
	mock := &mockProjectManagementService{
		deleteFunc: func(ctx context.Context, pubKey string) error {
			capturedKey = pubKey
			return nil
		},
	}

	root := newTestRootWithProject(nil, mock, nil, nil)
	stdout, _, err := executeCommand(t, root, "project", "delete", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedKey != "abc123" {
		t.Errorf("expected delete(abc123), got delete(%s)", capturedKey)
	}
	if !strings.Contains(stdout, "Deleted project") {
		t.Errorf("output missing confirmation\ngot: %s", stdout)
	}
}

func TestProjectDelete_JSON(t *testing.T) {
	mock := &mockProjectManagementService{
		deleteFunc: func(ctx context.Context, pubKey string) error {
			return nil
		},
	}

	root := newTestRootWithProject(nil, mock, nil, nil)
	stdout, _, err := executeCommand(t, root, "--json", "all", "project", "delete", "abc123")
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

func TestProjectDelete_DryRun(t *testing.T) {
	mock := &mockProjectManagementService{}
	root := newTestRootWithProject(nil, mock, nil, nil)
	stdout, _, err := executeCommand(t, root, "project", "delete", "abc123", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would delete") {
		t.Errorf("dry-run output missing 'Would delete'\ngot: %s", stdout)
	}
}

// --- Project Use tests ---

func TestProjectUse_RequiresSecretFlag(t *testing.T) {
	mock := &mockProjectManagementService{}
	root := newTestRootWithProject(nil, mock, nil, nil)
	_, _, err := executeCommand(t, root, "project", "use", "abc123")
	if err == nil {
		t.Fatal("expected error when neither --secret-key nor --create-secret provided")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}

func TestProjectUse_EmptySecretKey(t *testing.T) {
	mock := &mockProjectManagementService{}
	root := newTestRootWithProject(nil, mock, nil, nil)
	_, _, err := executeCommand(t, root, "project", "use", "abc123", "--secret-key", "")
	if err == nil {
		t.Fatal("expected error for empty --secret-key")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}

func TestProjectUse_MutuallyExclusive(t *testing.T) {
	mock := &mockProjectManagementService{}
	root := newTestRootWithProject(nil, mock, nil, nil)
	_, _, err := executeCommand(t, root, "project", "use", "abc123", "--secret-key", "sk", "--create-secret")
	if err == nil {
		t.Fatal("expected error when both --secret-key and --create-secret provided")
	}
}

// --- Secret tests ---

func TestSecretList_Human(t *testing.T) {
	lastUsed := "2026-03-01T10:00:00Z"
	mock := &mockSecretService{
		listFunc: func(ctx context.Context, pubKey string) ([]service.Secret, error) {
			return []service.Secret{
				{ID: "sec_1", Hint: "ab**", LastUsedAt: &lastUsed},
			}, nil
		},
	}

	root := newTestRootWithProject(nil, nil, mock, nil)
	stdout, _, err := executeCommand(t, root, "project", "secret", "list", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"ID", "HINT", "sec_1", "ab**"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestSecretList_JSON(t *testing.T) {
	mock := &mockSecretService{
		listFunc: func(ctx context.Context, pubKey string) ([]service.Secret, error) {
			return []service.Secret{
				{ID: "sec_1", Hint: "ab**"},
			}, nil
		},
	}

	root := newTestRootWithProject(nil, nil, mock, nil)
	stdout, _, err := executeCommand(t, root, "--json", "all", "project", "secret", "list", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if len(result) != 1 || result[0]["id"] != "sec_1" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestSecretCreate_Human(t *testing.T) {
	mock := &mockSecretService{
		createFunc: func(ctx context.Context, pubKey string) (*service.SecretCreateResult, error) {
			return &service.SecretCreateResult{ID: "sec_1", Secret: "full-secret-key"}, nil
		},
	}

	root := newTestRootWithProject(nil, nil, mock, nil)
	stdout, _, err := executeCommand(t, root, "project", "secret", "create", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"ID:", "sec_1", "Secret:", "full-secret-key"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestSecretDelete_Human(t *testing.T) {
	var capturedPubKey, capturedSecretID string
	mock := &mockSecretService{
		deleteFunc: func(ctx context.Context, pubKey, secretID string) error {
			capturedPubKey = pubKey
			capturedSecretID = secretID
			return nil
		},
	}

	root := newTestRootWithProject(nil, nil, mock, nil)
	stdout, _, err := executeCommand(t, root, "project", "secret", "delete", "abc123", "sec_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPubKey != "abc123" || capturedSecretID != "sec_1" {
		t.Errorf("expected delete(abc123, sec_1), got delete(%s, %s)", capturedPubKey, capturedSecretID)
	}
	if !strings.Contains(stdout, "Deleted secret") {
		t.Errorf("output missing confirmation\ngot: %s", stdout)
	}
}

func TestSecretDelete_DryRun(t *testing.T) {
	mock := &mockSecretService{}
	root := newTestRootWithProject(nil, nil, mock, nil)
	stdout, _, err := executeCommand(t, root, "project", "secret", "delete", "abc123", "sec_1", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Would delete") {
		t.Errorf("dry-run output missing 'Would delete'\ngot: %s", stdout)
	}
}

// --- Usage tests ---

func TestUsage_Combined_Human(t *testing.T) {
	mock := &mockUsageService{
		combinedFunc: func(ctx context.Context, pubKey string, from, to string) (*service.UsageResult, error) {
			return &service.UsageResult{
				Units: map[string]string{"traffic": "bytes", "storage": "bytes", "operations": "count"},
				Data: []service.UsageDayMetrics{
					{Date: "2026-03-01", Traffic: 1024, Storage: 2048, Operations: 100},
				},
			}, nil
		},
	}

	root := newTestRootWithProject(nil, nil, nil, mock)
	stdout, _, err := executeCommand(t, root, "project", "usage", "abc123", "--from", "2025-02-01", "--to", "2025-03-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"DATE", "TRAFFIC", "STORAGE", "OPERATIONS", "2026-03-01", "1024"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestUsage_Combined_JSON(t *testing.T) {
	mock := &mockUsageService{
		combinedFunc: func(ctx context.Context, pubKey string, from, to string) (*service.UsageResult, error) {
			return &service.UsageResult{
				Units: map[string]string{"traffic": "bytes"},
				Data: []service.UsageDayMetrics{
					{Date: "2026-03-01", Traffic: 1024},
				},
			}, nil
		},
	}

	root := newTestRootWithProject(nil, nil, nil, mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "project", "usage", "abc123", "--from", "2025-02-01", "--to", "2025-03-01")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if result["units"] == nil {
		t.Error("missing units in JSON output")
	}
}

func TestUsage_Metric_Human(t *testing.T) {
	mock := &mockUsageService{
		metricFunc: func(ctx context.Context, pubKey, metric string, from, to string) (*service.MetricResult, error) {
			if metric != "traffic" {
				t.Errorf("expected metric 'traffic', got %q", metric)
			}
			return &service.MetricResult{
				Metric: "traffic",
				Unit:   "bytes",
				Data: []service.MetricDayData{
					{Date: "2026-03-01", Value: 1024},
				},
			}, nil
		},
	}

	root := newTestRootWithProject(nil, nil, nil, mock)
	stdout, _, err := executeCommand(t, root, "project", "usage", "abc123", "--from", "2025-02-01", "--to", "2025-03-01", "--metric", "traffic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, s := range []string{"traffic", "bytes", "DATE", "VALUE", "2026-03-01", "1024"} {
		if !strings.Contains(stdout, s) {
			t.Errorf("output missing %q\ngot:\n%s", s, stdout)
		}
	}
}

func TestUsage_InvalidDate(t *testing.T) {
	mock := &mockUsageService{}
	root := newTestRootWithProject(nil, nil, nil, mock)
	_, _, err := executeCommand(t, root, "project", "usage", "abc123", "--from", "not-a-date", "--to", "2026-03-31")
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
}

func TestUsage_InvalidMetric(t *testing.T) {
	mock := &mockUsageService{}
	root := newTestRootWithProject(nil, nil, nil, mock)
	_, _, err := executeCommand(t, root, "project", "usage", "abc123", "--from", "2025-02-01", "--to", "2025-03-01", "--metric", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid metric")
	}
}

func TestUsage_MissingRequiredFlags(t *testing.T) {
	mock := &mockUsageService{}
	root := newTestRootWithProject(nil, nil, nil, mock)
	_, _, err := executeCommand(t, root, "project", "usage", "abc123")
	if err == nil {
		t.Fatal("expected error when --from/--to are missing")
	}
}

func TestUsage_ToDateIsToday(t *testing.T) {
	mock := &mockUsageService{}
	root := newTestRootWithProject(nil, nil, nil, mock)
	today := time.Now().UTC().Format("2006-01-02")
	_, _, err := executeCommand(t, root, "project", "usage", "abc123", "--from", "2026-01-01", "--to", today)
	if err == nil {
		t.Fatal("expected error when --to is today")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
	if !strings.Contains(err.Error(), "before today") {
		t.Errorf("expected error to mention 'before today', got %q", err.Error())
	}
}

func TestUsage_ToDateInFuture(t *testing.T) {
	mock := &mockUsageService{}
	root := newTestRootWithProject(nil, nil, nil, mock)
	future := time.Now().UTC().AddDate(0, 1, 0).Format("2006-01-02")
	_, _, err := executeCommand(t, root, "project", "usage", "abc123", "--from", "2026-01-01", "--to", future)
	if err == nil {
		t.Fatal("expected error when --to is in the future")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
	if !strings.Contains(err.Error(), "before today") {
		t.Errorf("expected error to mention 'before today', got %q", err.Error())
	}
}

func TestUsage_FromAfterTo(t *testing.T) {
	mock := &mockUsageService{}
	root := newTestRootWithProject(nil, nil, nil, mock)
	_, _, err := executeCommand(t, root, "project", "usage", "abc123", "--from", "2026-03-15", "--to", "2026-03-01")
	if err == nil {
		t.Fatal("expected error when --from is after --to")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
	if !strings.Contains(err.Error(), "must be before") {
		t.Errorf("expected error to mention 'must be before', got %q", err.Error())
	}
}

func TestUsage_RangeExceeds90Days(t *testing.T) {
	mock := &mockUsageService{}
	root := newTestRootWithProject(nil, nil, nil, mock)
	_, _, err := executeCommand(t, root, "project", "usage", "abc123", "--from", "2025-01-01", "--to", "2025-06-01")
	if err == nil {
		t.Fatal("expected error when range exceeds 90 days")
	}
	var exitErr *ExitError
	if !errors.As(err, &exitErr) || exitErr.Code != 2 {
		t.Errorf("expected ExitError with code 2, got %v", err)
	}
	if !strings.Contains(err.Error(), "90") {
		t.Errorf("expected error to mention '90', got %q", err.Error())
	}
}
