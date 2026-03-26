package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/projectapi"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

type usageService struct {
	sdk     projectapi.Service
	verbose *output.VerboseLogger
}

// NewUsageService creates a service.UsageService backed by the Project API.
func NewUsageService(token string, httpClient *http.Client, verbose *output.VerboseLogger) (service.UsageService, error) {
	conf := &ucare.Config{
		HTTPClient: httpClient,
	}
	client, err := ucare.NewBearerClient(token, conf)
	if err != nil {
		return nil, err
	}
	return &usageService{
		sdk:     projectapi.NewService(client),
		verbose: verbose,
	}, nil
}

func (s *usageService) Combined(ctx context.Context, pubKey string, from, to string) (*service.UsageResult, error) {
	combined, err := s.sdk.GetUsage(ctx, pubKey, projectapi.UsageDateRange{
		From: from,
		To:   to,
	})
	if err != nil {
		return nil, wrapUsageAPIError(err, pubKey)
	}

	result := &service.UsageResult{
		Units: combined.Units,
		Data:  make([]service.UsageDayMetrics, len(combined.Data)),
	}
	for i, d := range combined.Data {
		result.Data[i] = service.UsageDayMetrics{
			Date:       d.Date,
			Traffic:    d.Traffic,
			Storage:    d.Storage,
			Operations: d.Operations,
		}
	}
	return result, nil
}

func (s *usageService) Metric(ctx context.Context, pubKey, metric string, from, to string) (*service.MetricResult, error) {
	m, err := s.sdk.GetUsageMetric(ctx, pubKey, metric, projectapi.UsageDateRange{
		From: from,
		To:   to,
	})
	if err != nil {
		return nil, wrapUsageAPIError(err, pubKey)
	}

	result := &service.MetricResult{
		Metric: m.Metric,
		Unit:   m.Unit,
		Data:   make([]service.MetricDayData, len(m.Data)),
	}
	for i, d := range m.Data {
		result.Data[i] = service.MetricDayData{
			Date:  d.Date,
			Value: d.Value,
		}
	}
	return result, nil
}

func wrapUsageAPIError(err error, pubKey string) error {
	var apiErr ucare.ProjectAPIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 400:
			return fmt.Errorf("%w\nhint: ensure --to is before today (UTC) and the date range is at most 90 days", err)
		case 404:
			return fmt.Errorf("project %q not found", pubKey)
		}
	}
	return err
}

var _ service.UsageService = (*usageService)(nil)
