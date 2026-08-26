package client

import (
	"context"
	"net/http"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/tag"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

type tagService struct {
	sdk tag.Service
}

// NewTagService creates a service.TagService backed by the Uploadcare SDK.
func NewTagService(publicKey, secretKey string, httpClient *http.Client, _ *output.VerboseLogger) (service.TagService, error) {
	creds := ucare.APICreds{PublicKey: publicKey, SecretKey: secretKey}
	conf, err := ucare.NewConfig(creds,
		ucare.WithSignBasedAuthentication(),
		ucare.WithHTTPClient(httpClient),
		ucare.WithUserAgent(UserAgent),
	)
	if err != nil {
		return nil, err
	}
	sdkClient, err := ucare.NewClient(creds, conf)
	if err != nil {
		return nil, err
	}
	return &tagService{sdk: tag.NewService(sdkClient)}, nil
}

func (s *tagService) List(ctx context.Context, fileUUID string) ([]string, error) {
	return s.sdk.List(ctx, fileUUID)
}

func (s *tagService) Replace(ctx context.Context, fileUUID string, tags []string) (*service.TagChangeResult, error) {
	result, err := s.sdk.Replace(ctx, fileUUID, tags)
	if err != nil {
		return nil, err
	}
	return mapTagResult(result), nil
}

func (s *tagService) Update(ctx context.Context, fileUUID string, opts service.TagUpdateOptions) (*service.TagChangeResult, error) {
	result, err := s.sdk.Update(ctx, fileUUID, tag.UpdateParams{Add: opts.Add, Delete: opts.Delete})
	if err != nil {
		return nil, err
	}
	return mapTagResult(result), nil
}

func mapTagResult(result tag.Result) *service.TagChangeResult {
	return &service.TagChangeResult{
		Tags:    append([]string{}, result.Tags...),
		Added:   append([]string{}, result.Added...),
		Deleted: append([]string{}, result.Deleted...),
	}
}

var _ service.TagService = (*tagService)(nil)
