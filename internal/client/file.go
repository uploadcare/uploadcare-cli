package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/file"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
	"github.com/uploadcare/uploadcare-go/v2/upload"
)

type fileService struct {
	sdkFileSvc   file.Service
	sdkUploadSvc upload.Service
	httpClient   *http.Client
	verbose      *output.VerboseLogger
}

// NewFileService creates a service.FileService backed by the Uploadcare SDK.
// An optional httpClient can be provided to customise HTTP behaviour (e.g.
// verbose logging). Pass nil to use the default client.
func NewFileService(publicKey, secretKey, cdnBase string, httpClient *http.Client, verbose *output.VerboseLogger) (service.FileService, error) {
	creds := ucare.APICreds{
		PublicKey: publicKey,
		SecretKey: secretKey,
	}
	conf, err := ucare.NewConfig(creds,
		ucare.WithSignBasedAuthentication(),
		ucare.WithCDNBase(cdnBase),
		ucare.WithHTTPClient(httpClient),
		ucare.WithUserAgent(UserAgent),
	)
	if err != nil {
		return nil, err
	}
	client, err := ucare.NewClient(creds, conf)
	if err != nil {
		return nil, err
	}
	return &fileService{
		sdkFileSvc:   file.NewService(client),
		sdkUploadSvc: upload.NewService(client),
		httpClient:   httpClient,
		verbose:      verbose,
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
		UUID:     info.ID,
		Size:     int64(info.Size),
		Filename: info.OriginalFileName,
		MimeType: info.MimeType,
		IsImage:  info.IsImage,
		IsReady:  info.IsReady,
		IsStored: info.StoredAt != nil,
		URL:      info.URL,
		Metadata: info.Metadata,
		Tags:     info.Tags,
	}
	if f.Tags == nil {
		f.Tags = []string{}
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
		UUID:     info.ID,
		Size:     int64(info.Size),
		Filename: info.OriginalFileName,
		MimeType: info.MimeType,
		IsImage:  info.IsImage,
		IsReady:  info.IsReady,
		IsStored: info.IsStored,
		Tags:     []string{},
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

func (s *fileService) Search(ctx context.Context, opts service.FileSearchOptions) (*service.FileSearchResult, error) {
	search, err := s.sdkFileSvc.Search(ctx, buildSearchParams(opts))
	if err != nil {
		return nil, err
	}

	files := make([]service.File, 0, opts.Limit)
	for len(files) < opts.Limit && search.Next() {
		match, err := search.ReadResult()
		// Next() can report a pending next pointer for a page that turns
		// out empty (stale matches filtered server-side); that is clean
		// exhaustion, not an error.
		if errors.Is(err, ucare.ErrEndOfResults) {
			break
		}
		if err != nil {
			return nil, err
		}
		files = append(files, *mapSearchMatch(*match))
	}
	return &service.FileSearchResult{Files: files, Total: search.Total()}, nil
}

func (s *fileService) IterateSearch(ctx context.Context, opts service.FileSearchOptions, fn func(service.File) error) (uint64, error) {
	search, err := s.sdkFileSvc.Search(ctx, buildSearchParams(opts))
	if err != nil {
		return 0, err
	}
	total := search.Total()
	// The API serves at most the first MaxSearchOffsetLimit matches of a
	// search; stop cleanly at the window instead of following a next
	// cursor the server cannot satisfy.
	remaining := file.MaxSearchOffsetLimit - opts.Offset
	for read := 0; read < remaining && search.Next(); read++ {
		match, err := search.ReadResult()
		if errors.Is(err, ucare.ErrEndOfResults) {
			break
		}
		if err != nil {
			return total, err
		}
		if err := fn(*mapSearchMatch(*match)); err != nil {
			return total, err
		}
	}
	return total, nil
}

func buildSearchParams(opts service.FileSearchOptions) file.SearchParams {
	params := file.SearchParams{
		Limit:     ucare.Uint64(uint64(opts.Limit)),
		Offset:    ucare.Uint64(uint64(opts.Offset)),
		Query:     opts.Query,
		Exact:     opts.Exact,
		IsImage:   opts.IsImage,
		Fuzziness: opts.Fuzziness,
	}
	if opts.IncludeAppData {
		params.Include = ucare.String(file.SearchIncludeAppData)
	}
	if opts.Phrase != nil {
		params.Phrase = &file.SearchPhrase{
			OriginalFilename: opts.Phrase.OriginalFilename,
			Metadata:         opts.Phrase.Metadata,
			DetectedMimeType: opts.Phrase.DetectedMimeType,
		}
	}
	if opts.DatetimeUploaded != nil {
		params.DatetimeUploaded = &file.SearchDatetime{
			Gt: opts.DatetimeUploaded.Gt, Gte: opts.DatetimeUploaded.Gte,
			Lt: opts.DatetimeUploaded.Lt, Lte: opts.DatetimeUploaded.Lte,
		}
	}
	if opts.Size != nil {
		params.Size = &file.SearchSize{
			Gt: opts.Size.Gt, Gte: opts.Size.Gte,
			Lt: opts.Size.Lt, Lte: opts.Size.Lte,
		}
	}
	if opts.Tags != nil {
		params.Tags = &file.SearchTags{Any: opts.Tags.Any, All: opts.Tags.All, None: opts.Tags.None}
	}
	for _, sort := range opts.Sort {
		params.Sort = append(params.Sort, file.SearchSort(sort))
	}
	return params
}

func mapSearchMatch(match file.SearchMatch) *service.File {
	f := mapFileInfo(file.Info{
		BasicFileInfo: file.BasicFileInfo{
			ID:               match.ID,
			MimeType:         match.MimeType,
			OriginalFileName: match.OriginalFileName,
			Size:             match.Size,
			IsImage:          match.IsImage,
			IsReady:          match.IsReady,
		},
		RemovedAt:       match.RemovedAt,
		StoredAt:        match.StoredAt,
		UploadedAt:      match.UploadedAt,
		OriginalFileURL: match.OriginalFileURL,
		URL:             match.URL,
		Metadata:        match.Metadata,
		Tags:            match.Tags,
		AppData:         match.AppData,
	})
	if match.Highlight != nil {
		f.Highlight = &service.FileSearchHighlight{
			OriginalFilename: match.Highlight.OriginalFileName,
			DetectedMimeType: match.Highlight.DetectedMimeType,
			Metadata:         match.Highlight.Metadata,
		}
	}
	return f
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
		Tags:               params.Tags,
		MultipartThreshold: params.MultipartThreshold,
	}

	uploadInfo, err := s.sdkUploadSvc.Upload(ctx, sdkParams)
	if err != nil {
		return nil, err
	}

	// The upload API returns only basic fields (no timestamps, URLs, or
	// metadata). Fetch the complete file info from the REST API.
	return s.enrichUploadInfo(ctx, uploadInfo.ID, mapUploadFileInfo(uploadInfo)), nil
}

func (s *fileService) UploadFromURL(ctx context.Context, params service.URLUploadParams) (*service.File, error) {
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

	timeout := params.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sdkParams := upload.FromURLParams{
		URL:      params.URL,
		ToStore:  toStore,
		Metadata: params.Metadata,
		Tags:     params.Tags,
	}
	if params.CheckDuplicates {
		sdkParams.CheckURLDuplicates = ucare.String(upload.URLDuplicatesTrue)
	}
	if params.SaveDuplicates {
		sdkParams.SaveURLDuplicates = ucare.String(upload.URLDuplicatesTrue)
	}

	s.verbose.Infof("uploading from URL: %s", params.URL)
	res, err := s.sdkUploadSvc.FromURL(ctx, sdkParams)
	if err != nil {
		return nil, err
	}

	info, ok := res.Info()
	if ok {
		s.verbose.Info("from-url", "completed synchronously")
		return s.enrichUploadInfo(ctx, info.ID, mapUploadFileInfo(info)), nil
	}

	s.verbose.Infof("from-url: waiting for async processing (timeout %s)", timeout)
	select {
	case info = <-res.Done():
		s.verbose.Info("from-url", "async processing complete")
		return s.enrichUploadInfo(ctx, info.ID, mapUploadFileInfo(info)), nil
	case err = <-res.Error():
		return nil, err
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("upload from URL timed out after %s", timeout)
		}
		return nil, ctx.Err()
	}
}

