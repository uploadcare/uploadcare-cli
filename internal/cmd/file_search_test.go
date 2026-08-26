package cmd

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/uploadcare/uploadcare-cli/internal/service"
)

func TestFileSearch_MultipleFiltersAndJSON(t *testing.T) {
	var got service.FileSearchOptions
	file := *testFile()
	file.Tags = []string{"approved", "featured"}
	mock := &mockFileService{
		searchFunc: func(_ context.Context, opts service.FileSearchOptions) (*service.FileSearchResult, error) {
			got = opts
			return &service.FileSearchResult{Files: []service.File{file}, Total: 7}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, stderr, err := executeCommand(t, root,
		"file", "search", "invoice",
		"--phrase", "metadata=project alpha",
		"--exact", "detected_mime_type=application/pdf",
		"--exact", "detected_mime_type=image/tiff",
		"--tag-all", " Approved ", "--tag-all", "FEATURED",
		"--tag-none", "archived",
		"--sort", "score", "--sort=-datetime_uploaded",
		"--is-image=false", "--limit", "25", "--offset", "10",
		"--json", "all",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Query != "invoice" || got.Limit != 25 || got.Offset != 10 {
		t.Fatalf("unexpected options: %+v", got)
	}
	if got.IsImage == nil || *got.IsImage {
		t.Fatalf("is_image = %v, want pointer to false", got.IsImage)
	}
	if !reflect.DeepEqual(got.Exact["detected_mime_type"], []string{"application/pdf", "image/tiff"}) {
		t.Errorf("exact values = %v", got.Exact)
	}
	if !reflect.DeepEqual(got.Tags.All, []string{"approved", "featured"}) || !reflect.DeepEqual(got.Tags.None, []string{"archived"}) {
		t.Errorf("tags = %+v", got.Tags)
	}
	if !reflect.DeepEqual(got.Sort, []string{"score", "-datetime_uploaded"}) {
		t.Errorf("sort = %v", got.Sort)
	}
	var results []map[string]any
	if err := json.Unmarshal([]byte(stdout), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if len(results) != 1 || results[0]["uuid"] != file.UUID {
		t.Fatalf("unexpected results: %s", stdout)
	}
	if !strings.Contains(stderr, "Found 7 matches; showing 1.") {
		t.Errorf("missing result status: %q", stderr)
	}
}

func TestFileSearch_RejectsInvalidIsImageValue(t *testing.T) {
	called := false
	mock := &mockFileService{searchFunc: func(_ context.Context, _ service.FileSearchOptions) (*service.FileSearchResult, error) {
		called = true
		return nil, nil
	}}

	_, _, err := executeCommand(t, newTestRoot(mock), "file", "search", "--is-image", "yes")
	if err == nil {
		t.Fatal("expected validation error")
	}
	exitErr, ok := err.(*ExitError)
	if !ok || exitErr.Code != 2 {
		t.Fatalf("error = %T %v, want exit code 2", err, err)
	}
	if called {
		t.Fatal("service was called for invalid --is-image value")
	}
}

func TestFileSearch_RequiresCriterionBeforeServiceCall(t *testing.T) {
	called := false
	mock := &mockFileService{searchFunc: func(_ context.Context, _ service.FileSearchOptions) (*service.FileSearchResult, error) {
		called = true
		return nil, nil
	}}

	_, _, err := executeCommand(t, newTestRoot(mock), "file", "search", "--sort", "score")
	if err == nil {
		t.Fatal("expected validation error")
	}
	exitErr, ok := err.(*ExitError)
	if !ok || exitErr.Code != 2 {
		t.Fatalf("error = %T %v, want exit code 2", err, err)
	}
	if called {
		t.Fatal("service was called for invalid search")
	}
}

func TestFileSearch_PageAllStreamsResults(t *testing.T) {
	mock := &mockFileService{
		iterateSearchFunc: func(_ context.Context, _ service.FileSearchOptions, fn func(service.File) error) (uint64, error) {
			for i := 0; i < 2; i++ {
				if err := fn(*testFile()); err != nil {
					return 2, err
				}
			}
			return 2, nil
		},
	}
	stdout, stderr, err := executeCommand(t, newTestRoot(mock),
		"file", "search", "invoice", "--page-all", "--json", "uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lines := strings.Count(strings.TrimSpace(stdout), "\n") + 1; lines != 2 {
		t.Fatalf("got %d NDJSON lines: %q", lines, stdout)
	}
	if !strings.Contains(stderr, "Found 2 matches; showing 2.") {
		t.Errorf("missing status: %q", stderr)
	}
}
