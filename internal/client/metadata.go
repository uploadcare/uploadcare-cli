package client

import (
	"context"
	"net/http"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/metadata"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

type metadataService struct {
	sdk     metadata.Service
	verbose *output.VerboseLogger
}

// NewMetadataService creates a service.MetadataService backed by the Uploadcare SDK.
func NewMetadataService(publicKey, secretKey string, httpClient *http.Client, verbose *output.VerboseLogger) (service.MetadataService, error) {
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
	return &metadataService{
		sdk:     metadata.NewService(client),
		verbose: verbose,
	}, nil
}

func (s *metadataService) List(ctx context.Context, fileUUID string) (map[string]string, error) {
	return s.sdk.List(ctx, fileUUID)
}

func (s *metadataService) Get(ctx context.Context, fileUUID, key string) (string, error) {
	return s.sdk.Get(ctx, fileUUID, key)
}

func (s *metadataService) Set(ctx context.Context, fileUUID, key, value string) error {
	_, err := s.sdk.Set(ctx, fileUUID, key, value)
	return err
}

func (s *metadataService) Delete(ctx context.Context, fileUUID, key string) error {
	return s.sdk.Delete(ctx, fileUUID, key)
}

var _ service.MetadataService = (*metadataService)(nil)
