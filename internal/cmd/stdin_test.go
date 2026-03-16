package cmd

import (
	"strings"
	"testing"
)

func TestReadLinesOrNDJSON_PlainUUIDs(t *testing.T) {
	input := "uuid-1\nuuid-2\nuuid-3\n"
	got, err := ReadLinesOrNDJSON(strings.NewReader(input), "uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"uuid-1", "uuid-2", "uuid-3"}
	if len(got) != len(want) {
		t.Fatalf("got %d values, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestReadLinesOrNDJSON_NDJSON(t *testing.T) {
	input := `{"uuid":"a1","size":100}
{"uuid":"b2","size":200}
`
	got, err := ReadLinesOrNDJSON(strings.NewReader(input), "uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"a1", "b2"}
	if len(got) != len(want) {
		t.Fatalf("got %d values, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestReadLinesOrNDJSON_WhitespaceAndEmptyLines(t *testing.T) {
	input := "\n  uuid-1  \n\n  uuid-2  \n\n"
	got, err := ReadLinesOrNDJSON(strings.NewReader(input), "uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"uuid-1", "uuid-2"}
	if len(got) != len(want) {
		t.Fatalf("got %d values, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestReadLinesOrNDJSON_EmptyInput(t *testing.T) {
	got, err := ReadLinesOrNDJSON(strings.NewReader(""), "uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %v", got)
	}
}

func TestReadLinesOrNDJSON_MalformedNDJSON(t *testing.T) {
	input := `{"uuid":"ok"}
{not valid json}
`
	_, err := ReadLinesOrNDJSON(strings.NewReader(input), "uuid")
	if err == nil {
		t.Fatal("expected error for malformed NDJSON")
	}
	if !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error should mention line 2, got: %v", err)
	}
}

func TestReadLinesOrNDJSON_MissingField(t *testing.T) {
	input := `{"uuid":"ok"}
{"id":"missing-uuid-field"}
`
	_, err := ReadLinesOrNDJSON(strings.NewReader(input), "uuid")
	if err == nil {
		t.Fatal("expected error for missing field")
	}
	if !strings.Contains(err.Error(), "missing field") {
		t.Errorf("error should mention missing field, got: %v", err)
	}
}