func (s *fileService) enrichUploadInfo(ctx context.Context, id string, fallback *service.File) *service.File {
	s.verbose.Infof("fetching full file info for %s", id)
	fileInfo, err := s.sdkFileSvc.Info(ctx, id, nil)
	if err != nil {
		s.verbose.Infof("file info fetch failed, using upload response: %v", err)
		return fallback
	}
	return mapFileInfo(fileInfo)
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

func (s *fileService) Download(ctx context.Context, params service.DownloadParams) (*service.DownloadResult, error) {
	f := params.Resolved
	if f == nil {
		info, err := s.sdkFileSvc.Info(ctx, params.UUID, nil)
		if err != nil {
			return nil, err
		}
		f = mapFileInfo(info)
	}
	if f.OriginalFileURL == "" {
		return nil, fmt.Errorf("file %s has no original_file_url (likely removed)", params.UUID)
	}

	srcURL := f.OriginalFileURL
	if params.Effects != "" && f.IsImage {
		srcURL = service.ApplyCDNEffects(srcURL, f.UUID, params.Effects)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
	if err != nil {
		return nil, err
	}

	httpClient := s.httpClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	s.verbose.Infof("downloading %s -> writer (size %d, effects=%q, image=%v)", params.UUID, f.Size, params.Effects, f.IsImage)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download %s: unexpected status %d", srcURL, resp.StatusCode)
	}

	var src io.Reader = resp.Body
	if params.Progress != nil {
		total := f.Size
		if resp.ContentLength > 0 {
			total = resp.ContentLength
		}
		src = &progressReader{r: resp.Body, total: total, cb: params.Progress}
	}

	n, err := io.Copy(params.Out, src)
	if err != nil {
		return nil, err
	}
	return &service.DownloadResult{
		UUID:        params.UUID,
		BytesCopied: n,
		SourceURL:   srcURL,
	}, nil
}

// progressReader wraps an io.Reader, invoking a callback after each read.
type progressReader struct {
	r     io.Reader
	total int64
	soFar int64
	cb    func(bytesSoFar, totalBytes int64)
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	if n > 0 {
		p.soFar += int64(n)
		if p.cb != nil {
			p.cb(p.soFar, p.total)
		}
	}
	return n, err
}

// Ensure compile-time interface satisfaction.
var _ service.FileService = (*fileService)(nil)
