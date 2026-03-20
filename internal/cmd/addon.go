package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
)

// addonNameMap maps CLI-friendly hyphenated names to SDK underscore names.
var addonNameMap = map[string]string{
	"aws-rekognition-detect-labels":            "aws_rekognition_detect_labels",
	"aws-rekognition-detect-moderation-labels": "aws_rekognition_detect_moderation_labels",
	"remove-bg":                                "remove_bg",
	"uc-clamav-virus-scan":                     "uc_clamav_virus_scan",
}

func validAddonNames() []string {
	names := make([]string, 0, len(addonNameMap))
	for name := range addonNameMap {
		names = append(names, name)
	}
	return names
}

func resolveAddonName(name string) (string, error) {
	if sdkName, ok := addonNameMap[name]; ok {
		return sdkName, nil
	}
	// Also accept SDK-style names directly
	for _, sdkName := range addonNameMap {
		if sdkName == name {
			return sdkName, nil
		}
	}
	return "", fmt.Errorf("unknown addon %q; valid addons: %v", name, validAddonNames())
}

func newAddonCmd(addonSvc service.AddonService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addon",
		Short: "Execute add-ons",
	}

	cmd.AddCommand(newAddonExecuteCmd(addonSvc))
	cmd.AddCommand(newAddonStatusCmd(addonSvc))

	return cmd
}

func newAddonExecuteCmd(addonSvc service.AddonService) *cobra.Command {
	var (
		params  string
		noWait  bool
		timeout time.Duration
		dryRun  bool
	)

	cmd := &cobra.Command{
		Use:   "execute <addon-name> <file-uuid>",
		Short: "Execute an add-on on a file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addonName, fileUUID := args[0], args[1]

			sdkName, err := resolveAddonName(addonName)
			if err != nil {
				return &ExitError{Code: 2, Err: err}
			}
			if err := validate.UUID(fileUUID); err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			var rawParams json.RawMessage
			if params != "" {
				if !json.Valid([]byte(params)) {
					return ExitErrorf(2, "invalid JSON in --params")
				}
				rawParams = json.RawMessage(params)
			}

			svc := addonSvc
			if svc == nil {
				var err error
				svc, err = addonServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)
			verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]string{
						"addon":  addonName,
						"file":   fileUUID,
						"status": "would execute",
					})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Would execute %s on %s\n", addonName, fileUUID)
				return nil
			}

			result, err := svc.Execute(cmd.Context(), sdkName, fileUUID, rawParams)
			if err != nil {
				return err
			}

			if noWait {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), result)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Request ID: %s\n", result.RequestID)
				return nil
			}

			status, err := pollAddon(cmd.Context(), svc, sdkName, result.RequestID, timeout, verbose)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), status)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", status.Status)
			if len(status.Result) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Result: %s\n", string(status.Result))
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&params, "params", "", "Add-on specific parameters (JSON)")
	f.BoolVar(&noWait, "no-wait", false, "Don't wait for execution to finish")
	f.DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for waiting")
	f.BoolVar(&dryRun, "dry-run", false, "Validate without executing")

	return cmd
}

func newAddonStatusCmd(addonSvc service.AddonService) *cobra.Command {
	return &cobra.Command{
		Use:   "status <addon-name> <request-id>",
		Short: "Check add-on execution status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addonName, requestID := args[0], args[1]

			sdkName, err := resolveAddonName(addonName)
			if err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			svc := addonSvc
			if svc == nil {
				var err error
				svc, err = addonServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			status, err := svc.Status(cmd.Context(), sdkName, requestID)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), status)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", status.Status)
			if len(status.Result) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Result: %s\n", string(status.Result))
			}
			return nil
		},
	}
}

func pollAddon(
	ctx context.Context,
	svc service.AddonService,
	addonName, requestID string,
	timeout time.Duration,
	verbose *output.VerboseLogger,
) (*service.AddonStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		status, err := svc.Status(ctx, addonName, requestID)
		if err != nil {
			return nil, err
		}

		switch status.Status {
		case "done":
			return status, nil
		case "error":
			return nil, fmt.Errorf("addon execution failed")
		}

		verbose.Infof("addon status: %s", status.Status)

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return nil, fmt.Errorf("addon execution timed out after %s", timeout)
		}
	}
}
