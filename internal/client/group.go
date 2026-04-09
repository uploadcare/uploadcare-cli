package client

import (
	"context"
	"net/http"
	"time"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/group"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
	"github.com/uploadcare/uploadcare-go/v2/upload"
)

type groupService struct {
	sdkGroupSvc  group.Service
	sdkUploadSvc upload.Service
	verbose      *output.VerboseLogger
}

// NewGroupService creates a service.GroupService backed by the Uploadcare SDK.
func NewGroupService(publicKey, secretKey string, httpClient *http.Client, verbose *output.VerboseLogger) (service.GroupService, error) {
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
	return &groupService{
		sdkGroupSvc:  group.NewService(client),
		sdkUploadSvc: upload.NewService(client),
		verbose:      verbose,
	}, nil
}

func (s *groupService) List(ctx context.Context, opts service.GroupListOptions) (*service.GroupListResult, error) {
	params := group.ListParams{}
	if opts.Ordering != "" {
		params.OrderBy = ucare.String(opts.Ordering)
	}
	if opts.Limit > 0 {
		params.Limit = ucare.Uint64(uint64(opts.Limit))
	}
	if opts.StartingPoint != "" {
		t, err := time.Parse(time.RFC3339, opts.StartingPoint)
		if err != nil {
			return nil, err
		}
		params.StartingFrom = &t
	}

	list, err := s.sdkGroupSvc.List(ctx, params)
	if err != nil {
		return nil, err
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}

	var groups []service.Group
	for i := 0; i < limit && list.Next(); i++ {
		info, err := list.ReadResult()
		if err != nil {
			return nil, err
		}
		groups = append(groups, *mapGroupInfo(info))
	}

	return &service.GroupListResult{
		Groups: groups,
	}, nil
}

func (s *groupService) Iterate(ctx context.Context, opts service.GroupListOptions, fn func(service.Group) error) error {
	params := group.ListParams{}
	if opts.Ordering != "" {
		params.OrderBy = ucare.String(opts.Ordering)
	}
	if opts.Limit > 0 {
		params.Limit = ucare.Uint64(uint64(opts.Limit))
	}
	if opts.StartingPoint != "" {
		t, err := time.Parse(time.RFC3339, opts.StartingPoint)
		if err != nil {
			return err
		}
		params.StartingFrom = &t
	}

	list, err := s.sdkGroupSvc.List(ctx, params)
	if err != nil {
		return err
	}

	for list.Next() {
		info, err := list.ReadResult()
		if err != nil {
			return err
		}
		if err := fn(*mapGroupInfo(info)); err != nil {
			return err
		}
	}
	return nil
}

func (s *groupService) Info(ctx context.Context, groupID string) (*service.Group, error) {
	info, err := s.sdkGroupSvc.Info(ctx, groupID)
	if err != nil {
		return nil, err
	}
	return mapGroupInfo(&info), nil
}

func (s *groupService) Create(ctx context.Context, uuids []string) (*service.Group, error) {
	info, err := s.sdkUploadSvc.CreateGroup(ctx, uuids)
	if err != nil {
		return nil, err
	}
	return mapUploadGroupInfo(info), nil
}

func (s *groupService) Delete(ctx context.Context, groupID string) error {
	return s.sdkGroupSvc.Delete(ctx, groupID)
}

func mapGroupInfo(info *group.Info) *service.Group {
	g := &service.Group{
		ID:         info.ID,
		FilesCount: int(info.FileCount),
		CDNURL:     info.CDNLink,
	}
	if info.CreatedAt != nil {
		g.DatetimeCreated = info.CreatedAt.Time
	}
	if info.StoredAt != nil {
		t := info.StoredAt.Time
		g.DatetimeStored = &t
	}
	return g
}

func mapUploadGroupInfo(info upload.GroupInfo) *service.Group {
	g := &service.Group{
		ID:         info.ID,
		FilesCount: int(info.FileCount),
		CDNURL:     info.CDNLink,
	}
	if info.CreatedAt != nil {
		g.DatetimeCreated = info.CreatedAt.Time
	}
	if info.StoredAt != nil {
		t := info.StoredAt.Time
		g.DatetimeStored = &t
	}
	for _, f := range info.Files {
		g.Files = append(g.Files, service.File{
			UUID:     f.ID,
			Size:     int64(f.Size),
			Filename: f.OriginalFileName,
			MimeType: f.MimeType,
			IsImage:  f.IsImage,
			IsReady:  f.IsReady,
			IsStored: f.IsStored,
		})
	}
	return g
}

var _ service.GroupService = (*groupService)(nil)
