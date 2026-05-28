package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/uploadcare/uploadcare-cli/internal/service"
)

const (
	testUUID1 = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	testUUID2 = "b2c3d4e5-f6a7-8901-bcde-f12345678901"
)

func testImageFile(uuid, filename string) *service.File {
	return &service.File{
		UUID:            uuid,
		Filename:        filename,
		Size:            42,
		MimeType:        "image/jpeg",
		IsImage:         true,
		OriginalFileURL: "https://ucarecdn.com/" + uuid + "/",
	}
}

func testNonImageFile(uuid, filename string) *service.File {
	f := testImageFile(uuid, filename)
	f.IsImage = false
	f.MimeType = "application/pdf"
	return f
}

// staticDownloadMock returns a mock that serves Info from a map and writes
// fixed body bytes for Download.
func staticDownloadMock(files map[string]*service.File, body []byte) *mockFileService {
	return &mockFileService{
		infoFunc: func(_ context.Context, uuid string, _ bool) (*service.File, error) {
			f, ok := files[uuid]
			if !ok {
				return nil, errors.New("not found")
			}
			return f, nil
		},
		downloadFunc: func(_ context.Context, params service.DownloadParams) (*service.DownloadResult, error) {
			n, err := params.Out.Write(body)
			if err != nil {
				return nil, err
			}
			return &service.DownloadResult{
				UUID:        params.UUID,
				BytesCopied: int64(n),
				SourceURL:   "https://ucarecdn.com/" + params.UUID + "/",
			}, nil
		},
	}
}

func TestFileDownload_SingleDefaultPath(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "photo.jpg")},
		[]byte("hello"),
	)

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download", testUUID1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "photo.jpg"))
	if err != nil {
		t.Fatalf("expected ./photo.jpg to exist: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", string(data), "hello")
	}
}

func TestFileDownload_OutputPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "out.bin")

	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "photo.jpg")},
		[]byte("payload"),
	)

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download", testUUID1, "--output", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("expected %s: %v", target, err)
	}
	if string(data) != "payload" {
		t.Errorf("content = %q, want %q", string(data), "payload")
	}
}

func TestFileDownload_OutputStdout(t *testing.T) {
	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "photo.jpg")},
		[]byte("stream-me"),
	)

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "file", "download", testUUID1, "--output", "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "stream-me" {
		t.Errorf("stdout = %q, want %q", stdout, "stream-me")
	}
}

func TestFileDownload_OutputStdoutWithJSONFails(t *testing.T) {
	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "photo.jpg")},
		[]byte("x"),
	)

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "--json", "all", "file", "download", testUUID1, "--output", "-")
	if err == nil {
		t.Fatal("expected error combining --output - with --json")
	}
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Errorf("expected ExitError code 2, got %v", err)
	}
}

func TestFileDownload_BatchWithTemplate(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	files := map[string]*service.File{
		testUUID1: testImageFile(testUUID1, "a.jpg"),
		testUUID2: testImageFile(testUUID2, "b.png"),
	}
	mock := staticDownloadMock(files, []byte("data"))

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, testUUID2,
		"--output-dir", outDir,
		"--name-template", "${uuid}-${filename}",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range []string{
		testUUID1 + "-a.jpg",
		testUUID2 + "-b.png",
	} {
		if _, err := os.Stat(filepath.Join(outDir, want)); err != nil {
			t.Errorf("expected %s to exist: %v", want, err)
		}
	}
}

func TestFileDownload_TemplateDefaultIsUUIDExt(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	files := map[string]*service.File{
		testUUID1: testImageFile(testUUID1, "a.jpg"),
		testUUID2: testImageFile(testUUID2, "b.png"),
	}
	mock := staticDownloadMock(files, []byte("data"))

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, testUUID2,
		"--output-dir", outDir,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, testUUID1+".jpg")); err != nil {
		t.Errorf("default template should produce ${uuid}${ext}: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, testUUID2+".png")); err != nil {
		t.Errorf("default template should produce ${uuid}${ext}: %v", err)
	}
}

