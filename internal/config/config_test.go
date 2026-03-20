package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// newTestLoader creates a Loader backed by a fresh Viper instance
// reading from the given config YAML content in a temp directory.
func newTestLoader(t *testing.T, yaml string) *Loader {
	t.Helper()

	dir := t.TempDir()
	if yaml != "" {
		err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0o644)
		if err != nil {
			t.Fatal(err)
		}
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)
	_ = v.ReadInConfig()

	return NewLoader(v)
}

func TestResolveProjectCredentials_DirectKeys(t *testing.T) {
	l := newTestLoader(t, `
public_key: "pub123"
secret_key: "sec456"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.PublicKey != "pub123" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "pub123")
	}
	if creds.SecretKey != "sec456" {
		t.Errorf("SecretKey = %q, want %q", creds.SecretKey, "sec456")
	}
}

func TestResolveProjectCredentials_NamedProject(t *testing.T) {
	l := newTestLoader(t, `
default_project: "my app"
projects:
  "my app":
    public_key: "proj-pub"
    secret_key: "proj-sec"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.PublicKey != "proj-pub" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "proj-pub")
	}
	if creds.SecretKey != "proj-sec" {
		t.Errorf("SecretKey = %q, want %q", creds.SecretKey, "proj-sec")
	}
}

func TestResolveProjectCredentials_MixedCaseProjectName(t *testing.T) {
	// Viper lowercases map keys, so "My App" becomes "my app" in the map.
	// The lookup must normalize the name to match.
	l := newTestLoader(t, `
default_project: "My App"
projects:
  "My App":
    public_key: "proj-pub"
    secret_key: "proj-sec"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.PublicKey != "proj-pub" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "proj-pub")
	}
	if creds.SecretKey != "proj-sec" {
		t.Errorf("SecretKey = %q, want %q", creds.SecretKey, "proj-sec")
	}
}

func TestResolveProjectCredentials_ExplicitProject(t *testing.T) {
	l := newTestLoader(t, `
default_project: "default"
projects:
  "default":
    public_key: "default-pub"
    secret_key: "default-sec"
  "staging":
    public_key: "staging-pub"
    secret_key: "staging-sec"
`)
	// Simulate --project flag by setting viper key directly
	l.v.Set("project", "staging")

	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.PublicKey != "staging-pub" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "staging-pub")
	}
}

func TestResolveProjectCredentials_FlagsOverrideAll(t *testing.T) {
	l := newTestLoader(t, `
public_key: "config-pub"
secret_key: "config-sec"
`)
	// Simulate flags (must mark flagSet so ResolveProjectCredentials knows
	// these came from flags, not the config file).
	l.v.Set("public_key", "flag-pub")
	l.v.Set("secret_key", "flag-sec")
	l.flagSet["public_key"] = true
	l.flagSet["secret_key"] = true

	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.PublicKey != "flag-pub" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "flag-pub")
	}
	if creds.SecretKey != "flag-sec" {
		t.Errorf("SecretKey = %q, want %q", creds.SecretKey, "flag-sec")
	}
}

func TestResolveProjectCredentials_EnvVars(t *testing.T) {
	l := newTestLoader(t, "")

	// Bind env vars manually (normally done in Init)
	l.v.BindEnv("public_key", "UPLOADCARE_PUBLIC_KEY")
	l.v.BindEnv("secret_key", "UPLOADCARE_SECRET_KEY")

	t.Setenv("UPLOADCARE_PUBLIC_KEY", "env-pub")
	t.Setenv("UPLOADCARE_SECRET_KEY", "env-sec")

	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.PublicKey != "env-pub" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "env-pub")
	}
	if creds.SecretKey != "env-sec" {
		t.Errorf("SecretKey = %q, want %q", creds.SecretKey, "env-sec")
	}
}

func TestResolveProjectCredentials_NoCredentials(t *testing.T) {
	l := newTestLoader(t, "")

	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("resolve should not error, got: %v", err)
	}
	if err := creds.RequireBoth(); !errors.Is(err, ErrNoProjectCredentials) {
		t.Errorf("RequireBoth() = %v, want ErrNoProjectCredentials", err)
	}
	if err := creds.RequirePublicKey(); !errors.Is(err, ErrNoProjectCredentials) {
		t.Errorf("RequirePublicKey() = %v, want ErrNoProjectCredentials", err)
	}
}

func TestResolveProjectCredentials_OnlyPublicKey(t *testing.T) {
	l := newTestLoader(t, `
public_key: "pub-only"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("resolve should not error, got: %v", err)
	}
	if creds.PublicKey != "pub-only" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "pub-only")
	}
	// Upload commands only need public key — should pass.
	if err := creds.RequirePublicKey(); err != nil {
		t.Errorf("RequirePublicKey() = %v, want nil", err)
	}
	// REST commands need both — should fail.
	if err := creds.RequireBoth(); !errors.Is(err, ErrNoProjectCredentials) {
		t.Errorf("RequireBoth() = %v, want ErrNoProjectCredentials", err)
	}
}

