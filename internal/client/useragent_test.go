package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/uploadcare/uploadcare-cli/internal/service"
)

// headerCapturingTransport records request headers, keyed by host, from each request.
type headerCapturingTransport struct {
	mu      sync.Mutex
	headers map[string]http.Header
}

func (t *headerCapturingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	if t.headers == nil {
		t.headers = make(map[string]http.Header)
	}
	if _, ok := t.headers[req.Host]; !ok {
		t.headers[req.Host] = req.Header.Clone()
	}
	t.mu.Unlock()

	return &http.Response{
		StatusCode: 403,
		Body:       io.NopCloser(strings.NewReader(`{"detail":"forbidden"}`)),
		Header:     http.Header{"Content-Type": {"application/json"}},
	}, nil
}

func (t *headerCapturingTransport) get(host string) http.Header {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.headers[host]
}

func (t *headerCapturingTransport) first() http.Header {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, h := range t.headers {
		return h
	}
	return nil
}

func TestUserAgentHeader_RESTClient(t *testing.T) {
	prev := UserAgent
	UserAgent = "UploadcareCLI/1.2.3"
	defer func() { UserAgent = prev }()

	transport := &headerCapturingTransport{}
	httpClient := &http.Client{Transport: transport}

	svc, err := NewFileService("pub", "sec", "", httpClient, nil)
	if err != nil {
		t.Fatalf("NewFileService: %v", err)
	}

	// The request will fail (403) but the transport captures the User-Agent header.
	_, _ = svc.Info(context.Background(), "00000000-0000-0000-0000-000000000000", false)

	ua := transport.get("api.uploadcare.com").Get("User-Agent")
	if !strings.Contains(ua, "UploadcareCLI/1.2.3") {
		t.Errorf("User-Agent = %q, want it to contain %q", ua, "UploadcareCLI/1.2.3")
	}
	if !strings.Contains(ua, "UploadcareGo/") {
		t.Errorf("User-Agent = %q, want it to also contain SDK prefix", ua)
	}
}

func TestUserAgentHeader_UploadClient(t *testing.T) {
	prev := UserAgent
	UserAgent = "UploadcareCLI/1.2.3"
	defer func() { UserAgent = prev }()

	transport := &headerCapturingTransport{}
	httpClient := &http.Client{Transport: transport}

	svc, err := NewFileService("pub", "sec", "", httpClient, nil)
	if err != nil {
		t.Fatalf("NewFileService: %v", err)
	}

	// Upload triggers a request to upload.uploadcare.com.
	_, _ = svc.Upload(context.Background(), service.UploadParams{
		Data: strings.NewReader("test"),
		Name: "test.txt",
		Size: 4,
	})

	ua := transport.get("upload.uploadcare.com").Get("User-Agent")
	if !strings.Contains(ua, "UploadcareCLI/1.2.3") {
		t.Errorf("User-Agent = %q, want it to contain %q", ua, "UploadcareCLI/1.2.3")
	}
	if !strings.Contains(ua, "UploadcareGo/") {
		t.Errorf("User-Agent = %q, want it to also contain SDK prefix", ua)
	}
}

func TestUserAgentHeader_BearerClient(t *testing.T) {
	prev := UserAgent
	UserAgent = "UploadcareCLI/0.9.0"
	defer func() { UserAgent = prev }()

	transport := &headerCapturingTransport{}
	httpClient := &http.Client{Transport: transport}

	svc, err := NewProjectManagementService("test-token", httpClient, nil)
	if err != nil {
		t.Fatalf("NewProjectManagementService: %v", err)
	}

	_, _ = svc.List(context.Background(), service.ProjectListOptions{})

	ua := transport.first().Get("User-Agent")
	if !strings.Contains(ua, "UploadcareCLI/0.9.0") {
		t.Errorf("User-Agent = %q, want it to contain %q", ua, "UploadcareCLI/0.9.0")
	}
}
