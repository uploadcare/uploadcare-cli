package cmd

import (
	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/client"
	"github.com/uploadcare/uploadcare-cli/internal/config"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
)

// formatOptionsFromCmd reads output flags from the root command and returns FormatOptions.
func formatOptionsFromCmd(cmd *cobra.Command) output.FormatOptions {
	f := cmd.Root().PersistentFlags()
	jsonRaw, _ := f.GetString("json")
	jq, _ := f.GetString("jq")
	quiet, _ := f.GetBool("quiet")
	verbose, _ := f.GetBool("verbose")

	jsonEnabled, fields := output.ParseJSONFlag(jsonRaw)

	// --jq implies --json (as documented in root help text)
	if jq != "" {
		jsonEnabled = true
	}

	return output.FormatOptions{
		JSON:    jsonEnabled,
		Fields:  fields,
		JQ:      jq,
		Quiet:   quiet,
		Verbose: verbose,
	}
}

// fileServiceFromCmd resolves credentials from config and creates a FileService.
func fileServiceFromCmd(cmd *cobra.Command) (service.FileService, error) {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	creds, err := loader.ResolveProjectCredentials(verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}
	if err := creds.RequireBoth(); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	httpClient := client.NewVerboseHTTPClient(verbose)

	cdnBase := loader.ResolveCDNBase(creds, verbose)
	svc, err := client.NewFileService(creds.PublicKey, creds.SecretKey, cdnBase, httpClient, verbose)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// metadataServiceFromCmd resolves credentials and creates a MetadataService.
func metadataServiceFromCmd(cmd *cobra.Command) (service.MetadataService, error) {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	creds, err := loader.ResolveProjectCredentials(verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}
	if err := creds.RequireBoth(); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	httpClient := client.NewVerboseHTTPClient(verbose)
	svc, err := client.NewMetadataService(creds.PublicKey, creds.SecretKey, httpClient, verbose)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// groupServiceFromCmd resolves credentials and creates a GroupService.
func groupServiceFromCmd(cmd *cobra.Command) (service.GroupService, error) {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	creds, err := loader.ResolveProjectCredentials(verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}
	if err := creds.RequireBoth(); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	httpClient := client.NewVerboseHTTPClient(verbose)
	svc, err := client.NewGroupService(creds.PublicKey, creds.SecretKey, httpClient, verbose)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// webhookServiceFromCmd resolves credentials and creates a WebhookService.
func webhookServiceFromCmd(cmd *cobra.Command) (service.WebhookService, error) {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	creds, err := loader.ResolveProjectCredentials(verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}
	if err := creds.RequireBoth(); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	httpClient := client.NewVerboseHTTPClient(verbose)
	svc, err := client.NewWebhookService(creds.PublicKey, creds.SecretKey, httpClient, verbose)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// convertServiceFromCmd resolves credentials and creates a ConvertService.
func convertServiceFromCmd(cmd *cobra.Command) (service.ConvertService, error) {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	creds, err := loader.ResolveProjectCredentials(verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}
	if err := creds.RequireBoth(); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	httpClient := client.NewVerboseHTTPClient(verbose)
	svc, err := client.NewConvertService(creds.PublicKey, creds.SecretKey, httpClient, verbose)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// addonServiceFromCmd resolves credentials and creates an AddonService.
func addonServiceFromCmd(cmd *cobra.Command) (service.AddonService, error) {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	creds, err := loader.ResolveProjectCredentials(verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}
	if err := creds.RequireBoth(); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	httpClient := client.NewVerboseHTTPClient(verbose)
	svc, err := client.NewAddonService(creds.PublicKey, creds.SecretKey, httpClient, verbose)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// projectServiceFromCmd resolves credentials and creates a REST API ProjectService.
func projectServiceFromCmd(cmd *cobra.Command) (service.ProjectService, error) {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	creds, err := loader.ResolveProjectCredentials(verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}
	if err := creds.RequireBoth(); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	httpClient := client.NewVerboseHTTPClient(verbose)
	svc, err := client.NewProjectService(creds.PublicKey, creds.SecretKey, httpClient, verbose)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// projectMgmtServiceFromCmd resolves bearer token and creates a ProjectManagementService.
func projectMgmtServiceFromCmd(cmd *cobra.Command) (service.ProjectManagementService, error) {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	token := loader.ResolveProjectAPIToken()
	if err := config.RequireProjectAPIToken(token); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	httpClient := client.NewVerboseHTTPClient(verbose)
	svc, err := client.NewProjectManagementService(token, httpClient, verbose)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// secretServiceFromCmd resolves bearer token and creates a SecretService.
func secretServiceFromCmd(cmd *cobra.Command) (service.SecretService, error) {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	token := loader.ResolveProjectAPIToken()
	if err := config.RequireProjectAPIToken(token); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	httpClient := client.NewVerboseHTTPClient(verbose)
	svc, err := client.NewSecretService(token, httpClient, verbose)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// usageServiceFromCmd resolves bearer token and creates a UsageService.
func usageServiceFromCmd(cmd *cobra.Command) (service.UsageService, error) {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	token := loader.ResolveProjectAPIToken()
	if err := config.RequireProjectAPIToken(token); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	httpClient := client.NewVerboseHTTPClient(verbose)
	svc, err := client.NewUsageService(token, httpClient, verbose)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// resolveProjectPubKey resolves a project identifier (name or pub_key).
// If the arg matches a project name in config, returns its public_key.
// Otherwise treats the arg as a literal pub_key.
func resolveProjectPubKey(cmd *cobra.Command, arg string) string {
	opts := formatOptionsFromCmd(cmd)
	verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

	loader, err := configLoaderFromCmd(cmd, verbose)
	if err != nil {
		return arg
	}

	pubKey, err := loader.LookupProjectPubKey(arg)
	if err != nil {
		// Not found in config — treat as literal pub_key.
		return arg
	}
	verbose.Info("project", arg+" → "+pubKey+" (from config)")
	return pubKey
}

// configCmdKey is the context key for the root command reference.
type configCmdKey struct{}

// configLoaderFromCmd lazily initializes and returns the config loader.
// Config is only loaded when a command actually needs it, so config-free
// commands (e.g. version) are not affected by a malformed config file.
func configLoaderFromCmd(cmd *cobra.Command, verbose *output.VerboseLogger) (*config.Loader, error) {
	rootCmd, _ := cmd.Context().Value(configCmdKey{}).(*cobra.Command)
	if rootCmd == nil {
		rootCmd = cmd
	}

	loader := config.NewLoader(nil)
	loader.SetVerbose(verbose)
	if err := loader.Init(); err != nil {
		return nil, err
	}
	loader.BindFlags(rootCmd)
	return loader, nil
}