func TestResolveProjectCredentials_OnlySecretKey(t *testing.T) {
	l := newTestLoader(t, `
secret_key: "sec-only"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("resolve should not error, got: %v", err)
	}
	if err := creds.RequirePublicKey(); !errors.Is(err, ErrNoProjectCredentials) {
		t.Errorf("RequirePublicKey() = %v, want ErrNoProjectCredentials", err)
	}
	if err := creds.RequireBoth(); !errors.Is(err, ErrNoProjectCredentials) {
		t.Errorf("RequireBoth() = %v, want ErrNoProjectCredentials", err)
	}
}

func TestResolveProjectCredentials_PartialFlagKeyBlocksProjectFallback(t *testing.T) {
	l := newTestLoader(t, `
default_project: "prod"
projects:
  "prod":
    public_key: "prod-pub"
    secret_key: "prod-sec"
`)
	// Simulate --public-key flag without --secret-key.
	l.v.Set("public_key", "flag-pub")
	l.flagSet["public_key"] = true

	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("resolve should not error, got: %v", err)
	}
	// Flag key was set, so project lookup must be skipped.
	if creds.PublicKey != "flag-pub" {
		t.Errorf("PublicKey = %q, want %q (direct, not from project)", creds.PublicKey, "flag-pub")
	}
	if creds.SecretKey != "" {
		t.Errorf("SecretKey = %q, want empty (should not fall through to project)", creds.SecretKey)
	}
}

func TestResolveProjectCredentials_ConfigPartialKeyYieldsToProject(t *testing.T) {
	// Config-file-only partial key should NOT block project lookup,
	// because top-level config keys (priority 5) are lower than
	// project lookup (priority 3-4).
	l := newTestLoader(t, `
public_key: "partial-pub"
default_project: "prod"
projects:
  "prod":
    public_key: "prod-pub"
    secret_key: "prod-sec"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.PublicKey != "prod-pub" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "prod-pub")
	}
	if creds.SecretKey != "prod-sec" {
		t.Errorf("SecretKey = %q, want %q", creds.SecretKey, "prod-sec")
	}
}

func TestResolveProjectCredentials_PartialEnvKeyBlocksProjectFallback(t *testing.T) {
	l := newTestLoader(t, `
default_project: "prod"
projects:
  "prod":
    public_key: "prod-pub"
    secret_key: "prod-sec"
`)
	l.v.BindEnv("secret_key", "UPLOADCARE_SECRET_KEY")
	t.Setenv("UPLOADCARE_SECRET_KEY", "env-sec")

	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("resolve should not error, got: %v", err)
	}
	// Direct key was set, so project lookup must be skipped.
	if creds.SecretKey != "env-sec" {
		t.Errorf("SecretKey = %q, want %q", creds.SecretKey, "env-sec")
	}
	if creds.PublicKey != "" {
		t.Errorf("PublicKey = %q, want empty (should not fall through to project)", creds.PublicKey)
	}
}

