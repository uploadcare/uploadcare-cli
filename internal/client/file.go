package client

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/file"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

type fileService struct {
	sdkFileSvc file.Service
}

// NewFileService creates a service.FileService backed by the Uploadcare SDK.
func NewFileService(publicKey, secretKey string) (service.FileService, error) {
	creds := ucare.APICreds{
		PublicKey: publicKey,
		SecretKey: secretKey,
	}
	conf := &ucare.Config{
		APIVersion:               ucare.APIv07,
		SignBasedAuthentication:   true,
	}
	client, err := ucare.NewClient(creds, conf)
	if err != nil {
		return nil, err
	}
	return &fileService{
		sdkFileSvc: file.NewService(client),
	}, nil
}

func (s *fileService) Info(ctx context.Context, uuid string, includeAppData bool) (*service.File, error) {
	var params *file.InfoParams
	if includeAppData {
		params = &file.InfoParams{Include: ucare.String("appdata")}
	}

	info, err := s.sdkFileSvc.Info(ctx, uuid, params)
	if err != nil {
		return nil, err
	}
	return mapFileInfo(info), nil
}

func mapFileInfo(info file.Info) *service.File {
	f := &service.File{
		UUID:     info.BasicFileInfo.ID,
		Size:     int64(info.BasicFileInfo.Size),
		Filename: info.BasicFileInfo.OriginalFileName,
		MimeType: info.BasicFileInfo.MimeType,
		IsImage:  info.BasicFileInfo.IsImage,
		IsReady:  info.BasicFileInfo.IsReady,
		IsStored: info.StoredAt != nil,
		URL:      info.URL,
		Metadata: info.Metadata,
	}

	if info.OriginalFileURL != nil {
		f.OriginalFileURL = *info.OriginalFileURL
	}

	if info.UploadedAt != nil {
		f.DatetimeUploaded = info.UploadedAt.Time
	}
	if info.StoredAt != nil {
		t := info.StoredAt.Time
		f.DatetimeStored = &t
	}
	if info.RemovedAt != nil {
		t := info.RemovedAt.Time
		f.DatetimeRemoved = &t
	}

	if len(info.AppData) > 0 {
		b, err := json.Marshal(info.AppData)
		if err == nil {
			f.AppData = json.RawMessage(b)
		}
	}

	return f
}

func (s *fileService) List(ctx context.Context, opts service.FileListOptions) (*service.FileListResult, error) {
	return nil, errors.New("not implemented")
}

func (s *fileService) Upload(ctx context.Context, params service.UploadParams) (*service.File, error) {
	return nil, errors.New("not implemented")
}

func (s *fileService) UploadFromURL(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
	return nil, errors.New("not implemented")
}

func (s *fileService) Store(ctx context.Context, uuids []string) ([]service.File, error) {
	return nil, errors.New("not implemented")
}

func (s *fileService) Delete(ctx context.Context, uuids []string) ([]service.File, error) {
	return nil, errors.New("not implemented")
}

func (s *fileService) LocalCopy(ctx context.Context, params service.LocalCopyParams) (*service.File, error) {
	return nil, errors.New("not implemented")
}

func (s *fileService) RemoteCopy(ctx context.Context, params service.RemoteCopyParams) (*service.RemoteCopyResult, error) {
	return nil, errors.New("not implemented")
}

// Ensure compile-time interface satisfaction.
var _ service.FileService = (*fileService)(nil)
