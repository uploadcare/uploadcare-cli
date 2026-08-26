package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/service"
)

type mockTagService struct {
	listFunc    func(context.Context, string) ([]string, error)
	replaceFunc func(context.Context, string, []string) (*service.TagChangeResult, error)
	updateFunc  func(context.Context, string, service.TagUpdateOptions) (*service.TagChangeResult, error)
}

func (m *mockTagService) List(ctx context.Context, uuid string) ([]string, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, uuid)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTagService) Replace(ctx context.Context, uuid string, tags []string) (*service.TagChangeResult, error) {
	if m.replaceFunc != nil {
		return m.replaceFunc(ctx, uuid, tags)
	}
	return nil, errors.New("not implemented")
}

func (m *mockTagService) Update(ctx context.Context, uuid string, opts service.TagUpdateOptions) (*service.TagChangeResult, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, uuid, opts)
	}
	return nil, errors.New("not implemented")
}

func newTagTestRoot(mock service.TagService) *cobra.Command {
	root := &cobra.Command{Use: "uploadcare", SilenceUsage: true, SilenceErrors: true}
	flags := root.PersistentFlags()
	flags.String("json", "", "Output as JSON")
	flags.String("jq", "", "jq expression")
	flags.BoolP("quiet", "q", false, "Suppress output")
	flags.BoolP("verbose", "v", false, "Verbose output")
	root.AddCommand(newTagCmd(mock))
	return root
}

func TestTagUpdate_AcceptsMultipleAddAndDeleteFlags(t *testing.T) {
	var got service.TagUpdateOptions
	mock := &mockTagService{updateFunc: func(_ context.Context, _ string, opts service.TagUpdateOptions) (*service.TagChangeResult, error) {
		got = opts
		return &service.TagChangeResult{Tags: []string{"one", "two"}, Added: []string{"one", "two"}, Deleted: []string{"old"}}, nil
	}}

	stdout, _, err := executeCommand(t, newTagTestRoot(mock),
		"tag", "update", "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"--add", " One ", "--add", "two", "--add", "ONE",
		"--delete", "old", "--delete", "ONE", "--json", "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got.Add, []string{"one", "two"}) {
		t.Errorf("add = %v", got.Add)
	}
	if !reflect.DeepEqual(got.Delete, []string{"old", "one"}) {
		t.Errorf("delete = %v", got.Delete)
	}
	var result service.TagChangeResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !reflect.DeepEqual(result.Tags, []string{"one", "two"}) {
		t.Errorf("result = %+v", result)
	}
}

func TestTagUpdate_DryRunDeletesBeforeAdding(t *testing.T) {
	updated := false
	mock := &mockTagService{
		listFunc: func(context.Context, string) ([]string, error) {
			return []string{"keep", "swap"}, nil
		},
		updateFunc: func(context.Context, string, service.TagUpdateOptions) (*service.TagChangeResult, error) {
			updated = true
			return nil, nil
		},
	}

	stdout, _, err := executeCommand(t, newTagTestRoot(mock),
		"tag", "update", "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"--delete", "swap", "--add", "swap", "--add", "new", "--dry-run", "--json", "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated {
		t.Fatal("dry-run called Update")
	}
	var result struct {
		Tags, Added, Deleted []string
		Status               string
	}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !reflect.DeepEqual(result.Tags, []string{"keep", "swap", "new"}) ||
		!reflect.DeepEqual(result.Added, []string{"swap", "new"}) ||
		!reflect.DeepEqual(result.Deleted, []string{"swap"}) {
		t.Fatalf("unexpected dry-run result: %+v", result)
	}
	if result.Status != "would change" {
		t.Errorf("dry-run JSON status = %q, want \"would change\"", result.Status)
	}
}

func TestTagUpdate_RequiresAtLeastOneOperation(t *testing.T) {
	_, _, err := executeCommand(t, newTagTestRoot(&mockTagService{}),
		"tag", "update", "a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	if err == nil {
		t.Fatal("expected error")
	}
	exitErr, ok := err.(*ExitError)
	if !ok || exitErr.Code != 2 {
		t.Fatalf("error = %T %v, want exit code 2", err, err)
	}
}