func TestResolveProjectCredentials_ExplicitProjectOverridesConfigKeys(t *testing.T) {
	// --project flag must win over top-level config file keys.
	l := newTestLoader(t, `
public_key: "legacy-pub"
secret_key: "legacy-sec"
projects:
  "staging":
    public_key: "staging-pub"
    secret_key: "staging-sec"
`)
	l.v.Set("project", "staging")

	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.PublicKey != "staging-pub" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "staging-pub")
	}
	if creds.SecretKey != "staging-sec" {
		t.Errorf("SecretKey = %q, want %q", creds.SecretKey, "staging-sec")
	}
}

func TestResolveProjectCredentials_ProjectNotFound(t *testing.T) {
	l := newTestLoader(t, `
default_project: "nonexistent"
projects:
  "production":
    public_key: "pub"
    secret_key: "sec"
`)
	_, err := l.ResolveProjectCredentials(nil)
	if !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("err = %v, want ErrProjectNotFound", err)
	}
}

func TestResolveProjectCredentials_ProjectMissingKeys(t *testing.T) {
	l := newTestLoader(t, `
default_project: "broken"
projects:
  "broken":
    public_key: "pub-only"
`)
	_, err := l.ResolveProjectCredentials(nil)
	if !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("err = %v, want ErrProjectNotFound", err)
	}
}

func TestResolveProjectAPIToken_ConfigFile(t *testing.T) {
	l := newTestLoader(t, `
project_api_token: "my-bearer-token"
`)
	token := l.ResolveProjectAPIToken()
	if token != "my-bearer-token" {
		t.Errorf("token = %q, want %q", token, "my-bearer-token")
	}
}

func TestResolveProjectAPIToken_EnvVar(t *testing.T) {
	l := newTestLoader(t, "")
	l.v.BindEnv("project_api_token", "UPLOADCARE_PROJECT_API_TOKEN")
	t.Setenv("UPLOADCARE_PROJECT_API_TOKEN", "env-token")

	token := l.ResolveProjectAPIToken()
	if token != "env-token" {
		t.Errorf("token = %q, want %q", token, "env-token")
	}
}

func TestResolveProjectAPIToken_Missing(t *testing.T) {
	l := newTestLoader(t, "")

	token := l.ResolveProjectAPIToken()
	if token != "" {
		t.Errorf("token = %q, want empty", token)
	}
}

func TestRequireProjectAPIToken(t *testing.T) {
	if err := RequireProjectAPIToken("some-token"); err != nil {
		t.Errorf("RequireProjectAPIToken(non-empty) = %v, want nil", err)
	}
	if err := RequireProjectAPIToken(""); !errors.Is(err, ErrNoProjectAPIToken) {
		t.Errorf("RequireProjectAPIToken(empty) = %v, want ErrNoProjectAPIToken", err)
	}
}

func TestResolve_Defaults(t *testing.T) {
	l := newTestLoader(t, "")

	l.v.SetDefault("rest_api_base", DefaultRESTAPIBase)
	l.v.SetDefault("upload_api_base", DefaultUploadAPIBase)
	l.v.SetDefault("project_api_base", DefaultProjectAPIBase)

	cfg := l.Resolve()

	if cfg.RESTAPIBase != DefaultRESTAPIBase {
		t.Errorf("RESTAPIBase = %q, want %q", cfg.RESTAPIBase, DefaultRESTAPIBase)
	}
	if cfg.UploadAPIBase != DefaultUploadAPIBase {
		t.Errorf("UploadAPIBase = %q, want %q", cfg.UploadAPIBase, DefaultUploadAPIBase)
	}
	if cfg.CDNBase != "" {
		t.Errorf("CDNBase = %q, want empty (no viper default)", cfg.CDNBase)
	}
	if cfg.ProjectAPIBase != DefaultProjectAPIBase {
		t.Errorf("ProjectAPIBase = %q, want %q", cfg.ProjectAPIBase, DefaultProjectAPIBase)
	}
}

func TestResolveCDNBase_ExplicitValue(t *testing.T) {
	l := newTestLoader(t, `
cdn_base: "https://custom-cdn.example.com"
`)
	got := l.ResolveCDNBase(&ProjectCredentials{PublicKey: "somepublickey"}, nil)
	if got != "https://custom-cdn.example.com" {
		t.Errorf("ResolveCDNBase = %q, want explicit value", got)
	}
}