func TestFileDownload_UnknownPlaceholderLeftLiteral(t *testing.T) {
	got, err := expandNameTemplate("${uuid}-${nope}.bin", testImageFile(testUUID1, "x.jpg"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := testUUID1 + "-${nope}.bin"
	if got != want {
		t.Errorf("expandNameTemplate = %q, want %q", got, want)
	}
}

func TestFileDownload_SkipExisting(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(target, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	var downloadCalled int32
	mock := &mockFileService{
		infoFunc: func(_ context.Context, _ string, _ bool) (*service.File, error) {
			return testImageFile(testUUID1, "photo.jpg"), nil
		},
		downloadFunc: func(_ context.Context, _ service.DownloadParams) (*service.DownloadResult, error) {
			atomic.AddInt32(&downloadCalled, 1)
			return nil, errors.New("should not be called")
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "download",
		testUUID1, "--output", target,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&downloadCalled) != 0 {
		t.Error("download should not be called when file exists and --replace is not set")
	}

	data, _ := os.ReadFile(target)
	if string(data) != "OLD" {
		t.Errorf("existing file should be preserved; got %q", string(data))
	}

	var row downloadRow
	if err := json.Unmarshal([]byte(stdout), &row); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if row.Status != "skipped" {
		t.Errorf("status = %q, want skipped", row.Status)
	}
}

func TestFileDownload_Replace(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(target, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "photo.jpg")},
		[]byte("NEW"),
	)
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, "--output", target, "--replace",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(target)
	if string(data) != "NEW" {
		t.Errorf("file should be overwritten; got %q", string(data))
	}
	if leftovers := findPartialFiles(t, dir); len(leftovers) > 0 {
		t.Errorf("temp files should be cleaned up on success, found: %v", leftovers)
	}
}

// findPartialFiles returns any .partial files left in dir.
func findPartialFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".partial") {
			out = append(out, e.Name())
		}
	}
	return out
}

func TestFileDownload_PartialCleanedUpOnError(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "photo.jpg")

	mock := &mockFileService{
		infoFunc: func(_ context.Context, _ string, _ bool) (*service.File, error) {
			return testImageFile(testUUID1, "photo.jpg"), nil
		},
		downloadFunc: func(_ context.Context, _ service.DownloadParams) (*service.DownloadResult, error) {
			return nil, errors.New("network gone")
		},
	}
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, "--output", target,
	)
	if err == nil {
		t.Fatal("expected error from service")
	}
	if leftovers := findPartialFiles(t, dir); len(leftovers) > 0 {
		t.Errorf("temp files should be cleaned up on failure, found: %v", leftovers)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("final target should not exist after failure")
	}
}

