package client

import (
	"context"
	"net/http"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/projectapi"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

type secretService struct {
	sdk     projectapi.Service
	verbose *output.VerboseLogger
}

// NewSecretService creates a service.SecretService backed by the Project API.
func NewSecretService(token string, httpClient *http.Client, verbose *output.VerboseLogger) (service.SecretService, error) {
	conf := &ucare.Config{
		HTTPClient: httpClient,
	}
	client, err := ucare.NewBearerClient(token, conf)
	if err != nil {
		return nil, err
	}
	return &secretService{
		sdk:     projectapi.NewService(client),
		verbose: verbose,
	}, nil
}

func (s *secretService) List(ctx context.Context, pubKey string) ([]service.Secret, error) {
	list, err := s.sdk.ListSecrets(ctx, pubKey, nil)
	if err != nil {
		return nil, err
	}

	var secrets []service.Secret
	for list.Next() {
		item, err := list.ReadResult()
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, service.Secret{
			ID:         item.ID,
			Hint:       item.Hint,
			LastUsedAt: item.LastUsedAt,
		})
	}
	return secrets, nil
}

func (s *secretService) Create(ctx context.Context, pubKey string) (*service.SecretCreateResult, error) {
	revealed, err := s.sdk.CreateSecret(ctx, pubKey)
	if err != nil {
		return nil, err
	}
	return &service.SecretCreateResult{
		ID:     revealed.ID,
		Secret: revealed.Secret,
	}, nil
}

func (s *secretService) Delete(ctx context.Context, pubKey, secretID string) error {
	return s.sdk.DeleteSecret(ctx, pubKey, secretID)
}

var _ service.SecretService = (*secretService)(nil)
