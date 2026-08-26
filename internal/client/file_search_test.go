package client

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/file"
	"github.com/uploadcare/uploadcare-go/v2/tag"
)

func TestBuildSearchParams(t *testing.T) {
	isImage := false
	size := uint64(1024)
	opts := service.FileSearchOptions{
		Limit: 25, Offset: 10, IncludeAppData: true, Query: "invoice",
		Phrase:  &service.FileSearchPhrase{Metadata: "project alpha"},
		Exact:   map[string][]string{"detected_mime_type": {"application/pdf"}},
		Size:    &service.FileSearchSize{Gte: &size},
		IsImage: &isImage,
		Tags:    &service.FileSearchTags{All: []string{"approved"}, None: []string{"archived"}},
		Sort:    []string{"score", "-datetime_uploaded"},
	}

	params := buildSearchParams(opts)
	if params.Limit == nil || *params.Limit != 25 || params.Offset == nil || *params.Offset != 10 {
		t.Fatalf("pagination params = %+v", params)
	}
	if params.Include == nil || *params.Include != file.SearchIncludeAppData {
		t.Errorf("include = %v", params.Include)
	}
	if params.Phrase == nil || params.Phrase.Metadata != "project alpha" {
		t.Errorf("phrase = %+v", params.Phrase)
	}
	if params.Size == nil || params.Size.Gte == nil || *params.Size.Gte != 1024 {
		t.Errorf("size = %+v", params.Size)
	}
	if params.Tags == nil || !reflect.DeepEqual(params.Tags.All, []string{"approved"}) {
		t.Errorf("tags = %+v", params.Tags)
	}
	wantSort := []file.SearchSort{file.SortByScore, file.SortByUploadedAtDesc}
	if !reflect.DeepEqual(params.Sort, wantSort) {
		t.Errorf("sort = %v, want %v", params.Sort, wantSort)
	}
}

func TestMapSearchMatch(t *testing.T) {
	match := file.SearchMatch{
		ID: "a1b2c3d4-e5f6-7890-abcd-ef1234567890", OriginalFileName: "invoice.pdf",
		Size: 123, MimeType: "application/pdf", IsReady: true,
		Metadata: map[string]string{"customer": "Acme"}, Tags: []string{"approved"},
		AppData: map[string]json.RawMessage{"scan": json.RawMessage(`{"safe":true}`)},
		Highlight: &file.SearchHighlight{
			OriginalFileName: []string{"<em>invoice</em>.pdf"},
			Metadata:         map[string]string{"customer": "<em>Acme</em>"},
		},
	}

	got := mapSearchMatch(match)
	if got.UUID != match.ID || got.Filename != "invoice.pdf" || !reflect.DeepEqual(got.Tags, []string{"approved"}) {
		t.Fatalf("mapped file = %+v", got)
	}
	if got.Highlight == nil || !reflect.DeepEqual(got.Highlight.OriginalFilename, []string{"<em>invoice</em>.pdf"}) {
		t.Errorf("highlight = %+v", got.Highlight)
	}
	if len(got.AppData) == 0 {
		t.Error("appdata was not mapped")
	}

	untagged := mapSearchMatch(file.SearchMatch{ID: match.ID})
	if untagged.Tags == nil {
		t.Error("nil tags were not normalized to an empty slice")
	}
}

func TestMapTagResult(t *testing.T) {
	got := mapTagResult(tag.Result{Tags: []string{"approved"}, Added: []string{"approved"}, Deleted: []string{"draft"}})
	if !reflect.DeepEqual(got.Tags, []string{"approved"}) || !reflect.DeepEqual(got.Deleted, []string{"draft"}) {
		t.Fatalf("mapped result = %+v", got)
	}
}