func TestResolveCDNBase_ComputedFromPublicKey(t *testing.T) {
	l := newTestLoader(t, "")
	got := l.ResolveCDNBase(&ProjectCredentials{PublicKey: "demopublickey"}, nil)
	if got == "" || got == DefaultCDNBase {
		t.Errorf("ResolveCDNBase = %q, want computed URL from public key", got)
	}
	if !strings.HasSuffix(got, ".ucarecd.net") {
		t.Errorf("ResolveCDNBase = %q, want *.ucarecd.net domain", got)
	}
}

func TestResolveCDNBase_FallbackNoPublicKey(t *testing.T) {
	l := newTestLoader(t, "")
	got := l.ResolveCDNBase(&ProjectCredentials{}, nil)
	if got != DefaultCDNBase {
		t.Errorf("ResolveCDNBase = %q, want %q", got, DefaultCDNBase)
	}
}

func TestResolveCDNBase_FallbackNilCreds(t *testing.T) {
	l := newTestLoader(t, "")
	got := l.ResolveCDNBase(nil, nil)
	if got != DefaultCDNBase {
		t.Errorf("ResolveCDNBase = %q, want %q", got, DefaultCDNBase)
	}
}

func TestResolveCDNBase_ExplicitOverridesComputation(t *testing.T) {
	l := newTestLoader(t, `
cdn_base: "https://override.example.com"
`)
	got := l.ResolveCDNBase(&ProjectCredentials{PublicKey: "demopublickey"}, nil)
	if got != "https://override.example.com" {
		t.Errorf("ResolveCDNBase = %q, want explicit override", got)
	}
}

func TestResolveCDNBase_EnvVar(t *testing.T) {
	l := newTestLoader(t, "")
	l.v.BindEnv("cdn_base", "UPLOADCARE_CDN_BASE")
	t.Setenv("UPLOADCARE_CDN_BASE", "https://env-cdn.example.com")

	got := l.ResolveCDNBase(&ProjectCredentials{PublicKey: "demopublickey"}, nil)
	if got != "https://env-cdn.example.com" {
		t.Errorf("ResolveCDNBase = %q, want env value", got)
	}
}

