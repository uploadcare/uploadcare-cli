package service

import (
	"context"
	"encoding/json"
	"time"
)

// --- Domain types ---

// File represents an Uploadcare file resource.
type File struct {
	UUID             string            `json:"uuid"`
	Size             int64             `json:"size"`
	Filename         string            `json:"filename"`
	MimeType         string            `json:"mime_type"`
	IsImage          bool              `json:"is_image"`
	IsStored         bool              `json:"is_stored"`
	IsReady          bool              `json:"is_ready"`
	DatetimeUploaded time.Time         `json:"datetime_uploaded"`
	DatetimeStored   *time.Time        `json:"datetime_stored"`
	DatetimeRemoved  *time.Time        `json:"datetime_removed"`
	URL              string            `json:"url"`
	OriginalFileURL  string            `json:"original_file_url"`
	Metadata         map[string]string `json:"metadata"`
	AppData          json.RawMessage   `json:"appdata,omitempty"`
}

// FileListOptions specifies parameters for listing files.
type FileListOptions struct {
	Ordering       string
	Limit          int
	StartingPoint  string
	Stored         *bool
	Removed        bool
	IncludeAppData bool
}

// FileListResult is a paginated list of files.
type FileListResult struct {
	Files    []File `json:"results"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Total    int    `json:"total"`
}

// UploadParams configures a direct file upload.
type UploadParams struct {
	Path               string
	Store              string // "auto", "true", "false"
	Metadata           map[string]string
	MultipartThreshold int64
	ForceMultipart     bool
	ForceDirect        bool
}

// URLUploadParams configures an upload-from-URL operation.
type URLUploadParams struct {
	URL             string
	Store           string // "auto", "true", "false"
	Metadata        map[string]string
	Wait            bool
	Timeout         time.Duration
	CheckDuplicates bool
	SaveDuplicates  bool
}

// LocalCopyParams configures a local (same-storage) copy.
type LocalCopyParams struct {
	UUID  string
	Store bool
}

// RemoteCopyParams configures a remote-storage copy.
type RemoteCopyParams struct {
	UUID       string
	Target     string
	MakePublic bool
	Pattern    string
}

// RemoteCopyResult is the response from a remote copy operation.
type RemoteCopyResult struct {
	Type   string `json:"type"`
	Result string `json:"result"`
}

// Group represents an Uploadcare file group.
type Group struct {
	ID              string     `json:"id"`
	DatetimeCreated time.Time  `json:"datetime_created"`
	DatetimeStored  *time.Time `json:"datetime_stored"`
	FilesCount      int        `json:"files_count"`
	CDNURL          string     `json:"cdn_url"`
	URL             string     `json:"url"`
	Files           []File     `json:"files"`
}

// GroupListOptions specifies parameters for listing groups.
type GroupListOptions struct {
	Ordering      string
	Limit         int
	StartingPoint string
}

// GroupListResult is a paginated list of groups.
type GroupListResult struct {
	Groups   []Group `json:"results"`
	Next     string  `json:"next"`
	Previous string  `json:"previous"`
}

// DocConvertParams configures a document conversion.
type DocConvertParams struct {
	UUID        string
	Format      string
	Page        *int
	SaveInGroup bool
	Store       bool
}

// VideoConvertParams configures a video conversion.
type VideoConvertParams struct {
	UUID       string
	Format     string
	Size       string
	ResizeMode string
	Quality    string
	Cut        string
	Thumbs     *int
	Store      bool
}

// ConvertResult is the response from starting a conversion.
type ConvertResult struct {
	Token  string `json:"token"`
	UUID   string `json:"uuid"`
	Status string `json:"status"`
}

// ConvertStatus is the current status of a conversion job.
type ConvertStatus struct {
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	ResultURL string `json:"result,omitempty"`
}

// AddonResult is the response from executing an add-on.
type AddonResult struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
}

// AddonStatus is the current status of an add-on execution.
type AddonStatus struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result,omitempty"`
}

// Webhook represents a configured webhook.
type Webhook struct {
	ID              int       `json:"id"`
	TargetURL       string    `json:"target_url"`
	Event           string    `json:"event"`
	IsActive        bool      `json:"is_active"`
	SigningSecret   string    `json:"signing_secret,omitempty"`
	DatetimeCreated time.Time `json:"created"`
	DatetimeUpdated time.Time `json:"updated"`
}

// WebhookCreateParams configures webhook creation.
type WebhookCreateParams struct {
	TargetURL     string
	Event         string
	IsActive      bool
	SigningSecret string
}

// WebhookUpdateParams configures webhook updates. Nil fields are not sent.
type WebhookUpdateParams struct {
	TargetURL     *string
	Event         *string
	IsActive      *bool
	SigningSecret *string
}

// Project represents project info from the REST API.
type Project struct {
	Name             string         `json:"name"`
	PubKey           string         `json:"pub_key"`
	Collaborators    []Collaborator `json:"collaborators"`
	AutostoreEnabled bool           `json:"autostore_enabled"`
}

// Collaborator is a project collaborator.
type Collaborator struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// ManagedProject represents a project from the Project API.
type ManagedProject struct {
	PubKey           string    `json:"pub_key"`
	Name             string    `json:"name"`
	FilesizeLimit    *int64    `json:"filesize_limit"`
	AutostoreEnabled bool      `json:"autostore_enabled"`
	DatetimeCreated  time.Time `json:"created"`
	IsDeleted        bool      `json:"is_deleted"`
}

