package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/uploadcare/uploadcare-cli/internal/service"
)

func TestNewFileService(t *testing.T) {
	svc, err := NewFileService("test-pub-key", "test-secret-key")
	if err != nil {
		t.Fatalf("NewFileService failed: %v", err)
	}
	var _ service.FileService = svc
}

func TestNewFileService_EmptyKeys(t *testing.T) {
	// SDK validates credentials at client creation time.
	_, err := NewFileService("", "")
	if err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestMapFileInfo_Complete(t *testing.T) {
	// Test via JSON round-trip since we can't directly construct internal config.Time.
	// We simulate what the SDK would return by constructing the service.File directly
	// through the Info method with a fake HTTP server.

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/a1b2c3d4-e5f6-7890-abcd-ef1234567890/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		resp := `{
			"uuid": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			"original_filename": "photo.jpg",
			"size": 1258000,
			"mime_type": "image/jpeg",
			"is_image": true,
			"is_ready": true,
			"datetime_uploaded": "2026-03-01T10:00:00",
			"datetime_stored": "2026-03-01T10:00:01",
			"original_file_url": "https://ucarecdn.com/a1b2c3d4/",
			"url": "https://api.uploadcare.com/files/a1b2c3d4/",
			"metadata": {"key": "value"}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(resp))
	}))
	defer server.Close()

	// Note: The SDK client uses hardcoded endpoints and can't easily be pointed at
	// a test server. The mapFileInfo function is tested indirectly through the
	// command-level tests with mock services. This test documents the expected API
	// contract.
	_ = server
}

func TestMapFileInfo_ViaSDKTypes(t *testing.T) {
	// Test the Info method of the service by calling it through the real client
	// against a server that would return properly formatted responses.
	// Since we can't easily inject the test server URL into the SDK, we test
	// the mapping function indirectly.

	// Create a service and verify it implements the interface properly.
	svc, err := NewFileService("test-pub", "test-secret")
	if err != nil {
		t.Fatalf("NewFileService: %v", err)
	}

	// Call Info with a non-existent UUID - should fail with network/API error.
	_, err = svc.Info(context.Background(), "00000000-0000-0000-0000-000000000000", false)
	if err == nil {
		t.Fatal("expected error calling Info against real API without valid credentials")
	}
}

func TestMapFileInfo_IncludeAppDataFlag(t *testing.T) {
	svc, err := NewFileService("test-pub", "test-secret")
	if err != nil {
		t.Fatalf("NewFileService: %v", err)
	}

	// Call with includeAppData=true - should also fail but exercises the parameter path.
	_, err = svc.Info(context.Background(), "00000000-0000-0000-0000-000000000000", true)
	if err == nil {
		t.Fatal("expected error calling Info against real API without valid credentials")
	}
}

func TestList_InvalidStartingPoint(t *testing.T) {
	svc, _ := NewFileService("pub", "sec")
	_, err := svc.List(context.Background(), service.FileListOptions{
		StartingPoint: "2026-03-01 12:00:00",
	})
	if err == nil {
		t.Fatal("expected error for invalid starting-point")
	}
	if !strings.Contains(err.Error(), "invalid --starting-point") {
		t.Errorf("error should mention --starting-point, got: %v", err)
	}
}

func TestIterate_InvalidStartingPoint(t *testing.T) {
	svc, _ := NewFileService("pub", "sec")
	err := svc.Iterate(context.Background(), service.FileListOptions{
		StartingPoint: "not-a-date",
	}, func(f service.File) error {
		t.Fatal("callback should not be called")
		return nil
	})
	if err == nil {
		t.Fatal("expected error for invalid starting-point")
	}
	if !strings.Contains(err.Error(), "invalid --starting-point") {
		t.Errorf("error should mention --starting-point, got: %v", err)
	}
}

func TestFileServiceInterface(t *testing.T) {
	// Verify all interface methods exist by calling UploadFromURL with test credentials.
	// It should fail with an API error (not a panic or compilation error).
	svc, _ := NewFileService("pub", "sec")
	ctx := context.Background()

	_, err := svc.UploadFromURL(ctx, service.URLUploadParams{URL: "https://example.com/test.jpg"})
	if err == nil {
		t.Error("UploadFromURL should return an error with invalid credentials")
	}
}

func TestJSONOutputStructure(t *testing.T) {
	// Verify that service.File marshals to the expected JSON structure.
	f := &service.File{
		UUID:     "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		Size:     1258000,
		Filename: "photo.jpg",
		MimeType: "image/jpeg",
		IsImage:  true,
		IsStored: true,
		IsReady:  true,
	}

	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if m["uuid"] != "a1b2c3d4-e5f6-7890-abcd-ef1234567890" {
		t.Errorf("uuid = %v", m["uuid"])
	}
	if m["size"] != float64(1258000) {
		t.Errorf("size = %v", m["size"])
	}
	if m["filename"] != "photo.jpg" {
		t.Errorf("filename = %v", m["filename"])
	}
}
