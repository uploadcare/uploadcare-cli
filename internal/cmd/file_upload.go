package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
)

type uploadFileEntry struct {
	path        string
	size        int64
	contentType string
}

func newFileUploadCmd(fileSvc service.FileService) *cobra.Command {
	var (
		store              string
		metadata           []string
		multipartThreshold int64
		forceMultipart     bool
		forceDirect        bool
		showProgress       bool
		dryRun             bool
		fromStdin          bool
	)

	cmd := &cobra.Command{
		Use:   "upload <path>...",
		Short: "Upload files",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if forceMultipart && forceDirect {
				return ExitErrorf(2, "--force-multipart and --force-direct are mutually exclusive")
			}

			switch store {
			case "auto", "true", "false":
			default:
				return ExitErrorf(2, "invalid --store value: %q (must be \"auto\", \"true\", or \"false\")", store)
			}

			svc := fileSvc
			if svc == nil {
				var err error
				svc, err = fileServiceFromCmd(cmd)
				if err != nil {
					return err
				}
			}

			opts := formatOptionsFromCmd(cmd)
			formatter := output.New(opts)

			meta, err := parseMetadata(metadata)
			if err != nil {
				return &ExitError{Code: 2, Err: err}
			}

			paths := append([]string{}, args...)
			if fromStdin {
				stdinPaths, err := ReadLinesOrNDJSON(os.Stdin, "path")
				if err != nil {
					return fmt.Errorf("reading stdin: %w", err)
				}
				paths = append(paths, stdinPaths...)
			}

			if len(paths) == 0 {
				return ExitErrorf(2, "no file paths provided")
			}

			var entries []uploadFileEntry
			for _, path := range paths {
				fi, err := os.Stat(path)
				if err != nil {
					return ExitErrorf(2, "file %q: %v", path, err)
				}
				if !fi.Mode().IsRegular() {
					return ExitErrorf(2, "%q is not a regular file", path)
				}
				ct, err := detectContentType(path)
				if err != nil {
					return ExitErrorf(2, "detecting content type for %q: %v", path, err)
				}
				entries = append(entries, uploadFileEntry{
					path:        path,
					size:        fi.Size(),
					contentType: ct,
				})
			}

			if dryRun {
				return runUploadDryRun(cmd, entries, opts, formatter)
			}

			var threshold *int64
			if forceDirect {
				v := int64(0)
				threshold = &v
			} else if forceMultipart {
				v := int64(-1)
				threshold = &v
			} else {
				threshold = &multipartThreshold
			}

			var results []*service.File
			for _, entry := range entries {
				f, err := os.Open(entry.path)
				if err != nil {
					return err
				}

				var data io.ReadSeeker = f
				if showProgress {
					data = &progressReader{
						r:     f,
						total: entry.size,
						w:     cmd.ErrOrStderr(),
						label: entry.path,
					}
				}

				result, err := svc.Upload(cmd.Context(), service.UploadParams{
					Data:               data,
					Name:               baseName(entry.path),
					Size:               entry.size,
					ContentType:        entry.contentType,
					Store:              store,
					Metadata:           meta,
					MultipartThreshold: threshold,
				})
				f.Close()
				if err != nil {
					return fmt.Errorf("uploading %q: %w", entry.path, err)
				}

				if showProgress {
					fmt.Fprintln(cmd.ErrOrStderr())
				}

				results = append(results, result)
			}

			if len(results) == 1 {
				if opts.JSON {
					return formatter.Format(cmd.OutOrStdout(), results[0])
				}
				return formatter.Format(cmd.OutOrStdout(), fileInfoTable(results[0]))
			}

			if opts.JSON {
				return formatter.Format(cmd.OutOrStdout(), results)
			}

			table := output.NewTableData("UUID", "SIZE", "FILENAME")
			for _, r := range results {
				table.AddRow(r.UUID, strconv.FormatInt(r.Size, 10), r.Filename)
			}
			return formatter.Format(cmd.OutOrStdout(), table)
		},
	}

	f := cmd.Flags()
	f.StringVar(&store, "store", "auto", "File storage behavior (auto, true, false)")
	f.StringSliceVar(&metadata, "metadata", nil, "Metadata key=value pairs (repeatable)")
	f.Int64Var(&multipartThreshold, "multipart-threshold", 10485760, "Size threshold for multipart upload in bytes")
	f.BoolVar(&forceMultipart, "force-multipart", false, "Force multipart upload")
	f.BoolVar(&forceDirect, "force-direct", false, "Force direct upload")
	f.BoolVar(&showProgress, "progress", false, "Show upload progress on stderr")
	f.BoolVar(&dryRun, "dry-run", false, "Validate without uploading")
	f.BoolVar(&fromStdin, "from-stdin", false, "Read file paths from stdin")

	return cmd
}

func runUploadDryRun(cmd *cobra.Command, entries []uploadFileEntry, opts output.FormatOptions, formatter output.Formatter) error {
	type dryRunEntry struct {
		Path        string `json:"path"`
		Size        int64  `json:"size"`
		ContentType string `json:"content_type"`
	}
	var dryEntries []dryRunEntry
	for _, e := range entries {
		dryEntries = append(dryEntries, dryRunEntry{
			Path:        e.path,
			Size:        e.size,
			ContentType: e.contentType,
		})
	}

	if opts.JSON {
		return formatter.Format(cmd.OutOrStdout(), dryEntries)
	}

	table := output.NewTableData("PATH", "SIZE", "CONTENT-TYPE")
	for _, e := range dryEntries {
		table.AddRow(e.Path, strconv.FormatInt(e.Size, 10), e.ContentType)
	}
	return formatter.Format(cmd.OutOrStdout(), table)
}

func parseMetadata(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			return nil, fmt.Errorf("invalid metadata format %q, expected key=value", pair)
		}
		m[k] = v
	}
	return m, nil
}

func detectContentType(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}

// baseName returns the base name of a file path.
func baseName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

// progressReader wraps an io.ReadSeeker and reports progress to stderr.
type progressReader struct {
	r     io.ReadSeeker
	total int64
	read  atomic.Int64
	w     io.Writer
	label string
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	if n > 0 {
		current := p.read.Add(int64(n))
		pct := float64(current) / float64(p.total) * 100
		fmt.Fprintf(p.w, "\r%s: %.1f%% (%d/%d bytes)", p.label, pct, current, p.total)
	}
	return n, err
}

func (p *progressReader) Seek(offset int64, whence int) (int64, error) {
	pos, err := p.r.Seek(offset, whence)
	if err == nil {
		p.read.Store(pos)
	}
	return pos, err
}