func TestResolveCDNBase_PerProjectOverridesAutoComputed(t *testing.T) {
	l := newTestLoader(t, `
default_project: "my app"
projects:
  "my app":
    public_key: "proj-pub"
    secret_key: "proj-sec"
    cdn_base: "https://my-project-cdn.example.com"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := l.ResolveCDNBase(creds, nil)
	if got != "https://my-project-cdn.example.com" {
		t.Errorf("ResolveCDNBase = %q, want per-project cdn_base", got)
	}
}

func TestResolveCDNBase_GlobalFallbackWhenProjectHasNoCDNBase(t *testing.T) {
	l := newTestLoader(t, `
cdn_base: "https://global-cdn.example.com"
default_project: "my app"
projects:
  "my app":
    public_key: "proj-pub"
    secret_key: "proj-sec"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := l.ResolveCDNBase(creds, nil)
	if got != "https://global-cdn.example.com" {
		t.Errorf("ResolveCDNBase = %q, want global cdn_base", got)
	}
}

func TestResolveCDNBase_PerProjectOverridesGlobal(t *testing.T) {
	l := newTestLoader(t, `
cdn_base: "https://global-cdn.example.com"
default_project: "my app"
projects:
  "my app":
    public_key: "proj-pub"
    secret_key: "proj-sec"
    cdn_base: "https://my-project-cdn.example.com"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := l.ResolveCDNBase(creds, nil)
	if got != "https://my-project-cdn.example.com" {
		t.Errorf("ResolveCDNBase = %q, want per-project cdn_base over global", got)
	}
}

func TestResolveCDNBase_FlagOverridesPerProject(t *testing.T) {
	l := newTestLoader(t, `
default_project: "my app"
projects:
  "my app":
    public_key: "proj-pub"
    secret_key: "proj-sec"
    cdn_base: "https://my-project-cdn.example.com"
`)
	// Simulate --cdn-base flag
	l.v.Set("cdn_base", "https://flag-cdn.example.com")
	l.flagSet["cdn_base"] = true

	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := l.ResolveCDNBase(creds, nil)
	if got != "https://flag-cdn.example.com" {
		t.Errorf("ResolveCDNBase = %q, want flag override", got)
	}
}

func TestResolveCDNBase_EnvOverridesPerProject(t *testing.T) {
	l := newTestLoader(t, `
default_project: "my app"
projects:
  "my app":
    public_key: "proj-pub"
    secret_key: "proj-sec"
    cdn_base: "https://my-project-cdn.example.com"
`)
	l.v.BindEnv("cdn_base", "UPLOADCARE_CDN_BASE")
	t.Setenv("UPLOADCARE_CDN_BASE", "https://env-cdn.example.com")

	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := l.ResolveCDNBase(creds, nil)
	if got != "https://env-cdn.example.com" {
		t.Errorf("ResolveCDNBase = %q, want env override", got)
	}
}

func TestResolveCDNBase_AutoComputedWhenNoProjectCDNBase(t *testing.T) {
	l := newTestLoader(t, `
default_project: "my app"
projects:
  "my app":
    public_key: "demopublickey"
    secret_key: "proj-sec"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := l.ResolveCDNBase(creds, nil)
	if got == "" || got == DefaultCDNBase {
		t.Errorf("ResolveCDNBase = %q, want auto-computed URL", got)
	}
	if !strings.HasSuffix(got, ".ucarecd.net") {
		t.Errorf("ResolveCDNBase = %q, want *.ucarecd.net domain", got)
	}
}

func TestResolve_OverrideBaseURLs(t *testing.T) {
	l := newTestLoader(t, `
rest_api_base: "https://custom-rest.example.com"
upload_api_base: "https://custom-upload.example.com"
cdn_base: "https://custom-cdn.example.com"
project_api_base: "https://custom-project.example.com"
`)
	cfg := l.Resolve()

	if cfg.RESTAPIBase != "https://custom-rest.example.com" {
		t.Errorf("RESTAPIBase = %q, want custom", cfg.RESTAPIBase)
	}
	if cfg.UploadAPIBase != "https://custom-upload.example.com" {
		t.Errorf("UploadAPIBase = %q, want custom", cfg.UploadAPIBase)
	}
	if cfg.CDNBase != "https://custom-cdn.example.com" {
		t.Errorf("CDNBase = %q, want custom", cfg.CDNBase)
	}
	if cfg.ProjectAPIBase != "https://custom-project.example.com" {
		t.Errorf("ProjectAPIBase = %q, want custom", cfg.ProjectAPIBase)
	}
}

func TestResolveProjectCredentials_PriorityOrder(t *testing.T) {
	// Config file has top-level keys, default_project, and a named project.
	// default_project (priority 4) should win over top-level config keys (priority 5).
	l := newTestLoader(t, `
public_key: "top-pub"
secret_key: "top-sec"
default_project: "named"
projects:
  "named":
    public_key: "named-pub"
    secret_key: "named-sec"
`)
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if creds.PublicKey != "named-pub" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "named-pub")
	}
	if creds.SecretKey != "named-sec" {
		t.Errorf("SecretKey = %q, want %q", creds.SecretKey, "named-sec")
	}
}

func TestBindFlags_SetsViperFromChangedFlags(t *testing.T) {
	l := newTestLoader(t, "")

	root := &cobra.Command{Use: "test"}
	flags := root.PersistentFlags()
	flags.String("public-key", "", "")
	flags.String("secret-key", "", "")
	flags.String("project-api-token", "", "")
	flags.String("project", "", "")
	flags.String("rest-api-base", "", "")
	flags.String("upload-api-base", "", "")
	flags.String("cdn-base", "", "")
	flags.String("project-api-base", "", "")

	// Simulate flag parsing
	root.SetArgs([]string{"--public-key", "flag-pub", "--secret-key", "flag-sec"})
	_ = root.Execute()

	l.BindFlags(root)

	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.PublicKey != "flag-pub" {
		t.Errorf("PublicKey = %q, want %q", creds.PublicKey, "flag-pub")
	}
	if creds.SecretKey != "flag-sec" {
		t.Errorf("SecretKey = %q, want %q", creds.SecretKey, "flag-sec")
	}
}

