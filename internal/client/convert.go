package client

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-go/v2/conversion"
	"github.com/uploadcare/uploadcare-go/v2/ucare"
)

type convertService struct {
	sdk     conversion.Service
	verbose *output.VerboseLogger
}

// NewConvertService creates a service.ConvertService backed by the Uploadcare SDK.
func NewConvertService(publicKey, secretKey string, httpClient *http.Client, verbose *output.VerboseLogger) (service.ConvertService, error) {
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
	return &convertService{
		sdk:     conversion.NewService(client),
		verbose: verbose,
	}, nil
}

func (s *convertService) Document(ctx context.Context, params service.DocConvertParams) (*service.ConvertResult, error) {
	path := buildDocConvertPath(params)
	s.verbose.Infof("document conversion path: %s", path)

	sdkParams := conversion.Params{
		Paths: []string{path},
	}
	if params.Store {
		sdkParams.ToStore = ucare.String(conversion.ToStoreTrue)
	}
	if params.SaveInGroup {
		sdkParams.SaveInGroup = ucare.String("true")
	}

	result, err := s.sdk.Document(ctx, sdkParams)
	if err != nil {
		return nil, err
	}

	if len(result.Problems) > 0 {
		for p, msg := range result.Problems {
			return nil, fmt.Errorf("conversion problem for %s: %s", p, msg)
		}
	}

	if len(result.Jobs) == 0 {
		return nil, fmt.Errorf("no conversion job returned")
	}

	job := result.Jobs[0]
	return &service.ConvertResult{
		Token:  strconv.FormatInt(job.Token, 10),
		UUID:   job.ID,
		Status: "pending",
	}, nil
}

func (s *convertService) DocumentStatus(ctx context.Context, token string) (*service.ConvertStatus, error) {
	tokenInt, err := strconv.ParseInt(token, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid token %q: %w", token, err)
	}
	result, err := s.sdk.DocumentStatus(ctx, tokenInt)
	if err != nil {
		return nil, err
	}
	cs := &service.ConvertStatus{
		Status: result.Status,
	}
	if result.Error != nil {
		cs.Error = *result.Error
	}
	if result.Result.ID != "" {
		cs.ResultURL = result.Result.ID
	}
	return cs, nil
}

func (s *convertService) Video(ctx context.Context, params service.VideoConvertParams) (*service.ConvertResult, error) {
	path := buildVideoConvertPath(params)
	s.verbose.Infof("video conversion path: %s", path)

	sdkParams := conversion.Params{
		Paths: []string{path},
	}
	if params.Store {
		sdkParams.ToStore = ucare.String(conversion.ToStoreTrue)
	}

	result, err := s.sdk.Video(ctx, sdkParams)
	if err != nil {
		return nil, err
	}

	if len(result.Problems) > 0 {
		for p, msg := range result.Problems {
			return nil, fmt.Errorf("conversion problem for %s: %s", p, msg)
		}
	}

	if len(result.Jobs) == 0 {
		return nil, fmt.Errorf("no conversion job returned")
	}

	job := result.Jobs[0]
	return &service.ConvertResult{
		Token:  strconv.FormatInt(job.Token, 10),
		UUID:   job.ID,
		Status: "pending",
	}, nil
}

func (s *convertService) VideoStatus(ctx context.Context, token string) (*service.ConvertStatus, error) {
	tokenInt, err := strconv.ParseInt(token, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid token %q: %w", token, err)
	}
	result, err := s.sdk.VideoStatus(ctx, tokenInt)
	if err != nil {
		return nil, err
	}
	cs := &service.ConvertStatus{
		Status: result.Status,
	}
	if result.Error != nil {
		cs.Error = *result.Error
	}
	if result.Result.ID != "" {
		cs.ResultURL = result.Result.ID
	}
	return cs, nil
}

func buildDocConvertPath(p service.DocConvertParams) string {
	path := p.UUID + "/document/-/format/" + p.Format + "/"
	if p.Page != nil {
		path += fmt.Sprintf("-/page/%d/", *p.Page)
	}
	return path
}

func buildVideoConvertPath(p service.VideoConvertParams) string {
	path := p.UUID + "/video/"
	if p.Format != "" {
		path += "-/format/" + p.Format + "/"
	}
	if p.Size != "" {
		path += "-/resize/" + p.Size + "/"
	}
	if p.ResizeMode != "" {
		path += "-/resize/" + p.ResizeMode + "/"
	}
	if p.Quality != "" {
		path += "-/quality/" + p.Quality + "/"
	}
	if p.Cut != "" {
		path += "-/cut/" + p.Cut + "/"
	}
	if p.Thumbs != nil {
		path += fmt.Sprintf("-/thumbs~%d/", *p.Thumbs)
	}
	return path
}

var _ service.ConvertService = (*convertService)(nil)
