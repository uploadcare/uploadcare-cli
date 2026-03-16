package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/file"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
	"github.com/uploadcare/uploadcare-go/v2/upload"
)

type fileService struct {
	sdkFileSvc    file.Service
	sdkUploadSvc  upload.Service
}

// NewFileService creates a service.FileService backed by the Uploadcare SDK.
func NewFileService(publicKey, secretKey string) (service.FileService, error) {
	creds := ucare.APICreds{
		PublicKey: publicKey,
		SecretKey: secretKey,
	}
	conf := &ucare.Config{
		APIVersion:             ucare.APIv07,
		SignBasedAuthentication: true,
	}
	client, err := ucare.NewClient(creds, conf)
	if err != nil {
		return nil, err
	}
	return &fileService{
		sdkFileSvc:   file.NewService(client),
		sdkUploadSvc: upload.NewService(client),
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

func mapUploadFileInfo(info upload.FileInfo) *service.File {
	return &service.File{
		UUID:     info.BasicFileInfo.ID,
		Size:     int64(info.BasicFileInfo.Size),
		Filename: info.BasicFileInfo.OriginalFileName,
		MimeType: info.BasicFileInfo.MimeType,
		IsImage:  info.BasicFileInfo.IsImage,
		IsReady:  info.BasicFileInfo.IsReady,
		IsStored: info.IsStored,
	}
}

func (s *fileService) List(ctx context.Context, opts service.FileListOptions) (*service.FileListResult, error) {
	params, err := buildListParams(opts)
	if err != nil {
		return nil, err
	}
	list, err := s.sdkFileSvc.List(ctx, params)
	if err != nil {
		return nil, err
	}

	var files []service.File
	limit := opts.Limit
	if limit <= 0 {
		limit = 100
	}
	for i := 0; i < limit && list.Next(); i++ {
		info, err := list.ReadResult()
		if err != nil {
			return nil, err
		}
		files = append(files, *mapFileInfo(*info))
	}

	return &service.FileListResult{
		Files: files,
		Total: len(files),
	}, nil
}

func (s *fileService) Iterate(ctx context.Context, opts service.FileListOptions, fn func(service.File) error) error {
	params, err := buildListParams(opts)
	if err != nil {
		return err
	}
	list, err := s.sdkFileSvc.List(ctx, params)
	if err != nil {
		return err
	}

	for list.Next() {
		info, err := list.ReadResult()
		if err != nil {
			return err
		}
		if err := fn(*mapFileInfo(*info)); err != nil {
			return err
		}
	}
	return nil
}

func buildListParams(opts service.FileListOptions) (file.ListParams, error) {
	params := file.ListParams{}
	if opts.Ordering != "" {
		params.OrderBy = ucare.String(opts.Ordering)
	}
	if opts.Limit > 0 {
		params.Limit = ucare.Uint64(uint64(opts.Limit))
	}
	if opts.StartingPoint != "" {
		t, err := time.Parse(time.RFC3339, opts.StartingPoint)
		if err != nil {
			return params, fmt.Errorf("invalid --starting-point value: %w", err)
		}
		params.StartingFrom = &t
	}
	if opts.Stored != nil {
		params.Stored = opts.Stored
	}
	if opts.Removed {
		params.Removed = ucare.Bool(true)
	}
	if opts.IncludeAppData {
		params.Include = ucare.String("appdata")
	}
	return params, nil
}

func (s *fileService) Upload(ctx context.Context, params service.UploadParams) (*service.File, error) {
	var toStore *string
	switch params.Store {
	case "true":
		toStore = ucare.String(upload.ToStoreTrue)
	case "false":
		toStore = ucare.String(upload.ToStoreFalse)
	case "auto", "":
		toStore = ucare.String(upload.ToStoreAuto)
	default:
		return nil, fmt.Errorf("invalid store value: %q (must be \"auto\", \"true\", or \"false\")", params.Store)
	}

	sdkParams := upload.UploadParams{
		Data:               params.Data,
		Name:               params.Name,
		Size:               params.Size,
		ContentType:        params.ContentType,
		ToStore:            toStore,
		Metadata:           params.Metadata,
		MultipartThreshold: params.MultipartThreshold,
	}

	uploadInfo, err := s.sdkUploadSvc.Upload(ctx, sdkParams)
	if err != nil {
		return nil, err
	}

	// The upload API returns only basic fields (no timestamps, URLs, or
	// metadata). Fetch the complete file info from the REST API.
	fileInfo, err := s.sdkFileSvc.Info(ctx, uploadInfo.BasicFileInfo.ID, nil)
	if err != nil {
		// Fall back to the partial upload response rather than failing
		// the whole operation — the file was uploaded successfully.
		return mapUploadFileInfo(uploadInfo), nil
	}
	return mapFileInfo(fileInfo), nil
}

func (s *fileService) UploadFromURL(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
	return nil, errors.New("not implemented")
}

func (s *fileService) Store(ctx context.Context, uuids []string) (*service.BatchResult, error) {
	batch, err := s.sdkFileSvc.BatchStore(ctx, uuids)
	if err != nil {
		return nil, err
	}
	return mapBatchInfo(batch), nil
}

func (s *fileService) Delete(ctx context.Context, uuids []string) (*service.BatchResult, error) {
	batch, err := s.sdkFileSvc.BatchDelete(ctx, uuids)
	if err != nil {
		return nil, err
	}
	return mapBatchInfo(batch), nil
}

func mapBatchInfo(batch file.BatchInfo) *service.BatchResult {
	result := &service.BatchResult{
		Problems: batch.Problems,
	}
	for _, info := range batch.Results {
		result.Files = append(result.Files, *mapFileInfo(info))
	}
	return result
}

func (s *fileService) LocalCopy(ctx context.Context, params service.LocalCopyParams) (*service.File, error) {
	storeVal := file.StoreFalse
	if params.Store {
		storeVal = file.StoreTrue
	}

	sdkParams := file.LocalCopyParams{
		Source: params.UUID,
		Store:  ucare.String(storeVal),
	}

	copyInfo, err := s.sdkFileSvc.LocalCopy(ctx, sdkParams)
	if err != nil {
		return nil, err
	}
	return mapFileInfo(copyInfo.Result), nil
}

func (s *fileService) RemoteCopy(ctx context.Context, params service.RemoteCopyParams) (*service.RemoteCopyResult, error) {
	sdkParams := file.RemoteCopyParams{
		Source: params.UUID,
		Target: params.Target,
	}
	if params.MakePublic {
		sdkParams.MakePublic = ucare.String(file.MakePublicTrue)
	} else {
		sdkParams.MakePublic = ucare.String(file.MakePublicFalse)
	}
	if params.Pattern != "" {
		sdkParams.Pattern = ucare.String(params.Pattern)
	}

	copyInfo, err := s.sdkFileSvc.RemoteCopy(ctx, sdkParams)
	if err != nil {
		return nil, err
	}

	result := &service.RemoteCopyResult{
		AlreadyExists: copyInfo.AlreadyExists,
	}
	if copyInfo.Result != nil {
		result.Result = *copyInfo.Result
	}
	return result, nil
}

// Ensure compile-time interface satisfaction.
var _ service.FileService = (*fileService)(nil)
