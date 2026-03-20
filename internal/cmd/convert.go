package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
)

func newConvertCmd(convertSvc service.ConvertService) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert documents and videos",
	}

	cmd.AddCommand(newConvertDocumentCmd(convertSvc))
	cmd.AddCommand(newConvertVideoCmd(convertSvc))

	return cmd
}

func newConvertDocumentCmd(convertSvc service.ConvertService) *cobra.Command {
	var (
		format      string
		page        int
		saveInGroup bool
		store       bool
		noWait      bool
		timeout     time.Duration
		dryRun      bool
	)

	cmd := &cobra.Command{
		Use:   "document <uuid>",
		Short: "Convert a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			if err := validate.UUID(uuid); err != nil {
				return &ExitError{Code: 2, Err: err}
			}
			if format == "" {
				return ExitErrorf(2, "--format is required")
			}

			svc := convertSvc
			if svc == nil {
				var err error
				svc, err = convertServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)
			verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

			params := service.DocConvertParams{
				UUID:        uuid,
				Format:      format,
				SaveInGroup: saveInGroup,
				Store:       store,
			}
			if cmd.Flags().Changed("page") {
				params.Page = &page
			}

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]interface{}{
						"uuid":   uuid,
						"format": format,
						"status": "would convert",
					})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Would convert %s to %s\n", uuid, format)
				return nil
			}

			result, err := svc.Document(cmd.Context(), params)
			if err != nil {
				return err
			}

			if noWait {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), result)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Token: %s\nUUID: %s\n", result.Token, result.UUID)
				return nil
			}

			status, err := pollConversion(cmd.Context(), svc.DocumentStatus, result.Token, timeout, verbose)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]string{
					"status": status.Status,
					"result": status.ResultURL,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\nResult: %s\n", status.Status, status.ResultURL)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&format, "format", "", "Target format (e.g. pdf, doc, txt)")
	f.IntVar(&page, "page", 0, "Page number for multi-page documents")
	f.BoolVar(&saveInGroup, "save-in-group", false, "Save multi-page result as file group")
	f.BoolVar(&store, "store", false, "Store the converted file")
	f.BoolVar(&noWait, "no-wait", false, "Don't wait for conversion to finish")
	f.DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for waiting")
	f.BoolVar(&dryRun, "dry-run", false, "Validate without converting")

	return cmd
}

func newConvertVideoCmd(convertSvc service.ConvertService) *cobra.Command {
	var (
		format     string
		size       string
		resizeMode string
		quality    string
		cut        string
		thumbs     int
		store      bool
		noWait     bool
		timeout    time.Duration
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "video <uuid>",
		Short: "Convert a video",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			if err := validate.UUID(uuid); err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			svc := convertSvc
			if svc == nil {
				var err error
				svc, err = convertServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)
			verbose := output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr())

			params := service.VideoConvertParams{
				UUID:       uuid,
				Format:     format,
				Size:       size,
				ResizeMode: resizeMode,
				Quality:    quality,
				Cut:        cut,
				Store:      store,
			}
			if cmd.Flags().Changed("thumbs") {
				params.Thumbs = &thumbs
			}

			if dryRun {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), map[string]interface{}{
						"uuid":   uuid,
						"format": format,
						"status": "would convert",
					})
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Would convert video %s\n", uuid)
				return nil
			}

			result, err := svc.Video(cmd.Context(), params)
			if err != nil {
				return err
			}

			if noWait {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), result)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Token: %s\nUUID: %s\n", result.Token, result.UUID)
				return nil
			}

			status, err := pollConversion(cmd.Context(), svc.VideoStatus, result.Token, timeout, verbose)
			if err != nil {
				return err
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), map[string]string{
					"status": status.Status,
					"result": status.ResultURL,
				})
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\nResult: %s\n", status.Status, status.ResultURL)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVar(&format, "format", "", "Target format (e.g. mp4, webm, ogg)")
	f.StringVar(&size, "size", "", "Output size (e.g. 640x480)")
	f.StringVar(&resizeMode, "resize-mode", "", "Resize mode (preserve_ratio, change_ratio, scale_crop, add_padding)")
	f.StringVar(&quality, "quality", "", "Output quality (normal, better, best, lighter, lightest)")
	f.StringVar(&cut, "cut", "", "Cut video (e.g. 000:00:05.000/000:00:15.000)")
	f.IntVar(&thumbs, "thumbs", 0, "Number of thumbnails to generate")
	f.BoolVar(&store, "store", false, "Store the converted file")
	f.BoolVar(&noWait, "no-wait", false, "Don't wait for conversion to finish")
	f.DurationVar(&timeout, "timeout", 5*time.Minute, "Timeout for waiting")
	f.BoolVar(&dryRun, "dry-run", false, "Validate without converting")

	return cmd
}

func pollConversion(
	ctx context.Context,
	statusFn func(ctx context.Context, token string) (*service.ConvertStatus, error),
	token string,
	timeout time.Duration,
	verbose *output.VerboseLogger,
) (*service.ConvertStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		status, err := statusFn(ctx, token)
		if err != nil {
			return nil, err
		}

		switch status.Status {
		case "finished":
			return status, nil
		case "failed", "canceled":
			return nil, fmt.Errorf("conversion %s: %s", status.Status, status.Error)
		}

		verbose.Infof("conversion status: %s", status.Status)

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return nil, fmt.Errorf("conversion timed out after %s", timeout)
		}
	}
}
