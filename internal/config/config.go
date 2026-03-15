package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Sentinel errors.
var (
	ErrNoProjectCredentials = errors.New("no project credentials found")
	ErrNoProjectAPIToken    = errors.New("no project API token found")
	ErrProjectNotFound      = errors.New("project not found in config")
)

// Default API base URLs.
const (
	DefaultRESTAPIBase    = "https://api.uploadcare.com"
	DefaultUploadAPIBase  = "https://upload.uploadcare.com"
	DefaultCDNBase        = "https://ucarecdn.com"
	DefaultProjectAPIBase = "https://api.uploadcare.com/apps/api/project-api/v1/"
)

// ConfigDir returns the path to ~/.uploadcare.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".uploadcare")
}

// ConfigPath returns the full path to the config file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// ProjectCredentials holds resolved project-level credentials.
// Fields may be empty — use RequirePublicKey or RequireBoth to validate
// that the credentials a command needs are present.
type ProjectCredentials struct {
	PublicKey string
	SecretKey string
}

// RequirePublicKey returns an error if the public key is missing.
// Use this for upload API commands that only need the public key.
func (c *ProjectCredentials) RequirePublicKey() error {
	if c.PublicKey == "" {
		return fmt.Errorf("%w: public_key is required", ErrNoProjectCredentials)
	}
	return nil
}

// RequireBoth returns an error if either key is missing.
// Use this for REST API commands that need both keys for HMAC auth.
func (c *ProjectCredentials) RequireBoth() error {
	if c.PublicKey == "" && c.SecretKey == "" {
		return fmt.Errorf("%w: set via --public-key/--secret-key, UPLOADCARE_PUBLIC_KEY/UPLOADCARE_SECRET_KEY, or configure a project in %s", ErrNoProjectCredentials, ConfigPath())
	}
	if c.PublicKey == "" {
		return fmt.Errorf("%w: --secret-key is set but --public-key is missing", ErrNoProjectCredentials)
	}
	if c.SecretKey == "" {
		return fmt.Errorf("%w: --public-key is set but --secret-key is missing", ErrNoProjectCredentials)
	}
	return nil
}

// Config holds the fully resolved CLI configuration.
type Config struct {
	// Credentials
	PublicKey       string
	SecretKey       string
	ProjectAPIToken string

	// API base URLs
	RESTAPIBase    string
	UploadAPIBase  string
	CDNBase        string
	ProjectAPIBase string

	// Output
	Verbose bool
}

// ProjectEntry represents a project in the config file's projects map.
type ProjectEntry struct {
	PublicKey string `mapstructure:"public_key"`
	SecretKey string `mapstructure:"secret_key"`
}

// Loader loads and resolves configuration from flags, env vars, and config file.
type Loader struct {
	v       *viper.Viper
	flagSet map[string]bool // tracks keys explicitly set via CLI flags
}

// NewLoader creates a new config loader backed by the given Viper instance.
// If v is nil, a new Viper instance is created.
func NewLoader(v *viper.Viper) *Loader {
	if v == nil {
		v = viper.New()
	}
	return &Loader{v: v, flagSet: make(map[string]bool)}
}

// Init sets up Viper to read from the config file and bind environment variables.
// Call this once after creating the Loader.
// Returns an error if the config file exists but cannot be parsed.
func (l *Loader) Init() error {
	l.v.SetConfigName("config")
	l.v.SetConfigType("yaml")
	l.v.AddConfigPath(ConfigDir())

	// Bind env vars
	l.v.SetEnvPrefix("")
	l.v.BindEnv("public_key", "UPLOADCARE_PUBLIC_KEY")
	l.v.BindEnv("secret_key", "UPLOADCARE_SECRET_KEY")
	l.v.BindEnv("project_api_token", "UPLOADCARE_PROJECT_API_TOKEN")
	l.v.BindEnv("project", "UPLOADCARE_PROJECT")
	l.v.BindEnv("verbose", "UPLOADCARE_VERBOSE")

	l.v.BindEnv("rest_api_base", "UPLOADCARE_REST_API_BASE")
	l.v.BindEnv("upload_api_base", "UPLOADCARE_UPLOAD_API_BASE")
	l.v.BindEnv("cdn_base", "UPLOADCARE_CDN_BASE")
	l.v.BindEnv("project_api_base", "UPLOADCARE_PROJECT_API_BASE")

	// Defaults
	l.v.SetDefault("rest_api_base", DefaultRESTAPIBase)
	l.v.SetDefault("upload_api_base", DefaultUploadAPIBase)
	l.v.SetDefault("cdn_base", DefaultCDNBase)
	l.v.SetDefault("project_api_base", DefaultProjectAPIBase)

	// Read config file — missing file is fine, anything else is a real error.
	if err := l.v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}
	return nil
}

// BindFlags binds cobra persistent flags to viper keys.
// Call this in PersistentPreRun after flags are parsed.
func (l *Loader) BindFlags(cmd *cobra.Command) {
	flags := cmd.Root().PersistentFlags()

	l.bindFlag(flags, "public-key", "public_key")
	l.bindFlag(flags, "secret-key", "secret_key")
	l.bindFlag(flags, "project-api-token", "project_api_token")
	l.bindFlag(flags, "project", "project")
	l.bindFlag(flags, "rest-api-base", "rest_api_base")
	l.bindFlag(flags, "upload-api-base", "upload_api_base")
	l.bindFlag(flags, "cdn-base", "cdn_base")
	l.bindFlag(flags, "project-api-base", "project_api_base")
	l.bindBoolFlag(flags, "verbose", "verbose")
}

