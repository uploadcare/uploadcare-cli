package client

import (
	"context"
	"net/http"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/project"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

type projectService struct {
	sdk     project.Service
	verbose *output.VerboseLogger
}

// NewProjectService creates a service.ProjectService backed by the Uploadcare REST API.
func NewProjectService(publicKey, secretKey string, httpClient *http.Client, verbose *output.VerboseLogger) (service.ProjectService, error) {
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
	return &projectService{
		sdk:     project.NewService(client),
		verbose: verbose,
	}, nil
}

func (s *projectService) Info(ctx context.Context) (*service.Project, error) {
	info, err := s.sdk.Info(ctx)
	if err != nil {
		return nil, err
	}

	collabs := make([]service.Collaborator, len(info.Collaborators))
	for i, c := range info.Collaborators {
		collabs[i] = service.Collaborator{
			Email: c.Email,
			Name:  c.Name,
		}
	}

	return &service.Project{
		Name:             info.Name,
		PubKey:           info.PubKey,
		Collaborators:    collabs,
		AutostoreEnabled: info.AutostoreEnabled,
	}, nil
}

var _ service.ProjectService = (*projectService)(nil)
