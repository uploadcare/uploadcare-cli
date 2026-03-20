package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
	"github.com/uploadcare/uploadcare-go/v2/webhook"
)

type webhookService struct {
	sdk     webhook.Service
	verbose *output.VerboseLogger
}

// NewWebhookService creates a service.WebhookService backed by the Uploadcare SDK.
func NewWebhookService(publicKey, secretKey string, httpClient *http.Client, verbose *output.VerboseLogger) (service.WebhookService, error) {
	creds := ucare.APICreds{
		PublicKey: publicKey,
		SecretKey: secretKey,
	}
	conf := &ucare.Config{
		APIVersion:             ucare.APIv07,
		SignBasedAuthentication: true,
		HTTPClient:             httpClient,
	}
	client, err := ucare.NewClient(creds, conf)
	if err != nil {
		return nil, err
	}
	return &webhookService{
		sdk:     webhook.NewService(client),
		verbose: verbose,
	}, nil
}

func (s *webhookService) List(ctx context.Context) ([]service.Webhook, error) {
	infos, err := s.sdk.List(ctx)
	if err != nil {
		return nil, err
	}
	var result []service.Webhook
	for _, info := range infos {
		result = append(result, mapWebhookInfo(info))
	}
	return result, nil
}

func (s *webhookService) Create(ctx context.Context, params service.WebhookCreateParams) (*service.Webhook, error) {
	sdkParams := webhook.Params{
		TargetURL: ucare.String(params.TargetURL),
		Event:     ucare.String(params.Event),
		IsActive:  ucare.Bool(params.IsActive),
	}
	if params.SigningSecret != "" {
		sdkParams.SigningSecret = ucare.String(params.SigningSecret)
	}
	info, err := s.sdk.Create(ctx, sdkParams)
	if err != nil {
		return nil, err
	}
	w := mapWebhookInfo(info)
	return &w, nil
}

func (s *webhookService) Update(ctx context.Context, id string, params service.WebhookUpdateParams) (*service.Webhook, error) {
	intID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid webhook ID %q: %w", id, err)
	}
	sdkParams := webhook.Params{
		ID: &intID,
	}
	if params.TargetURL != nil {
		sdkParams.TargetURL = params.TargetURL
	}
	if params.Event != nil {
		sdkParams.Event = params.Event
	}
	if params.IsActive != nil {
		sdkParams.IsActive = params.IsActive
	}
	if params.SigningSecret != nil {
		sdkParams.SigningSecret = params.SigningSecret
	}
	info, err := s.sdk.Update(ctx, sdkParams)
	if err != nil {
		return nil, err
	}
	w := mapWebhookInfo(info)
	return &w, nil
}

func (s *webhookService) Delete(ctx context.Context, id string) error {
	intID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid webhook ID %q: %w", id, err)
	}
	return s.sdk.Delete(ctx, intID)
}

func mapWebhookInfo(info webhook.Info) service.Webhook {
	w := service.Webhook{
		ID:        int(info.ID),
		TargetURL: info.TargetURL,
		Event:     info.Event,
		IsActive:  info.IsActive,
	}
	if info.SigningSecret != nil {
		w.SigningSecret = *info.SigningSecret
	}
	if info.CreatedAt != nil {
		w.DatetimeCreated = info.CreatedAt.Time
	}
	if info.UpdatedAt != nil {
		w.DatetimeUpdated = info.UpdatedAt.Time
	}
	return w
}

var _ service.WebhookService = (*webhookService)(nil)