// ProjectListOptions specifies parameters for listing projects.
type ProjectListOptions struct {
	PageAll bool
}

// ProjectListResult is a paginated list of managed projects.
type ProjectListResult struct {
	Projects []ManagedProject `json:"results"`
	Next     string           `json:"next"`
	Previous string           `json:"previous"`
}

// ProjectCreateParams configures project creation.
type ProjectCreateParams struct {
	Name             string
	FilesizeLimit    *int64
	AutostoreEnabled *bool
}

// ProjectUpdateParams configures project updates.
type ProjectUpdateParams struct {
	Name             *string
	FilesizeLimit    *int64
	AutostoreEnabled *bool
}

// Secret represents an API secret (list view — hints only, not full keys).
type Secret struct {
	ID   string `json:"id"`
	Hint string `json:"hint"`
}

// SecretCreateResult is the response from creating a new secret.
type SecretCreateResult struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

// UsageResult is combined usage metrics for a project.
type UsageResult struct {
	Traffic    int64 `json:"traffic"`
	Storage    int64 `json:"storage"`
	Operations int64 `json:"operations"`
}

// MetricResult is a single usage metric with time series.
type MetricResult struct {
	Metric string           `json:"metric"`
	Total  int64            `json:"total"`
	Points map[string]int64 `json:"points"`
}

// MimeType represents an available MIME type.
type MimeType struct {
	MimeType string `json:"mime_type"`
	Category string `json:"category"`
}

// --- Service interfaces ---

// FileService provides file operations via the REST API.
type FileService interface {
	List(ctx context.Context, opts FileListOptions) (*FileListResult, error)
	Info(ctx context.Context, uuid string, includeAppData bool) (*File, error)
	Upload(ctx context.Context, params UploadParams) (*File, error)
	UploadFromURL(ctx context.Context, params URLUploadParams) (*File, error)
	Store(ctx context.Context, uuids []string) ([]File, error)
	Delete(ctx context.Context, uuids []string) ([]File, error)
	LocalCopy(ctx context.Context, params LocalCopyParams) (*File, error)
	RemoteCopy(ctx context.Context, params RemoteCopyParams) (*RemoteCopyResult, error)
}

// MetadataService provides file metadata CRUD operations.
type MetadataService interface {
	List(ctx context.Context, fileUUID string) (map[string]string, error)
	Get(ctx context.Context, fileUUID, key string) (string, error)
	Set(ctx context.Context, fileUUID, key, value string) error
	Delete(ctx context.Context, fileUUID, key string) error
}

// GroupService provides file group operations.
type GroupService interface {
	List(ctx context.Context, opts GroupListOptions) (*GroupListResult, error)
	Info(ctx context.Context, groupID string) (*Group, error)
	Create(ctx context.Context, uuids []string) (*Group, error)
	Delete(ctx context.Context, groupID string) error
}

// ConvertService provides document and video conversion operations.
type ConvertService interface {
	Document(ctx context.Context, params DocConvertParams) (*ConvertResult, error)
	DocumentStatus(ctx context.Context, token string) (*ConvertStatus, error)
	Video(ctx context.Context, params VideoConvertParams) (*ConvertResult, error)
	VideoStatus(ctx context.Context, token string) (*ConvertStatus, error)
}

// AddonService provides add-on execution operations.
type AddonService interface {
	Execute(ctx context.Context, addonName, fileUUID string, params json.RawMessage) (*AddonResult, error)
	Status(ctx context.Context, addonName, requestID string) (*AddonStatus, error)
}

// WebhookService provides webhook management operations.
type WebhookService interface {
	List(ctx context.Context) ([]Webhook, error)
	Create(ctx context.Context, params WebhookCreateParams) (*Webhook, error)
	Update(ctx context.Context, id string, params WebhookUpdateParams) (*Webhook, error)
	Delete(ctx context.Context, id string) error
}

// ProjectService provides project info via the REST API.
type ProjectService interface {
	Info(ctx context.Context) (*Project, error)
}

// ProjectManagementService provides project management via the Project API.
type ProjectManagementService interface {
	List(ctx context.Context, opts ProjectListOptions) (*ProjectListResult, error)
	Get(ctx context.Context, pubKey string) (*ManagedProject, error)
	Create(ctx context.Context, params ProjectCreateParams) (*ManagedProject, error)
	Update(ctx context.Context, pubKey string, params ProjectUpdateParams) (*ManagedProject, error)
	Delete(ctx context.Context, pubKey string) error
}

// SecretService provides API secret management via the Project API.
type SecretService interface {
	List(ctx context.Context, pubKey string) ([]Secret, error)
	Create(ctx context.Context, pubKey string) (*SecretCreateResult, error)
	Delete(ctx context.Context, pubKey, secretID string) error
}

// UsageService provides usage metrics via the Project API.
type UsageService interface {
	Combined(ctx context.Context, pubKey string, from, to time.Time) (*UsageResult, error)
	Metric(ctx context.Context, pubKey, metric string, from, to time.Time) (*MetricResult, error)
}

// MimeTypeService provides MIME type listing via the Project API.
type MimeTypeService interface {
	List(ctx context.Context) ([]MimeType, error)
}