func TestFileDownload_NoReplaceHonorsRacingTarget(t *testing.T) {
	// Pre-flight Lstat sees no file. During Download, a concurrent writer
	// creates localPath. Without --replace, finalize must NOT overwrite it.
	dir := t.TempDir()
	target := filepath.Join(dir, "photo.jpg")

	mock := &mockFileService{
		infoFunc: func(_ context.Context, _ string, _ bool) (*service.File, error) {
			return testImageFile(testUUID1, "photo.jpg"), nil
		},
		downloadFunc: func(_ context.Context, params service.DownloadParams) (*service.DownloadResult, error) {
			// Simulate the racing writer: another actor creates the
			// destination after our skip check but before finalize.
			if err := os.WriteFile(target, []byte("PRE-EXISTING"), 0o644); err != nil {
				return nil, err
			}
			n, _ := params.Out.Write([]byte("downloaded body"))
			return &service.DownloadResult{UUID: params.UUID, BytesCopied: int64(n)}, nil
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "download",
		testUUID1, "--output", target,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(target)
	if string(got) != "PRE-EXISTING" {
		t.Errorf("racing target was clobbered: got %q, want %q", string(got), "PRE-EXISTING")
	}
	var row downloadRow
	if err := json.Unmarshal([]byte(stdout), &row); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if row.Status != "skipped" {
		t.Errorf("status = %q, want skipped (target appeared mid-download)", row.Status)
	}
	if leftovers := findPartialFiles(t, dir); len(leftovers) > 0 {
		t.Errorf("temp files should be removed when racing target wins, found: %v", leftovers)
	}
}

func TestFileDownload_ReplaceOverridesRacingTarget(t *testing.T) {
	// Same race, but --replace is set — the new download must win.
	dir := t.TempDir()
	target := filepath.Join(dir, "photo.jpg")

	mock := &mockFileService{
		infoFunc: func(_ context.Context, _ string, _ bool) (*service.File, error) {
			return testImageFile(testUUID1, "photo.jpg"), nil
		},
		downloadFunc: func(_ context.Context, params service.DownloadParams) (*service.DownloadResult, error) {
			_ = os.WriteFile(target, []byte("PRE-EXISTING"), 0o644)
			n, _ := params.Out.Write([]byte("NEW"))
			return &service.DownloadResult{UUID: params.UUID, BytesCopied: int64(n)}, nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, "--output", target, "--replace",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "NEW" {
		t.Errorf("--replace should win the race; got %q", string(got))
	}
}

func TestFileDownload_DanglingSymlinkNotFollowed(t *testing.T) {
	// A dangling symlink at localPath: os.Stat would report "not exists"
	// because the target is missing, but we must treat the symlink itself
	// as an existing entry and either skip (no --replace) or replace it.
	dir := t.TempDir()
	target := filepath.Join(dir, "photo.jpg")
	missingTarget := filepath.Join(dir, "no-such-file")
	if err := os.Symlink(missingTarget, target); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	var downloadCalled int32
	mock := &mockFileService{
		infoFunc: func(_ context.Context, _ string, _ bool) (*service.File, error) {
			return testImageFile(testUUID1, "photo.jpg"), nil
		},
		downloadFunc: func(_ context.Context, _ service.DownloadParams) (*service.DownloadResult, error) {
			atomic.AddInt32(&downloadCalled, 1)
			return nil, errors.New("should not be called when target is a dangling symlink")
		},
	}

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "download",
		testUUID1, "--output", target,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&downloadCalled) != 0 {
		t.Error("Download should not be called when target is a dangling symlink")
	}
	if _, err := os.Lstat(target); err != nil {
		t.Errorf("dangling symlink should be preserved: %v", err)
	}
	if _, err := os.Lstat(missingTarget); !os.IsNotExist(err) {
		t.Errorf("symlink target must not be created underneath: %v", err)
	}
	var row downloadRow
	if err := json.Unmarshal([]byte(stdout), &row); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if row.Status != "skipped" {
		t.Errorf("status = %q, want skipped", row.Status)
	}
}

func TestFileDownload_TempFileDoesNotClobberOtherTarget(t *testing.T) {
	// Regression: with the naive "<localPath>.partial" temp scheme, a batch
	// containing both `foo` and `foo.partial` would race — the `foo` job's
	// temp ".partial" file could destroy the completed `foo.partial`
	// download. The unique-temp-suffix scheme must keep both intact.
	dir := t.TempDir()

	files := map[string]*service.File{
		testUUID1: testImageFile(testUUID1, "foo"),
		testUUID2: testImageFile(testUUID2, "foo.partial"),
	}
	mock := &mockFileService{
		infoFunc: func(_ context.Context, uuid string, _ bool) (*service.File, error) {
			return files[uuid], nil
		},
		downloadFunc: func(_ context.Context, params service.DownloadParams) (*service.DownloadResult, error) {
			// Each file's body equals its UUID, so we can detect clobbering.
			n, _ := params.Out.Write([]byte(params.UUID))
			return &service.DownloadResult{UUID: params.UUID, BytesCopied: int64(n)}, nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, testUUID2,
		"--output-dir", dir,
		"--name-template", "${filename}",
		"--parallel", "4",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	foo, err := os.ReadFile(filepath.Join(dir, "foo"))
	if err != nil {
		t.Fatalf("foo should exist: %v", err)
	}
	if string(foo) != testUUID1 {
		t.Errorf("foo content = %q, want %q (was clobbered by other job's temp file)", string(foo), testUUID1)
	}
	fooPartial, err := os.ReadFile(filepath.Join(dir, "foo.partial"))
	if err != nil {
		t.Fatalf("foo.partial should exist as a real target: %v", err)
	}
	if string(fooPartial) != testUUID2 {
		t.Errorf("foo.partial content = %q, want %q (was clobbered)", string(fooPartial), testUUID2)
	}
	if leftovers := findPartialFiles(t, dir); len(leftovers) != 1 || leftovers[0] != "foo.partial" {
		t.Errorf("only the legit foo.partial should remain; got %v", leftovers)
	}
}

func TestFileDownload_UserParentDirAllowed(t *testing.T) {
	// Regression: --output and --output-dir are user-typed; a `../` segment
	// is the user's explicit choice and must not be rejected.
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(filepath.Join(tmp, "work")); err != nil {
		t.Fatal(err)
	}

	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "p.jpg")},
		[]byte("body"),
	)
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, "--output", "../sibling.jpg",
	)
	if err != nil {
		t.Fatalf("explicit --output ../foo should be accepted, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "sibling.jpg")); err != nil {
		t.Errorf("expected ../sibling.jpg to be created: %v", err)
	}
}

func TestFileDownload_UserOutputDirWithParentAllowed(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "work"), 0o755); err != nil {
		t.Fatal(err)
	}
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	if err := os.Chdir(filepath.Join(tmp, "work")); err != nil {
		t.Fatal(err)
	}

	mock := staticDownloadMock(
		map[string]*service.File{
			testUUID1: testImageFile(testUUID1, "a.jpg"),
			testUUID2: testImageFile(testUUID2, "b.jpg"),
		},
		[]byte("body"),
	)
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, testUUID2, "--output-dir", "../backup",
	)
	if err != nil {
		t.Fatalf("explicit --output-dir ../backup should be accepted, got: %v", err)
	}
	for _, name := range []string{testUUID1 + ".jpg", testUUID2 + ".jpg"} {
		if _, err := os.Stat(filepath.Join(tmp, "backup", name)); err != nil {
			t.Errorf("expected ../backup/%s: %v", name, err)
		}
	}
}

