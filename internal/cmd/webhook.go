package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
)

var validWebhookEvents = []string{
	"file.uploaded",
	"file.stored",
	"file.deleted",
	"file.info_updated",
}

func isValidWebhookEvent(event string) bool {
	for _, e := range validWebhookEvents {
		if e == event {
			return true
		}
	}
	return false
}

func newWebhookCmd(webhookSvc service.WebhookService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage webhooks",
	}

	cmd.AddCommand(newWebhookListCmd(webhookSvc))
	cmd.AddCommand(newWebhookCreateCmd(webhookSvc))
	cmd.AddCommand(newWebhookUpdateCmd(webhookSvc))
	cmd.AddCommand(newWebhookDeleteCmd(webhookSvc))

	return cmd
}

func newWebhookListCmd(webhookSvc service.WebhookService) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all webhooks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := webhookSvc
			if svc == nil {
				var err error
				svc, err = webhookServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			webhooks, err := svc.List(cmd.Context())
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), webhooks)
			}

			table := output.NewTableData("ID", "TARGET_URL", "EVENT", "ACTIVE", "CREATED")
			for _, w := range webhooks {
				table.AddRow(
					strconv.Itoa(w.ID),
					w.TargetURL,
					w.Event,
					strconv.FormatBool(w.IsActive),
					formatTime(w.DatetimeCreated),
				)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}
}

func newWebhookCreateCmd(webhookSvc service.WebhookService) *cobra.Command {
	var (
		event         string
		active        bool
		signingSecret string
		dryRun        bool
	)

	cmd := &cobra.Command{
		Use:   "create <target-url>",
		Short: "Create a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := args[0]
			if err := validate.URL(targetURL); err != nil {
				return &ExitError{Code: 2, Err: err}
			}
			if !isValidWebhookEvent(event) {
				return ExitErrorf(2, "invalid event %q; must be one of: %v", event, validWebhookEvents)
			}

			svc := webhookSvc
			if svc == nil {
				var err error
				svc, err = webhookServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]interface{}{
						"target_url": targetURL,
						"event":      event,
						"is_active":  active,
						"status":     "would create",
					})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Would create webhook: %s (event: %s, active: %v)\n", targetURL, event, active)
				return nil
			}

			params := service.WebhookCreateParams{
				TargetURL:     targetURL,
				Event:         event,
				IsActive:      active,
				SigningSecret: signingSecret,
			}

			w, err := svc.Create(cmd.Context(), params)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), w)
			}

			table := &output.TableData{}
			table.AddRow("ID:", strconv.Itoa(w.ID))
			table.AddRow("Target URL:", w.TargetURL)
			table.AddRow("Event:", w.Event)
			table.AddRow("Active:", strconv.FormatBool(w.IsActive))
			table.AddRow("Created:", formatTime(w.DatetimeCreated))
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	f := cmd.Flags()
	f.StringVar(&event, "event", "file.uploaded", "Event type")
	f.BoolVar(&active, "active", true, "Whether the webhook is active")
	f.StringVar(&signingSecret, "signing-secret", "", "Signing secret for webhook verification")
	f.BoolVar(&dryRun, "dry-run", false, "Validate without creating")

	return cmd
}

func newWebhookUpdateCmd(webhookSvc service.WebhookService) *cobra.Command {
	var (
		targetURL     string
		event         string
		active        string
		signingSecret string
		dryRun        bool
	)

	cmd := &cobra.Command{
		Use:   "update <webhook-id>",
		Short: "Update a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if _, err := strconv.ParseInt(id, 10, 64); err != nil {
				return ExitErrorf(2, "invalid webhook ID %q: must be an integer", id)
			}

			svc := webhookSvc
			if svc == nil {
				var err error
				svc, err = webhookServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			params := service.WebhookUpdateParams{}
			if cmd.Flags().Changed("target-url") {
				if err := validate.URL(targetURL); err != nil {
					return &ExitError{Code: 2, Err: err}
				}
				params.TargetURL = &targetURL
			}
			if cmd.Flags().Changed("event") {
				if !isValidWebhookEvent(event) {
					return ExitErrorf(2, "invalid event %q; must be one of: %v", event, validWebhookEvents)
				}
				params.Event = &event
			}
			if cmd.Flags().Changed("active") {
				b, err := strconv.ParseBool(active)
				if err != nil {
					return ExitErrorf(2, "invalid --active value: %q", active)
				}
				params.IsActive = &b
			}
			if cmd.Flags().Changed("signing-secret") {
				params.SigningSecret = &signingSecret
			}

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]interface{}{
						"id":     id,
						"status": "would update",
					})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Would update webhook %s\n", id)
				return nil
			}

			w, err := svc.Update(cmd.Context(), id, params)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), w)
			}

			table := &output.TableData{}
			table.AddRow("ID:", strconv.Itoa(w.ID))
			table.AddRow("Target URL:", w.TargetURL)
			table.AddRow("Event:", w.Event)
			table.AddRow("Active:", strconv.FormatBool(w.IsActive))
			table.AddRow("Updated:", formatTime(w.DatetimeUpdated))
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	f := cmd.Flags()
	f.StringVar(&targetURL, "target-url", "", "New target URL")
	f.StringVar(&event, "event", "", "New event type")
	f.StringVar(&active, "active", "", "Whether the webhook is active (true/false)")
	f.StringVar(&signingSecret, "signing-secret", "", "New signing secret")
	f.BoolVar(&dryRun, "dry-run", false, "Show what would change without applying")

	return cmd
}

func newWebhookDeleteCmd(webhookSvc service.WebhookService) *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "delete <webhook-id>",
		Short: "Delete a webhook",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if _, err := strconv.ParseInt(id, 10, 64); err != nil {
				return ExitErrorf(2, "invalid webhook ID %q: must be an integer", id)
			}

			svc := webhookSvc
			if svc == nil {
				var err error
				svc, err = webhookServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]string{
						"id":     id,
						"status": "would delete",
					})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Would delete webhook %s\n", id)
				return nil
			}

			if err := svc.Delete(cmd.Context(), id); err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]string{
					"id":     id,
					"status": "deleted",
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Deleted webhook %s\n", id)
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Verify without deleting")

	return cmd
}
