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
		Long: `Convert documents and videos stored in Uploadcare.

Conversions are asynchronous. By default the CLI polls for completion
(every 2s, up to --timeout). Use --no-wait to return the conversion
token immediately and check status later.

Subcommands: document, video.`,
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
		Long: `Convert a document file to another format.

The --format flag is required and specifies the target format
(e.g. pdf, doc, docx, txt, odt, rtf, html, etc.).

Use --page to extract a specific page from multi-page documents.
Use --save-in-group to save multi-page results as a file group.
Use --store to permanently store the converted file.

By default waits for conversion to complete (polling every 2s, up to
--timeout). Use --no-wait to return the conversion token immediately.

Use --dry-run to validate parameters without converting.

JSON fields (with --no-wait): token, uuid.
JSON fields (after completion): status, result.`,
		Example: `  # Convert a document to PDF
  uploadcare convert document 740e1b8c-1ad8-4324-b7ec-112345678900 --format pdf

  # Convert and store the result
  uploadcare convert document 740e1b8c-1ad8-4324-b7ec-112345678900 --format pdf --store

  # Extract page 3 as an image
  uploadcare convert document 740e1b8c-1ad8-4324-b7ec-112345678900 --format png --page 3

  # Start conversion without waiting
  uploadcare convert document 740e1b8c-1ad8-4324-b7ec-112345678900 --format pdf --no-wait --json all

  # Dry run: validate parameters
  uploadcare convert document 740e1b8c-1ad8-4324-b7ec-112345678900 --format pdf --dry-run --json all`,
		Args: cobra.ExactArgs(1),
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
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would convert %s to %s\n", uuid, format)
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
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Token: %s\nUUID: %s\n", result.Token, result.UUID)
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\nResult: %s\n", status.Status, status.ResultURL)
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
		Long: `Convert a video file with optional format, size, quality, and cutting.

All flags are optional. Common target formats: mp4, webm, ogg.

Resize options:
  --size WxH           Target dimensions (e.g. 640x480)
  --resize-mode MODE   How to resize: preserve_ratio, change_ratio,
                       scale_crop, add_padding

Quality options: normal, better, best, lighter, lightest.

Cut option: --cut START/END in HHH:MM:SS.mmm format
  (e.g. 000:00:05.000/000:00:15.000 for a 10-second clip).

Use --thumbs N to generate N thumbnail images from the video.
Use --store to permanently store the converted file.

By default waits for conversion to complete (polling every 2s, up to
--timeout). Use --no-wait to return the conversion token immediately.

Use --dry-run to validate parameters without converting.

JSON fields (with --no-wait): token, uuid.
JSON fields (after completion): status, result.`,
		Example: `  # Convert video to mp4
  uploadcare convert video 740e1b8c-1ad8-4324-b7ec-112345678900 --format mp4

  # Convert with resize and quality
  uploadcare convert video 740e1b8c-1ad8-4324-b7ec-112345678900 \
    --format mp4 --size 1280x720 --quality better

  # Cut a 10-second clip
  uploadcare convert video 740e1b8c-1ad8-4324-b7ec-112345678900 \
    --format mp4 --cut 000:00:05.000/000:00:15.000

  # Generate 5 thumbnails
  uploadcare convert video 740e1b8c-1ad8-4324-b7ec-112345678900 --thumbs 5

  # Start conversion without waiting
  uploadcare convert video 740e1b8c-1ad8-4324-b7ec-112345678900 --format mp4 --no-wait --json all`,
		Args: cobra.ExactArgs(1),
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
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Would convert video %s\n", uuid)
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
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Token: %s\nUUID: %s\n", result.Token, result.UUID)
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

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\nResult: %s\n", status.Status, status.ResultURL)
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