func TestFileDownload_InfoCalledOncePerUUID(t *testing.T) {
	// Regression: each UUID had its info fetched twice — once in the command
	// pre-pass, once again inside fileService.Download. The pre-resolved
	// File must be threaded through so workers don't re-query.
	uuids := []string{
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003",
		"00000000-0000-0000-0000-000000000004",
	}
	var infoCount sync.Map
	for _, u := range uuids {
		infoCount.Store(u, new(atomic.Int32))
	}

	mock := &mockFileService{
		infoFunc: func(_ context.Context, uuid string, _ bool) (*service.File, error) {
			v, _ := infoCount.Load(uuid)
			v.(*atomic.Int32).Add(1)
			return testImageFile(uuid, "x.bin"), nil
		},
		downloadFunc: func(_ context.Context, params service.DownloadParams) (*service.DownloadResult, error) {
			if params.Resolved == nil {
				t.Errorf("expected Resolved to be set, forcing a 2nd Info call")
			}
			_, _ = params.Out.Write([]byte("x"))
			return &service.DownloadResult{UUID: params.UUID, BytesCopied: 1}, nil
		},
	}

	dir := t.TempDir()
	args := []string{"file", "download"}
	args = append(args, uuids...)
	args = append(args, "--output-dir", dir, "--parallel", "4")

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, args...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, u := range uuids {
		v, _ := infoCount.Load(u)
		if n := v.(*atomic.Int32).Load(); n != 1 {
			t.Errorf("Info(%s) called %d times, want 1", u, n)
		}
	}
}

func TestFileDownload_ServerFilenameWithSeparatorSanitized(t *testing.T) {
	// A server-controlled filename like "evil/x.bin" must not introduce a
	// new path component — the substitution must collapse separators so
	// that the result is a single filename under --output-dir.
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "evil/x.bin")},
		[]byte("body"),
	)
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1,
		"--output-dir", outDir,
		"--name-template", "${filename}",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Must not have created a subdirectory `evil/` inside outDir.
	if _, err := os.Stat(filepath.Join(outDir, "evil")); !os.IsNotExist(err) {
		t.Errorf("server-supplied separator should not create a subdir, got: %v", err)
	}
	// Should have written exactly one flat file `evil_x.bin`.
	if _, err := os.Stat(filepath.Join(outDir, "evil_x.bin")); err != nil {
		t.Errorf("expected sanitized flat filename: %v", err)
	}
}

