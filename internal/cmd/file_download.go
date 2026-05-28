package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/uploadcare/uploadcare-cli/internal/output"
	"github.com/uploadcare/uploadcare-cli/internal/service"
	"github.com/uploadcare/uploadcare-cli/internal/validate"
)

// errTargetAppeared signals that the destination existed at final install
// time even though it was missing during the earlier skip check. With
// --replace=false we honor the pre-existing file and report skipped.
var errTargetAppeared = errors.New("target file appeared during download")

const (
	defaultNameTemplate = "${uuid}${ext}"
	maxParallel         = 32
)

var templatePlaceholderRe = regexp.MustCompile(`\$\{\w+\}`)

type downloadRow struct {
	UUID      string `json:"uuid"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	SourceURL string `json:"source_url"`
	Status    string `json:"status"` // downloaded, skipped, dry-run
}

type downloadConfig struct {
	outputPath   string
	outputDir    string
	nameTemplate string
	effects      string
	replace      bool
	dryRun       bool
	fromStdin    bool
	showProgress bool
	parallel     int
}

func newFileDownloadCmd(fileSvc service.FileService) *cobra.Command {
	var cfg downloadConfig

	cmd := &cobra.Command{
		Use:   "download <uuid>...",
		Short: "Download files from the CDN",
		Long: `Download one or more files from the Uploadcare CDN to the local filesystem.

Accepts UUIDs as positional arguments, from stdin (--from-stdin), or both.

Single-UUID mode:
  --output <path>    Write to a specific path. Parent dirs are created.
  --output -         Stream the file to stdout (incompatible with --json).
  (omitted)          Write to ./<filename> (or ./<uuid> if no filename).

Batch mode (multiple UUIDs or --from-stdin):
  --output-dir <dir>          Required. Files written under this directory.
  --name-template <pattern>   Filename template. Default: ${uuid}${ext}
                              Placeholders: ${uuid}, ${filename}, ${ext}, ${effects}

Use --replace to overwrite existing files; without it, existing paths are
skipped (status=skipped).

Use --effects to apply Uploadcare CDN transformations to images on download
(e.g. "resize/200x/-/rotate/90/"). Effects are silently ignored for non-image
files.

Use --dry-run to resolve filenames and target paths without downloading.

Use --progress to print streaming progress to stderr. Use --parallel N to
download up to N files concurrently (default 1, max 32).

JSON fields: uuid, path, size, source_url, status.`,
		Example: `  # Download one file to ./<filename>
  uploadcare file download 740e1b8c-1ad8-4324-b7ec-112345678900

  # Download to a specific path
  uploadcare file download <uuid> --output ./photo.jpg

  # Stream to stdout
  uploadcare file download <uuid> --output - > photo.jpg

  # Mirror all stored files into ./backup, 8 concurrent workers
  uploadcare file list --page-all --stored true --json uuid \
    | uploadcare file download --from-stdin --output-dir ./backup --parallel 8

  # Download with CDN transformation (resize image to 200px wide)
  uploadcare file download <uuid> --output thumb.jpg --effects resize/200x/

  # Dry run: show what would be downloaded
  uploadcare file download <uuid1> <uuid2> --output-dir ./out --dry-run --json all`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFileDownload(cmd, args, fileSvc, cfg)
		},
	}

	f := cmd.Flags()
	f.StringVar(&cfg.outputPath, "output", "", "Output path for a single file (use '-' for stdout)")
	f.StringVar(&cfg.outputDir, "output-dir", "", "Output directory for batch downloads")
	f.StringVar(&cfg.nameTemplate, "name-template", "", "Filename template for batch mode (default: ${uuid}${ext})")
	f.StringVar(&cfg.effects, "effects", "", "CDN effects path to apply to images (e.g. resize/200x/)")
	f.BoolVar(&cfg.replace, "replace", false, "Overwrite existing files")
	f.BoolVar(&cfg.dryRun, "dry-run", false, "Resolve paths without downloading")
	f.BoolVar(&cfg.fromStdin, "from-stdin", false, "Read UUIDs from stdin")
	f.BoolVar(&cfg.showProgress, "progress", false, "Show download progress on stderr")
	f.IntVar(&cfg.parallel, "parallel", 1, "Number of concurrent downloads (max 32)")

	return cmd
}

func runFileDownload(cmd *cobra.Command, args []string, fileSvc service.FileService, cfg downloadConfig) error {
	if err := cfg.validateStatic(); err != nil {
		return err
	}

	opts := formatOptionsFromCmd(cmd)
	if err := opts.Validate(); err != nil {
		return &ExitError{Code: 2, Err: err}
	}

	uuids, err := collectUUIDs(args, cfg.fromStdin)
	if err != nil {
		return err
	}
	if len(uuids) == 0 {
		return ExitErrorf(2, "no UUIDs provided")
	}
	for _, u := range uuids {
		if err := validate.UUID(u); err != nil {
			return &ExitError{Code: 2, Err: err}
		}
	}
	if err := cfg.validateForInput(uuids, opts); err != nil {
		return err
	}
	if cfg.nameTemplate == "" {
		cfg.nameTemplate = defaultNameTemplate
	}

	svc := fileSvc
	if svc == nil {
		svc, err = fileServiceFromCmd(cmd)
		if err != nil {
			return err
		}
	}

	runner := downloadRunner{
		svc:      svc,
		cfg:      cfg,
		stdout:   cmd.OutOrStdout(),
		verbose:  output.NewVerboseLogger(opts.Verbose, cmd.ErrOrStderr()),
		reporter: newProgressReporter(cmd.ErrOrStderr(), cfg.showProgress && !opts.Quiet && !cfg.stdoutSink(), cfg.parallel),
	}

	rows, err := runner.run(cmd.Context(), uuids)
	if err != nil {
		return err
	}
	return writeDownloadRows(cmd.OutOrStdout(), output.New(opts), opts, rows, cfg.stdoutSink())
}

func (c downloadConfig) validateStatic() error {
	if c.outputPath != "" && c.outputDir != "" {
		return ExitErrorf(2, "--output and --output-dir are mutually exclusive")
	}
	if c.parallel < 1 {
		return ExitErrorf(2, "--parallel must be >= 1")
	}
	if c.parallel > maxParallel {
		return ExitErrorf(2, "--parallel must be <= %d", maxParallel)
	}
	return nil
}

func (c downloadConfig) validateForInput(uuids []string, opts output.FormatOptions) error {
	batch := len(uuids) > 1 || c.fromStdin
	if batch && c.outputPath != "" {
		return ExitErrorf(2, "--output is only valid for a single UUID; use --output-dir for batch downloads")
	}
	if batch && c.outputDir == "" {
		return ExitErrorf(2, "batch downloads require --output-dir")
	}
	if c.stdoutSink() && opts.JSON {
		return ExitErrorf(2, "--output - cannot be combined with --json")
	}
	if c.stdoutSink() && c.parallel > 1 {
		return ExitErrorf(2, "--output - cannot be combined with --parallel > 1")
	}
	return nil
}

func (c downloadConfig) stdoutSink() bool {
	return c.outputPath == "-"
}

type downloadRunner struct {
	svc      service.FileService
	cfg      downloadConfig
	stdout   io.Writer
	verbose  *output.VerboseLogger
	reporter *progressReporter
}

func (r *downloadRunner) run(ctx context.Context, uuids []string) ([]downloadRow, error) {
	plans, err := r.plan(ctx, uuids)
	if err != nil {
		return nil, err
	}

	rows := make([]downloadRow, len(plans))
	err = runDownloadJobs(ctx, len(plans), r.cfg.parallel, func(ctx context.Context, idx int) error {
		row, err := r.download(ctx, plans[idx])
		rows[idx] = row
		return err
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *downloadRunner) plan(ctx context.Context, uuids []string) ([]downloadPlan, error) {
	plans := make([]downloadPlan, len(uuids))
	seenPath := make(map[string]string, len(uuids))

	for i, uuid := range uuids {
		info, err := r.svc.Info(ctx, uuid, false)
		if err != nil {
			return nil, fmt.Errorf("looking up %s: %w", uuid, err)
		}
		localPath, err := resolveLocalPath(info, r.cfg)
		if err != nil {
			return nil, &ExitError{Code: 2, Err: err}
		}
		if localPath != "" {
			if owner, dup := seenPath[localPath]; dup {
				return nil, ExitErrorf(2,
					"duplicate target path %q: UUIDs %s and %s resolve to the same local file (adjust --name-template to disambiguate)",
					localPath, owner, uuid,
				)
			}
			seenPath[localPath] = uuid
		}
		plans[i] = downloadPlan{UUID: uuid, File: info, LocalPath: localPath}
	}

	return plans, nil
}

func runDownloadJobs(ctx context.Context, count, parallel int, run func(context.Context, int) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan int)
	var wg sync.WaitGroup
	var firstErr error
	var firstErrOnce sync.Once

	for range min(parallel, count) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for idx := range jobs {
				if ctx.Err() != nil {
					continue
				}
				if err := run(ctx, idx); err != nil {
					firstErrOnce.Do(func() {
						firstErr = err
						cancel()
					})
				}
			}
		}()
	}

	for i := 0; i < count && ctx.Err() == nil; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return firstErr
}

func writeDownloadRows(w io.Writer, formatter output.Formatter, opts output.FormatOptions, rows []downloadRow, stdoutSink bool) error {
	if stdoutSink {
		return nil
	}
	if opts.JSON {
		if len(rows) == 1 {
			return formatter.Format(w, rows[0])
		}
		return formatter.Format(w, rows)
	}
	return formatter.Format(w, downloadRowsTable(rows))
}

type downloadPlan struct {
	UUID      string
	File      *service.File
	LocalPath string // empty when writing to stdout
}

// resolveLocalPath computes the destination path for a UUID given the
// configured output flags. Returns "" when writing to stdout.
//
// Path-traversal handling: explicit user-typed paths (--output, --output-dir)
// are trusted as-is. When the path is generated from server-controlled data
// (template expansion or default filename), we guard against the generated
// portion escaping the user's chosen output directory.
func resolveLocalPath(info *service.File, cfg downloadConfig) (string, error) {
	if cfg.stdoutSink() {
		return "", nil
	}
	if cfg.outputPath != "" {
		return cfg.outputPath, nil
	}
	if cfg.outputDir != "" {
		expanded, err := expandNameTemplate(cfg.nameTemplate, info, cfg.effects)
		if err != nil {
			return "", err
		}
		joined := filepath.Join(cfg.outputDir, expanded)
		if _, err := pathInside(cfg.outputDir, joined); err != nil {
			return "", fmt.Errorf("template expansion %q escapes --output-dir %q", expanded, cfg.outputDir)
		}
		return joined, nil
	}
	// Default mode: write to ./<filename>. The filename is server-controlled,
	// so reject anything that would push the write outside the current
	// directory (or contains a path separator at all).
	name := info.Filename
	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		name = info.UUID
	}
	return name, nil
}

func pathInside(parent, target string) (string, error) {
	cleanedParent := filepath.Clean(parent)
	cleanedTarget := filepath.Clean(target)
	rel, err := filepath.Rel(cleanedParent, cleanedTarget)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is outside %s", cleanedTarget, cleanedParent)
	}
	return rel, nil
}

// ensureSafeOutputDirTarget verifies that every intermediate path component
// between parent (exclusive) and target — that *currently exists* — is a
// regular directory, never a symlink. This prevents a symlink planted
// inside --output-dir from redirecting the write outside the directory
// during MkdirAll, CreateTemp, or rename.
//
// Components that don't yet exist are fine: they'll be created by
// MkdirAll, which itself never creates symlinks.
func ensureSafeOutputDirTarget(parent, target string) error {
	cleanedParent := filepath.Clean(parent)
	rel, err := pathInside(parent, target)
	if err != nil {
		return err
	}
	parts := strings.Split(rel, string(filepath.Separator))
	cur := cleanedParent
	for _, p := range parts[:len(parts)-1] {
		cur = filepath.Join(cur, p)
		info, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("intermediate component %s is a symlink", cur)
		}
	}
	return nil
}

func (r *downloadRunner) download(ctx context.Context, plan downloadPlan) (downloadRow, error) {
	info := plan.File
	localPath := plan.LocalPath
	row := downloadRow{
		UUID: plan.UUID,
		Path: localPath,
		Size: info.Size,
	}

	if r.cfg.dryRun {
		row.SourceURL = computeSourceURL(info, r.cfg.effects)
		row.Status = "dry-run"
		return row, nil
	}

	out := r.stdout
	var target *downloadTarget

	if !r.cfg.stdoutSink() {
		// Lstat (not Stat) so a dangling symlink at localPath is treated as
		// "exists" — we must not follow it and clobber whatever it points at.
		if !r.cfg.replace {
			if _, err := os.Lstat(localPath); err == nil {
				r.verbose.Infof("skipped: %s exists (use --replace to overwrite)", localPath)
				row.Status = "skipped"
				row.SourceURL = computeSourceURL(info, r.cfg.effects)
				return row, nil
			}
		}

		if r.cfg.outputDir != "" {
			if err := ensureSafeOutputDirTarget(r.cfg.outputDir, localPath); err != nil {
				return row, &ExitError{Code: 2, Err: fmt.Errorf("refusing to write to %s: %w", localPath, err)}
			}
		}

		if err := ensureParentDir(localPath); err != nil {
			return row, err
		}

		var err error
		target, err = newDownloadTarget(localPath, r.cfg.replace)
		if err != nil {
			return row, err
		}
		out = target.file
	}

	var progressCb func(bytesSoFar, totalBytes int64)
	if r.reporter != nil && r.reporter.enabled {
		progressCb = r.reporter.callback(localPath)
	}

	res, err := r.svc.Download(ctx, service.DownloadParams{
		UUID:     plan.UUID,
		Out:      out,
		Effects:  r.cfg.effects,
		Progress: progressCb,
		Resolved: info,
	})

	var finalizeErr error
	if target != nil {
		finalizeErr = target.finish(err == nil)
	}
	if err != nil {
		return row, fmt.Errorf("downloading %s: %w", plan.UUID, err)
	}
	if finalizeErr != nil {
		if errors.Is(finalizeErr, errTargetAppeared) {
			r.verbose.Infof("skipped: %s appeared during download (use --replace to overwrite)", localPath)
			row.Status = "skipped"
			row.SourceURL = computeSourceURL(info, r.cfg.effects)
			return row, nil
		}
		return row, fmt.Errorf("finalizing %s: %w", plan.UUID, finalizeErr)
	}
	if r.reporter != nil {
		r.reporter.finish(localPath)
	}
	row.SourceURL = res.SourceURL
	row.Size = res.BytesCopied
	row.Status = "downloaded"
	return row, nil
}

func ensureParentDir(path string) error {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
	}
	return nil
}

type downloadTarget struct {
	file      *os.File
	finalPath string
	tmpPath   string
	replace   bool
}

func newDownloadTarget(localPath string, replace bool) (*downloadTarget, error) {
	// Use a unique temp suffix so that batch targets which themselves look
	// like another job's ".partial" don't clobber each other.
	fp, err := os.CreateTemp(filepath.Dir(localPath), filepath.Base(localPath)+".*.partial")
	if err != nil {
		return nil, fmt.Errorf("creating temp file for %s: %w", localPath, err)
	}
	return &downloadTarget{
		file:      fp,
		finalPath: localPath,
		tmpPath:   fp.Name(),
		replace:   replace,
	}, nil
}

func (t *downloadTarget) finish(success bool) error {
	closeErr := t.file.Close()
	if !success {
		_ = os.Remove(t.tmpPath)
		return nil
	}
	if closeErr != nil {
		_ = os.Remove(t.tmpPath)
		return fmt.Errorf("closing %s: %w", t.tmpPath, closeErr)
	}
	if t.replace {
		if err := os.Rename(t.tmpPath, t.finalPath); err != nil {
			_ = os.Remove(t.tmpPath)
			return fmt.Errorf("renaming %s -> %s: %w", t.tmpPath, t.finalPath, err)
		}
		return nil
	}

	// No-replace install: os.Link is the atomic "create-if-absent" primitive
	// on Unix. It fails with EEXIST for regular files, directories, and
	// dangling symlinks alike, so a target that raced into existence is
	// honored rather than clobbered.
	if err := os.Link(t.tmpPath, t.finalPath); err != nil {
		_ = os.Remove(t.tmpPath)
		if errors.Is(err, os.ErrExist) {
			return errTargetAppeared
		}
		return fmt.Errorf("linking %s -> %s: %w", t.tmpPath, t.finalPath, err)
	}
	_ = os.Remove(t.tmpPath)
	return nil
}

// effectsForFile returns the configured effects only for image files,
// matching the service layer's behavior. Non-image files always download
// untransformed.
func effectsForFile(info *service.File, effects string) string {
	if !info.IsImage {
		return ""
	}
	return effects
}

// computeSourceURL is kept for the existing test surface; production code
// uses service.ApplyCDNEffects + effectsForFile directly.
func computeSourceURL(info *service.File, effects string) string {
	return service.ApplyCDNEffects(info.OriginalFileURL, info.UUID, effectsForFile(info, effects))
}

// sanitizeServerSegment neutralizes path separators and bare dot-segments
// in a server-controlled string so that the result is safe as a single
// filename component. Without this, a server filename like "evil/x.bin"
// could combine with a symlink under --output-dir to escape the directory.
func sanitizeServerSegment(s string) string {
	out := strings.NewReplacer("/", "_", "\\", "_", "\x00", "_").Replace(s)
	if out == "." || out == ".." {
		return "_"
	}
	return out
}

func expandNameTemplate(tmpl string, f *service.File, effects string) (string, error) {
	safeFilename := sanitizeServerSegment(f.Filename)
	ext := ""
	if safeFilename != "" {
		ext = filepath.Ext(safeFilename)
	}
	effectsSlug := strings.ReplaceAll(strings.Trim(effects, "/"), "/", "_")

	out := templatePlaceholderRe.ReplaceAllStringFunc(tmpl, func(m string) string {
		switch m {
		case "${uuid}":
			return f.UUID
		case "${filename}":
			return safeFilename
		case "${ext}":
			return ext
		case "${effects}":
			return effectsSlug
		default:
			return m
		}
	})

	// Reject control characters defensively (the server should never produce
	// them in filenames, but the JSON API doesn't strictly forbid it).
	// Path-traversal in the expansion is permitted here and policed by the
	// outputDir-containment check in resolveLocalPath.
	for i, r := range out {
		if r == '\t' || r == '\n' {
			continue
		}
		if (r >= 0x00 && r <= 0x1F) || r == 0x7F {
			return "", fmt.Errorf("template expansion %q: control character 0x%02X at position %d", out, r, i)
		}
	}
	if out == "" {
		return "", fmt.Errorf("expanded template is empty")
	}
	return out, nil
}

func downloadRowsTable(rows []downloadRow) *output.TableData {
	t := output.NewTableData("UUID", "PATH", "SIZE", "STATUS")
	for _, r := range rows {
		t.AddRow(r.UUID, r.Path, strconv.FormatInt(r.Size, 10), r.Status)
	}
	return t
}

// progressReporter rate-limits and serializes per-file progress updates.
type progressReporter struct {
	w        io.Writer
	enabled  bool
	parallel int
	mu       sync.Mutex
	last     map[string]time.Time
}

func newProgressReporter(w io.Writer, enabled bool, parallel int) *progressReporter {
	return &progressReporter{
		w:        w,
		enabled:  enabled,
		parallel: parallel,
		last:     make(map[string]time.Time),
	}
}

const progressMinInterval = 100 * time.Millisecond

func (p *progressReporter) callback(label string) func(bytesSoFar, totalBytes int64) {
	return func(soFar, total int64) {
		now := time.Now()
		p.mu.Lock()
		if last, ok := p.last[label]; ok && now.Sub(last) < progressMinInterval && soFar != total {
			p.mu.Unlock()
			return
		}
		p.last[label] = now
		var pct float64
		if total > 0 {
			pct = float64(soFar) / float64(total) * 100
		}
		if p.parallel <= 1 {
			_, _ = fmt.Fprintf(p.w, "\r%s: %.1f%% (%d/%d bytes)", label, pct, soFar, total)
		} else {
			_, _ = fmt.Fprintf(p.w, "%s: %.1f%% (%d/%d bytes)\n", label, pct, soFar, total)
		}
		p.mu.Unlock()
	}
}

func (p *progressReporter) finish(label string) {
	if !p.enabled {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.last, label)
	if p.parallel <= 1 {
		_, _ = fmt.Fprintln(p.w)
	}
}
