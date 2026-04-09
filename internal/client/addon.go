package client

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/addon"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

type addonService struct {
	sdk     addon.Service
	verbose *output.VerboseLogger
}

// NewAddonService creates a service.AddonService backed by the Uploadcare SDK.
func NewAddonService(publicKey, secretKey string, httpClient *http.Client, verbose *output.VerboseLogger) (service.AddonService, error) {
	creds := ucare.APICreds{
		PublicKey: publicKey,
		SecretKey: secretKey,
	}
	conf := &ucare.Config{
		APIVersion:             ucare.APIv07,
		SignBasedAuthentication: true,
		HTTPClient:             httpClient,
		UserAgent:              UserAgent,
	}
	client, err := ucare.NewClient(creds, conf)
	if err != nil {
		return nil, err
	}
	return &addonService{
		sdk:     addon.NewService(client),
		verbose: verbose,
	}, nil
}

func (s *addonService) Execute(ctx context.Context, addonName, fileUUID string, params json.RawMessage) (*service.AddonResult, error) {
	execParams := addon.ExecuteParams{
		Target: fileUUID,
	}
	if len(params) > 0 {
		var p interface{}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		execParams.Params = p
	}

	s.verbose.Infof("executing addon %s on file %s", addonName, fileUUID)
	result, err := s.sdk.Execute(ctx, addonName, execParams)
	if err != nil {
		return nil, err
	}
	return &service.AddonResult{
		RequestID: result.RequestID,
		Status:    "in_progress",
	}, nil
}

func (s *addonService) Status(ctx context.Context, addonName, requestID string) (*service.AddonStatus, error) {
	result, err := s.sdk.Status(ctx, addonName, requestID)
	if err != nil {
		return nil, err
	}
	return &service.AddonStatus{
		Status: result.Status,
		Result: result.Result,
	}, nil
}

var _ service.AddonService = (*addonService)(nil)