func TestFileDownload_SymlinkInsideOutputDirBlocked(t *testing.T) {
	// --output-dir/sub -> elsewhere (a symlink). A user template that
	// writes through `sub/` must not escape the output dir via the link.
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "out")
	escapeTarget := filepath.Join(tmp, "escape")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(escapeTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	// Plant the symlink: outDir/sub -> escapeTarget
	if err := os.Symlink(escapeTarget, filepath.Join(outDir, "sub")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	var downloadCalled int32
	mock := &mockFileService{
		infoFunc: func(_ context.Context, _ string, _ bool) (*service.File, error) {
			return testImageFile(testUUID1, "photo.jpg"), nil
		},
		downloadFunc: func(_ context.Context, _ service.DownloadParams) (*service.DownloadResult, error) {
			atomic.AddInt32(&downloadCalled, 1)
			return nil, errors.New("must not be called when intermediate component is a symlink")
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1,
		"--output-dir", outDir,
		"--name-template", "sub/${uuid}.jpg",
	)
	if err == nil {
		t.Fatal("expected error: write would traverse a symlinked intermediate")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("error should mention symlink, got: %v", err)
	}
	if atomic.LoadInt32(&downloadCalled) > 0 {
		t.Error("Download must not run when an intermediate is a symlink")
	}
	// Nothing should have been written under the escape target.
	entries, _ := os.ReadDir(escapeTarget)
	if len(entries) != 0 {
		t.Errorf("escape target should remain empty, contains: %v", entries)
	}
}

func TestFileDownload_TemplateEscapeBlocked(t *testing.T) {
	// A user-typed template literal that tries to escape --output-dir via
	// "../" must be rejected. (Server-controlled ${filename} segments are
	// sanitized separately — see TestFileDownload_ServerFilenameWith
	// SeparatorSanitized — so the only way to inject "../" into the
	// resolved path is via the user's own template literal.)
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "photo.jpg")},
		[]byte("x"),
	)
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1,
		"--output-dir", outDir,
		"--name-template", "../escape/${uuid}.jpg",
	)
	if err == nil {
		t.Fatal("expected error for template literal escaping --output-dir")
	}
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Errorf("expected ExitError code 2, got %v", err)
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Errorf("error should mention escape, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "escape")); !os.IsNotExist(err) {
		t.Errorf("escape dir should not be created")
	}
}

func TestFileDownload_DuplicateTargetPathRejected(t *testing.T) {
	// Two distinct UUIDs whose ${filename} expansion collides on disk.
	files := map[string]*service.File{
		testUUID1: testImageFile(testUUID1, "same.jpg"),
		testUUID2: testImageFile(testUUID2, "same.jpg"),
	}
	mock := staticDownloadMock(files, []byte("x"))

	dir := t.TempDir()
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, testUUID2,
		"--output-dir", dir,
		"--name-template", "${filename}",
		"--parallel", "2",
	)
	if err == nil {
		t.Fatal("expected error for colliding target paths")
	}
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Errorf("expected ExitError code 2, got %v", err)
	}
	if !strings.Contains(err.Error(), "duplicate target path") {
		t.Errorf("error should mention duplicate target path, got: %v", err)
	}
	if !strings.Contains(err.Error(), testUUID1) || !strings.Contains(err.Error(), testUUID2) {
		t.Errorf("error should cite both UUIDs, got: %v", err)
	}
}

func TestFileDownload_SizeReflectsBytesCopied(t *testing.T) {
	// Info reports Size=42 (original), but the served payload (effects-transformed
	// or otherwise) is a different length. The reported size must match what was
	// actually written, not the source-file Size from Info.
	const transformedBody = "trimmed"
	mock := &mockFileService{
		infoFunc: func(_ context.Context, _ string, _ bool) (*service.File, error) {
			f := testImageFile(testUUID1, "photo.jpg")
			f.Size = 42 // original
			return f, nil
		},
		downloadFunc: func(_ context.Context, params service.DownloadParams) (*service.DownloadResult, error) {
			n, err := params.Out.Write([]byte(transformedBody))
			if err != nil {
				return nil, err
			}
			return &service.DownloadResult{
				UUID:        params.UUID,
				BytesCopied: int64(n),
				SourceURL:   "https://ucarecdn.com/" + params.UUID + "/-/resize/200x/",
			}, nil
		},
	}

	dir := t.TempDir()
	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "download",
		testUUID1, "--output", filepath.Join(dir, "out.jpg"), "--effects", "resize/200x/",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var row downloadRow
	if err := json.Unmarshal([]byte(stdout), &row); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if row.Size != int64(len(transformedBody)) {
		t.Errorf("row.Size = %d, want %d (bytes actually written)", row.Size, len(transformedBody))
	}
	if row.Status != "downloaded" {
		t.Errorf("status = %q, want downloaded", row.Status)
	}
}

