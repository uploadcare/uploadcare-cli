package client

import (
	"net/http"
	"time"

	"github.com/uploadcare/uploadcare-cli/internal/output"
)

// verboseTransport wraps an http.RoundTripper and logs request/response
// details via a VerboseLogger.
type verboseTransport struct {
	base   http.RoundTripper
	logger *output.VerboseLogger
}

func (t *verboseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.logger.Request(req.Method, req.URL.String())
	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	t.logger.Response(resp.Status, time.Since(start))
	return resp, nil
}

// NewVerboseHTTPClient returns an *http.Client that logs HTTP traffic
// to the given VerboseLogger. Returns nil when the logger is not enabled.
func NewVerboseHTTPClient(logger *output.VerboseLogger) *http.Client {
	if !logger.Enabled() {
		return nil
	}
	base := http.DefaultTransport
	return &http.Client{
		Transport: &verboseTransport{base: base, logger: logger},
	}
}