func TestBindFlags_UnchangedFlagsDontOverride(t *testing.T) {
	l := newTestLoader(t, `
public_key: "config-pub"
secret_key: "config-sec"
`)

	root := &cobra.Command{Use: "test"}
	flags := root.PersistentFlags()
	flags.String("public-key", "", "")
	flags.String("secret-key", "", "")
	flags.String("project-api-token", "", "")
	flags.String("project", "", "")
	flags.String("rest-api-base", "", "")
	flags.String("upload-api-base", "", "")
	flags.String("cdn-base", "", "")
	flags.String("project-api-base", "", "")

	// No flags set
	root.SetArgs([]string{})
	_ = root.Execute()

	l.BindFlags(root)

	// Config file values should be preserved
	creds, err := l.ResolveProjectCredentials(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.PublicKey != "config-pub" {
		t.Errorf("PublicKey = %q, want %q (from config)", creds.PublicKey, "config-pub")
	}
}

func TestInit(t *testing.T) {
	// We can't easily test Init() because it uses os.UserHomeDir().
	// Instead, test that NewLoader(nil) creates a valid viper.
	l := NewLoader(nil)
	if l.v == nil {
		t.Fatal("NewLoader(nil) should create a viper instance")
	}
	if l.Viper() == nil {
		t.Fatal("Viper() should return the instance")
	}
}

func TestInit_MissingConfigFile(t *testing.T) {
	dir := t.TempDir()

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir) // empty dir, no config file

	l := NewLoader(v)
	// Init reads from ConfigDir() which won't match our temp dir,
	// but we can test directly with a loader that has the right path.
	// Use a fresh viper pointed at the empty dir.
	if err := l.Init(); err != nil {
		t.Fatalf("Init() should not error when config file is missing, got: %v", err)
	}
}

func TestInit_MalformedConfigFile(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(`
public_key: [invalid yaml
  this is broken
`), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)

	l := NewLoader(v)
	if err := l.Init(); err == nil {
		t.Fatal("Init() should return an error for malformed config")
	}
}

func TestResolveProjectCredentials_NoProjectsMap(t *testing.T) {
	l := newTestLoader(t, `
default_project: "missing"
`)
	_, err := l.ResolveProjectCredentials(nil)
	if !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("err = %v, want ErrProjectNotFound", err)
	}
}

func TestResolve_VerboseDefault(t *testing.T) {
	l := newTestLoader(t, "")
	cfg := l.Resolve()
	if cfg.Verbose {
		t.Error("Verbose should be false by default")
	}
}

func TestResolve_VerboseFromConfig(t *testing.T) {
	l := newTestLoader(t, `
verbose: true
`)
	cfg := l.Resolve()
	if !cfg.Verbose {
		t.Error("Verbose should be true when set in config")
	}
}

func TestResolve_VerboseFromEnv(t *testing.T) {
	l := newTestLoader(t, "")
	l.v.BindEnv("verbose", "UPLOADCARE_VERBOSE")
	t.Setenv("UPLOADCARE_VERBOSE", "1")

	cfg := l.Resolve()
	if !cfg.Verbose {
		t.Error("Verbose should be true when UPLOADCARE_VERBOSE=1")
	}
}

func TestResolve_IncludesCredentials(t *testing.T) {
	l := newTestLoader(t, `
public_key: "pub"
secret_key: "sec"
project_api_token: "tok"
`)
	cfg := l.Resolve()
	if cfg.PublicKey != "pub" {
		t.Errorf("PublicKey = %q, want %q", cfg.PublicKey, "pub")
	}
	if cfg.SecretKey != "sec" {
		t.Errorf("SecretKey = %q, want %q", cfg.SecretKey, "sec")
	}
	if cfg.ProjectAPIToken != "tok" {
		t.Errorf("ProjectAPIToken = %q, want %q", cfg.ProjectAPIToken, "tok")
	}
}