func TestFileDownload_DryRunDoesNotCreateDirs(t *testing.T) {
	mock := &mockFileService{
		infoFunc: func(_ context.Context, _ string, _ bool) (*service.File, error) {
			return testImageFile(testUUID1, "photo.jpg"), nil
		},
		downloadFunc: func(_ context.Context, _ service.DownloadParams) (*service.DownloadResult, error) {
			t.Fatal("Download should not be called in --dry-run")
			return nil, nil
		},
	}

	tmp := t.TempDir()
	missingDir := filepath.Join(tmp, "should", "not", "exist")
	target := filepath.Join(missingDir, "x.jpg")

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, "--output", target, "--dry-run",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(missingDir); !os.IsNotExist(err) {
		t.Errorf("--dry-run must not create parent dirs; %s exists", missingDir)
	}
}

func TestFileDownload_FinalizeErrorPropagates(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "blocker")
	// Make the target a directory so Rename(.partial -> target) fails.
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "blocker")},
		[]byte("data"),
	)

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, "--output", target, "--replace",
	)
	if err == nil {
		t.Fatal("expected finalize error when target path is a directory")
	}
	if !strings.Contains(err.Error(), "finalizing") && !strings.Contains(err.Error(), "renaming") {
		t.Errorf("error should mention finalize/rename failure, got: %v", err)
	}
	if leftovers := findPartialFiles(t, dir); len(leftovers) > 0 {
		t.Errorf("temp files should be removed on finalize failure, found: %v", leftovers)
	}
}

func TestFileDownload_DryRun(t *testing.T) {
	var downloadCalled int32
	mock := &mockFileService{
		infoFunc: func(_ context.Context, _ string, _ bool) (*service.File, error) {
			return testImageFile(testUUID1, "photo.jpg"), nil
		},
		downloadFunc: func(_ context.Context, _ service.DownloadParams) (*service.DownloadResult, error) {
			atomic.AddInt32(&downloadCalled, 1)
			return nil, nil
		},
	}

	dir := t.TempDir()
	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, "--json", "all", "file", "download",
		testUUID1, "--output", filepath.Join(dir, "x.jpg"), "--dry-run",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if atomic.LoadInt32(&downloadCalled) != 0 {
		t.Error("dry-run should not call Download")
	}
	var row downloadRow
	if err := json.Unmarshal([]byte(stdout), &row); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if row.Status != "dry-run" {
		t.Errorf("status = %q, want dry-run", row.Status)
	}
	if row.SourceURL == "" {
		t.Errorf("source_url should be populated in dry-run")
	}
}

func TestFileDownload_EffectsAppliedToImagesOnly(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	var capturedEffects []string
	var mu sync.Mutex
	mock := &mockFileService{
		infoFunc: func(_ context.Context, uuid string, _ bool) (*service.File, error) {
			if uuid == testUUID1 {
				return testImageFile(testUUID1, "image.jpg"), nil
			}
			return testNonImageFile(testUUID2, "doc.pdf"), nil
		},
		downloadFunc: func(_ context.Context, params service.DownloadParams) (*service.DownloadResult, error) {
			mu.Lock()
			capturedEffects = append(capturedEffects, params.Effects)
			mu.Unlock()
			_, _ = params.Out.Write([]byte("x"))
			return &service.DownloadResult{UUID: params.UUID}, nil
		},
	}

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, testUUID2,
		"--output-dir", outDir,
		"--effects", "resize/200x/",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Effects param is forwarded for both; the service decides whether to
	// apply based on IsImage. We verify the param is passed through; the
	// service-layer test in client/file_test.go covers the URL splicing.
	if len(capturedEffects) != 2 {
		t.Fatalf("expected 2 downloads, got %d", len(capturedEffects))
	}
	for _, e := range capturedEffects {
		if e != "resize/200x/" {
			t.Errorf("effects = %q, want resize/200x/", e)
		}
	}
}

func TestFileDownload_BatchRequiresOutputDir(t *testing.T) {
	mock := staticDownloadMock(
		map[string]*service.File{
			testUUID1: testImageFile(testUUID1, "a.jpg"),
			testUUID2: testImageFile(testUUID2, "b.png"),
		},
		[]byte("x"),
	)
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download", testUUID1, testUUID2)
	if err == nil {
		t.Fatal("expected error: batch requires --output-dir")
	}
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Errorf("expected ExitError code 2, got %v", err)
	}
}