// bindFlag binds a string flag only if it was explicitly set by the user,
// so that flag defaults don't override env vars or config file values.
func (l *Loader) bindFlag(flags interface {
	GetString(string) (string, error)
	Changed(string) bool
}, flag, key string) {
	if flags.Changed(flag) {
		val, _ := flags.GetString(flag)
		l.v.Set(key, val)
		l.flagSet[key] = true
	}
}

// bindBoolFlag binds a bool flag only if it was explicitly set by the user.
func (l *Loader) bindBoolFlag(flags interface {
	GetBool(string) (bool, error)
	Changed(string) bool
}, flag, key string) {
	if flags.Changed(flag) {
		val, _ := flags.GetBool(flag)
		l.v.Set(key, val)
		l.flagSet[key] = true
	}
}

// ResolveProjectCredentials resolves project credentials using the priority order:
//  1. --public-key / --secret-key flags (bound via BindFlags)
//  2. UPLOADCARE_PUBLIC_KEY / UPLOADCARE_SECRET_KEY env vars
//  3. --project / UPLOADCARE_PROJECT → lookup in projects map
//  4. default_project → lookup in projects map
//  5. top-level public_key / secret_key in config file
//
// Returns whatever credentials were found without validating completeness.
// If either direct key is set via flags or env vars, project lookup is
// skipped (the user is choosing direct credentials). Top-level config file
// keys are only used as a final fallback after project lookup.
// Commands should call RequirePublicKey or RequireBoth on the result to
// validate what they need.
func (l *Loader) ResolveProjectCredentials() (*ProjectCredentials, error) {
	// If either direct key was explicitly provided via flags or env vars
	// (priorities 1-2), return immediately. This prevents a misconfigured
	// shell/CI from silently targeting the wrong project.
	if l.directKeysFromFlagOrEnv() {
		pubKey := l.v.GetString("public_key")
		secKey := l.v.GetString("secret_key")
		return &ProjectCredentials{PublicKey: pubKey, SecretKey: secKey}, nil
	}

	// No direct keys from flags/env — try project name lookup (priorities 3-4).
	projectName := l.v.GetString("project")
	if projectName == "" {
		projectName = l.v.GetString("default_project")
	}

	if projectName != "" {
		return l.lookupProject(projectName)
	}

	// Fall back to top-level config file keys (priority 5).
	pubKey := l.v.GetString("public_key")
	secKey := l.v.GetString("secret_key")
	return &ProjectCredentials{PublicKey: pubKey, SecretKey: secKey}, nil
}

// directKeysFromFlagOrEnv reports whether public_key or secret_key was
// explicitly provided via CLI flags or environment variables.
func (l *Loader) directKeysFromFlagOrEnv() bool {
	return l.flagSet["public_key"] || l.flagSet["secret_key"] ||
		os.Getenv("UPLOADCARE_PUBLIC_KEY") != "" || os.Getenv("UPLOADCARE_SECRET_KEY") != ""
}

// lookupProject looks up a named project in the config file's projects map.
func (l *Loader) lookupProject(name string) (*ProjectCredentials, error) {
	projects := l.v.GetStringMap("projects")
	if projects == nil {
		return nil, fmt.Errorf("%w: %q (no projects configured)", ErrProjectNotFound, name)
	}

	// Viper lowercases map keys, so normalize the lookup name to match.
	projectRaw, ok := projects[strings.ToLower(name)]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrProjectNotFound, name)
	}

	projectMap, ok := projectRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("%w: %q (invalid format)", ErrProjectNotFound, name)
	}

	pubKey, _ := projectMap["public_key"].(string)
	secKey, _ := projectMap["secret_key"].(string)

	if pubKey == "" || secKey == "" {
		return nil, fmt.Errorf("%w: %q (missing public_key or secret_key)", ErrProjectNotFound, name)
	}

	return &ProjectCredentials{PublicKey: pubKey, SecretKey: secKey}, nil
}

// ResolveProjectAPIToken resolves the account-level bearer token using the priority order:
//  1. --project-api-token flag (bound via BindFlags)
//  2. UPLOADCARE_PROJECT_API_TOKEN env var
//  3. top-level project_api_token in config file
//
// Returns whatever was found (possibly empty). Commands should call
// RequireProjectAPIToken on the result to validate.
func (l *Loader) ResolveProjectAPIToken() string {
	return l.v.GetString("project_api_token")
}

// RequireProjectAPIToken returns an error if token is empty.
func RequireProjectAPIToken(token string) error {
	if token == "" {
		return fmt.Errorf("%w: set via --project-api-token, UPLOADCARE_PROJECT_API_TOKEN, or 'project_api_token' in %s", ErrNoProjectAPIToken, ConfigPath())
	}
	return nil
}

// Resolve returns a fully resolved Config with all fields populated.
// Credential fields may be empty if not configured — use ResolveProjectCredentials
// or ResolveProjectAPIToken for validation.
func (l *Loader) Resolve() *Config {
	return &Config{
		PublicKey:       l.v.GetString("public_key"),
		SecretKey:       l.v.GetString("secret_key"),
		ProjectAPIToken: l.v.GetString("project_api_token"),
		RESTAPIBase:     l.v.GetString("rest_api_base"),
		UploadAPIBase:   l.v.GetString("upload_api_base"),
		CDNBase:         l.v.GetString("cdn_base"),
		ProjectAPIBase:  l.v.GetString("project_api_base"),
		Verbose:         l.v.GetBool("verbose"),
	}
}

// Viper returns the underlying Viper instance for advanced usage.
func (l *Loader) Viper() *viper.Viper {
	return l.v
}
