package output

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewVerboseLogger_Disabled(t *testing.T) {
	v := NewVerboseLogger(false, nil)
	if v != nil {
		t.Error("NewVerboseLogger(false) should return nil")
	}
	if v.Enabled() {
		t.Error("nil VerboseLogger should not be enabled")
	}
}

func TestNewVerboseLogger_Enabled(t *testing.T) {
	var buf bytes.Buffer
	v := NewVerboseLogger(true, &buf)
	if v == nil {
		t.Fatal("NewVerboseLogger(true) should not return nil")
	}
	if !v.Enabled() {
		t.Error("VerboseLogger should be enabled")
	}
}

func TestVerboseLogger_NilSafety(t *testing.T) {
	var v *VerboseLogger
	// These should not panic.
	v.Request("GET", "https://api.example.com/files")
	v.Response("200 OK", 100*time.Millisecond)
	v.AuthSource("env")
	v.Retry(1, "503 Service Unavailable")
}

func TestVerboseLogger_Request(t *testing.T) {
	var buf bytes.Buffer
	v := NewVerboseLogger(true, &buf)
	v.Request("GET", "https://api.uploadcare.com/files/?limit=100")

	got := buf.String()
	want := "--> GET https://api.uploadcare.com/files/?limit=100\n"
	if got != want {
		t.Errorf("Request output = %q, want %q", got, want)
	}
}

func TestVerboseLogger_Response(t *testing.T) {
	var buf bytes.Buffer
	v := NewVerboseLogger(true, &buf)
	v.Response("200 OK", 127*time.Millisecond)

	got := buf.String()
	want := "<-- 200 OK (127ms)\n"
	if got != want {
		t.Errorf("Response output = %q, want %q", got, want)
	}
}

func TestVerboseLogger_AuthSource(t *testing.T) {
	var buf bytes.Buffer
	v := NewVerboseLogger(true, &buf)
	v.AuthSource("config:staging")

	got := buf.String()
	if !strings.Contains(got, "auth: config:staging") {
		t.Errorf("AuthSource output = %q, want to contain %q", got, "auth: config:staging")
	}
}

func TestVerboseLogger_Retry(t *testing.T) {
	var buf bytes.Buffer
	v := NewVerboseLogger(true, &buf)
	v.Retry(2, "429 Too Many Requests")

	got := buf.String()
	if !strings.Contains(got, "retry #2: 429 Too Many Requests") {
		t.Errorf("Retry output = %q, want to contain retry info", got)
	}
}

func TestVerboseLogger_FullSequence(t *testing.T) {
	var buf bytes.Buffer
	v := NewVerboseLogger(true, &buf)

	v.Request("GET", "https://api.uploadcare.com/files/")
	v.AuthSource("flag")
	v.Retry(1, "503 Service Unavailable")
	v.Response("200 OK", 250*time.Millisecond)

	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "-->") {
		t.Errorf("line 0 should start with -->, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "auth:") {
		t.Errorf("line 1 should contain auth, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "retry") {
		t.Errorf("line 2 should contain retry, got %q", lines[2])
	}
	if !strings.HasPrefix(lines[3], "<--") {
		t.Errorf("line 3 should start with <--, got %q", lines[3])
	}
}
