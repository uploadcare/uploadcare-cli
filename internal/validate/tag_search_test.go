package validate

import (
	"reflect"
	"strings"
	"testing"

	"github.com/uploadcare/uploadcare-cli/internal/service"
)

func TestNormalizeTags(t *testing.T) {
	got, err := NormalizeTags([]string{" Cat ", "ANIMAL", "cat", "with.dot"}, MaxTagCount)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"cat", "animal", "with.dot"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestNormalizeTagsRejectsInvalidValue(t *testing.T) {
	if _, err := NormalizeTags([]string{"has space"}, MaxTagCount); err == nil {
		t.Fatal("expected invalid-character error")
	}
	if _, err := NormalizeTags([]string{strings.Repeat("a", MaxTagLength+1)}, MaxTagCount); err == nil {
		t.Fatal("expected length error")
	}
}

func TestFileSearchValidation(t *testing.T) {
	valid := service.FileSearchOptions{Query: "cats", Limit: 20}
	if err := FileSearch(valid); err != nil {
		t.Fatalf("valid search rejected: %v", err)
	}
	conflict := service.FileSearchOptions{
		Phrase: &service.FileSearchPhrase{Metadata: "camera"},
		Exact:  map[string][]string{"metadata[source]": {"web"}},
		Limit:  20,
	}
	if err := FileSearch(conflict); err == nil {
		t.Fatal("expected phrase/exact conflict")
	}
	window := service.FileSearchOptions{Query: "cats", Limit: 20, Offset: 981}
	if err := FileSearch(window); err == nil {
		t.Fatal("expected pagination window error")
	}
}
