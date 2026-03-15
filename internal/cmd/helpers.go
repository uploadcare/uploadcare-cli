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
	loader, err := configLoaderFromCmd(cmd)
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	creds, err := loader.ResolveProjectCredentials()
	if err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}
	if err := creds.RequireBoth(); err != nil {
		return nil, &ExitError{Code: 3, Err: err}
	}

	svc, err := client.NewFileService(creds.PublicKey, creds.SecretKey)
	if err != nil {
		return nil, &ExitError{Code: 1, Err: err}
	}
	return svc, nil
}

// configCmdKey is the context key for the root command reference.
type configCmdKey struct{}

// configLoaderFromCmd lazily initializes and returns the config loader.
// Config is only loaded when a command actually needs it, so config-free
// commands (e.g. version) are not affected by a malformed config file.
func configLoaderFromCmd(cmd *cobra.Command) (*config.Loader, error) {
	rootCmd, _ := cmd.Context().Value(configCmdKey{}).(*cobra.Command)
	if rootCmd == nil {
		rootCmd = cmd
	}

	loader := config.NewLoader(nil)
	if err := loader.Init(); err != nil {
		return nil, err
	}
	loader.BindFlags(rootCmd)
	return loader, nil
}