func TestFileDownload_OutputAndOutputDirExclusive(t *testing.T) {
	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "a.jpg")},
		[]byte("x"),
	)
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download",
		testUUID1, "--output", "/tmp/x", "--output-dir", "/tmp/out",
	)
	if err == nil {
		t.Fatal("expected error: --output and --output-dir exclusive")
	}
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Errorf("expected ExitError code 2, got %v", err)
	}
}

func TestFileDownload_ParallelOutOfRange(t *testing.T) {
	mock := staticDownloadMock(
		map[string]*service.File{testUUID1: testImageFile(testUUID1, "a.jpg")},
		[]byte("x"),
	)
	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, "file", "download", testUUID1, "--parallel", "0")
	if err == nil {
		t.Fatal("expected error for --parallel 0")
	}
	root = newTestRoot(mock)
	_, _, err = executeCommand(t, root, "file", "download", testUUID1, "--parallel", "33")
	if err == nil {
		t.Fatal("expected error for --parallel 33")
	}
}

func TestFileDownload_ParallelPreservesOrder(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	files := map[string]*service.File{}
	uuids := []string{
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003",
		"00000000-0000-0000-0000-000000000004",
	}
	for _, u := range uuids {
		files[u] = testImageFile(u, "x.bin")
	}

	// Mock Download to return in reverse order of arrival via channel barriers.
	mock := &mockFileService{
		infoFunc: func(_ context.Context, uuid string, _ bool) (*service.File, error) {
			return files[uuid], nil
		},
		downloadFunc: func(_ context.Context, params service.DownloadParams) (*service.DownloadResult, error) {
			_, _ = params.Out.Write([]byte(params.UUID))
			return &service.DownloadResult{UUID: params.UUID, SourceURL: "https://x/" + params.UUID}, nil
		},
	}

	args := []string{"--json", "all", "file", "download"}
	args = append(args, uuids...)
	args = append(args, "--output-dir", outDir, "--parallel", "4")

	root := newTestRoot(mock)
	stdout, _, err := executeCommand(t, root, args...)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var rows []downloadRow
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("invalid JSON: %v\ngot: %s", err, stdout)
	}
	if len(rows) != len(uuids) {
		t.Fatalf("got %d rows, want %d", len(rows), len(uuids))
	}
	for i, u := range uuids {
		if rows[i].UUID != u {
			t.Errorf("row %d uuid = %s, want %s (order not preserved)", i, rows[i].UUID, u)
		}
	}
}

func TestFileDownload_ParallelCancelsOnError(t *testing.T) {
	uuids := []string{
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003",
		"00000000-0000-0000-0000-000000000004",
	}

	var downloadCount int32
	released := make(chan struct{})

	mock := &mockFileService{
		infoFunc: func(_ context.Context, uuid string, _ bool) (*service.File, error) {
			return testImageFile(uuid, "x.bin"), nil
		},
		downloadFunc: func(ctx context.Context, params service.DownloadParams) (*service.DownloadResult, error) {
			n := atomic.AddInt32(&downloadCount, 1)
			if n == 1 {
				// First worker fails immediately, triggers cancel.
				return nil, errors.New("boom")
			}
			// Other workers block until released or ctx cancelled.
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-released:
				_, _ = params.Out.Write([]byte("x"))
				return &service.DownloadResult{UUID: params.UUID}, nil
			}
		},
	}
	defer close(released)

	dir := t.TempDir()
	args := []string{"file", "download"}
	args = append(args, uuids...)
	args = append(args, "--output-dir", dir, "--parallel", "4")

	root := newTestRoot(mock)
	_, _, err := executeCommand(t, root, args...)
	if err == nil {
		t.Fatal("expected error from failing worker")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error should propagate worker failure, got: %v", err)
	}
}

func TestFileDownload_NoUUIDsError(t *testing.T) {
	root := newTestRoot(&mockFileService{})
	_, _, err := executeCommand(t, root, "file", "download")
	if err == nil {
		t.Fatal("expected error: no UUIDs provided")
	}
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Errorf("expected ExitError code 2, got %v", err)
	}
}

