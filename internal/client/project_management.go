package client

import (
	"context"
	"net/http"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/projectapi"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

type projectManagementService struct {
	sdk     projectapi.Service
	verbose *output.VerboseLogger
}

// NewProjectManagementService creates a service.ProjectManagementService backed by the Project API.
func NewProjectManagementService(token string, httpClient *http.Client, verbose *output.VerboseLogger) (service.ProjectManagementService, error) {
	conf := &ucare.Config{
		HTTPClient: httpClient,
	}
	client, err := ucare.NewBearerClient(token, conf)
	if err != nil {
		return nil, err
	}
	return &projectManagementService{
		sdk:     projectapi.NewService(client),
		verbose: verbose,
	}, nil
}

func (s *projectManagementService) List(ctx context.Context, opts service.ProjectListOptions) (*service.ProjectListResult, error) {
	list, err := s.sdk.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	result := &service.ProjectListResult{}
	for list.Next() {
		p, err := list.ReadResult()
		if err != nil {
			return nil, err
		}
		result.Projects = append(result.Projects, mapProject(*p))
	}
	result.Total = len(result.Projects)

	return result, nil
}

func (s *projectManagementService) Get(ctx context.Context, pubKey string) (*service.ManagedProject, error) {
	p, err := s.sdk.Get(ctx, pubKey)
	if err != nil {
		return nil, err
	}
	mp := mapProject(p)
	return &mp, nil
}

func (s *projectManagementService) Create(ctx context.Context, params service.ProjectCreateParams) (*service.ManagedProject, error) {
	sdkParams := projectapi.CreateProjectParams{
		Name: params.Name,
	}
	if params.FilesizeLimit != nil || params.AutostoreEnabled != nil {
		sdkParams.Features = &projectapi.ProjectFeatures{
			Uploads: &projectapi.UploadSettings{
				FilesizeLimit: params.FilesizeLimit,
				Autostore:     params.AutostoreEnabled,
			},
		}
	}

	p, err := s.sdk.Create(ctx, sdkParams)
	if err != nil {
		return nil, err
	}
	mp := mapProject(p)
	return &mp, nil
}

func (s *projectManagementService) Update(ctx context.Context, pubKey string, params service.ProjectUpdateParams) (*service.ManagedProject, error) {
	sdkParams := projectapi.UpdateProjectParams{
		Name: params.Name,
	}
	if params.FilesizeLimit != nil || params.AutostoreEnabled != nil {
		sdkParams.Features = &projectapi.ProjectFeatures{
			Uploads: &projectapi.UploadSettings{
				FilesizeLimit: params.FilesizeLimit,
				Autostore:     params.AutostoreEnabled,
			},
		}
	}

	p, err := s.sdk.Update(ctx, pubKey, sdkParams)
	if err != nil {
		return nil, err
	}
	mp := mapProject(p)
	return &mp, nil
}

func (s *projectManagementService) Delete(ctx context.Context, pubKey string) error {
	return s.sdk.Delete(ctx, pubKey)
}

// mapProject flattens a projectapi.Project into a service.ManagedProject.
func mapProject(p projectapi.Project) service.ManagedProject {
	mp := service.ManagedProject{
		PubKey:               p.PubKey,
		Name:                 p.Name,
		IsBlocked:            p.IsBlocked,
		IsSearchIndexAllowed: p.IsSearchIndexAllowed,
		IsSharedProject:      p.IsSharedProject,
	}
	if p.Features != nil && p.Features.Uploads != nil {
		mp.FilesizeLimit = p.Features.Uploads.FilesizeLimit
		mp.AutostoreEnabled = p.Features.Uploads.Autostore
	}
	return mp
}

var _ service.ProjectManagementService = (*projectManagementService)(nil)