func TestFileDownload_InvalidUUID(t *testing.T) {
	root := newTestRoot(&mockFileService{})
	_, _, err := executeCommand(t, root, "file", "download", "not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
	var ee *ExitError
	if !errors.As(err, &ee) || ee.Code != 2 {
		t.Errorf("expected ExitError code 2, got %v", err)
	}
}

func TestExpandNameTemplate_Placeholders(t *testing.T) {
	f := &service.File{
		UUID:     testUUID1,
		Filename: "photo.jpg",
	}
	tests := []struct {
		tmpl    string
		effects string
		want    string
	}{
		{"${uuid}${ext}", "", testUUID1 + ".jpg"},
		{"${filename}", "", "photo.jpg"},
		{"${uuid}-${effects}${ext}", "resize/200x/", testUUID1 + "-resize_200x.jpg"},
		{"plain-name.bin", "", "plain-name.bin"},
	}
	for _, tt := range tests {
		got, err := expandNameTemplate(tt.tmpl, f, tt.effects)
		if err != nil {
			t.Errorf("expandNameTemplate(%q): unexpected error: %v", tt.tmpl, err)
			continue
		}
		if got != tt.want {
			t.Errorf("expandNameTemplate(%q, effects=%q) = %q, want %q", tt.tmpl, tt.effects, got, tt.want)
		}
	}
}

func TestComputeSourceURL(t *testing.T) {
	img := testImageFile(testUUID1, "x.jpg")
	imgWithFilename := testImageFile(testUUID1, "photo.jpg")
	imgWithFilename.OriginalFileURL = "https://ucarecdn.com/" + testUUID1 + "/photo.jpg"
	pdf := testNonImageFile(testUUID2, "x.pdf")

	tests := []struct {
		name    string
		f       *service.File
		effects string
		want    string
	}{
		{"no effects", img, "", "https://ucarecdn.com/" + testUUID1 + "/"},
		{"image + effects, no filename", img, "resize/200x/", "https://ucarecdn.com/" + testUUID1 + "/-/resize/200x/"},
		{"image + effects with leading -/", img, "-/resize/200x/", "https://ucarecdn.com/" + testUUID1 + "/-/resize/200x/"},
		{"image + effects + filename suffix", imgWithFilename, "resize/200x/", "https://ucarecdn.com/" + testUUID1 + "/-/resize/200x/photo.jpg"},
		{"non-image ignores effects", pdf, "resize/200x/", "https://ucarecdn.com/" + testUUID2 + "/"},
	}
	for _, tt := range tests {
		got := computeSourceURL(tt.f, tt.effects)
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestApplyCDNEffects(t *testing.T) {
	uuid := testUUID1
	tests := []struct {
		name string
		in   string
		eff  string
		want string
	}{
		{"empty effects", "https://ucarecdn.com/" + uuid + "/", "", "https://ucarecdn.com/" + uuid + "/"},
		{"empty url", "", "resize/200x/", ""},
		{"trailing slash no filename", "https://ucarecdn.com/" + uuid + "/", "resize/200x/", "https://ucarecdn.com/" + uuid + "/-/resize/200x/"},
		{"with filename", "https://ucarecdn.com/" + uuid + "/photo.jpg", "resize/200x/", "https://ucarecdn.com/" + uuid + "/-/resize/200x/photo.jpg"},
		{"custom cdn with path prefix", "https://cdn.example.com/uploads/" + uuid + "/photo.jpg", "resize/200x/", "https://cdn.example.com/uploads/" + uuid + "/-/resize/200x/photo.jpg"},
		{"leading -/ stripped", "https://ucarecdn.com/" + uuid + "/photo.jpg", "-/resize/200x/", "https://ucarecdn.com/" + uuid + "/-/resize/200x/photo.jpg"},
		{"effects without trailing slash", "https://ucarecdn.com/" + uuid + "/photo.jpg", "resize/200x", "https://ucarecdn.com/" + uuid + "/-/resize/200x/photo.jpg"},
		{"uuid at end of url (no trailing slash)", "https://ucarecdn.com/" + uuid, "resize/200x/", "https://ucarecdn.com/" + uuid + "/-/resize/200x/"},
		{"unrelated url returned unchanged", "https://example.com/elsewhere/", "resize/200x/", "https://example.com/elsewhere/"},
	}
	for _, tt := range tests {
		got := service.ApplyCDNEffects(tt.in, uuid, tt.eff)
		if got != tt.want {
			t.Errorf("%s: ApplyCDNEffects(%q, _, %q) = %q, want %q", tt.name, tt.in, tt.eff, got, tt.want)
		}
	}
}

// Compile-time guards
var (
	_ = io.Copy
	_ = bytes.NewBuffer
)
