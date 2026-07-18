package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	testReleaseTag = "v1.2.3"
	testAssetName  = "tnnl_testos_testarch.tar.gz"
)

func TestUpdaterInstallsOnlyAfterVerification(t *testing.T) {
	fixture := newUpdaterFixture(t, "1.0.0", "1.2.3")
	var output bytes.Buffer

	if err := fixture.updater().run(context.Background(), &output); err != nil {
		t.Fatalf("updater.run() error = %v", err)
	}

	if got, want := output.String(), "updated: v1.0.0 -> v1.2.3\n"; got != want {
		t.Fatalf("updater output = %q, want %q", got, want)
	}
	contents, err := os.ReadFile(fixture.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), string(fixture.candidate); got != want {
		t.Fatalf("installed contents = %q, want candidate %q", got, want)
	}
	info, err := os.Stat(fixture.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o755); got != want {
		t.Fatalf("installed mode = %o, want %o", got, want)
	}
	if got, want := fixture.requestPaths(), []string{
		"/releases/latest",
		"/releases/download/v1.2.3/" + testAssetName,
		"/releases/download/v1.2.3/checksums.txt",
	}; !equalStrings(got, want) {
		t.Fatalf("request paths = %q, want %q", got, want)
	}
}

func TestUpdaterLeavesExecutableUntouchedOnChecksumMismatch(t *testing.T) {
	fixture := newUpdaterFixture(t, "1.0.0", "1.2.3")
	fixture.manifest = []byte(strings.Repeat("0", sha256.Size*2) + "  " + testAssetName + "\n")

	err := fixture.updater().run(context.Background(), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("updater.run() error = %v, want checksum mismatch", err)
	}
	fixture.assertCurrentUnchanged(t)
}

func TestUpdaterDoesNotExtractBeforeChecksumVerification(t *testing.T) {
	fixture := newUpdaterFixture(t, "1.0.0", "1.2.3")
	fixture.archive = []byte("not a gzip archive")
	fixture.manifest = []byte(strings.Repeat("f", sha256.Size*2) + "  " + testAssetName + "\n")

	err := fixture.updater().run(context.Background(), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("updater.run() error = %v, want checksum mismatch before extraction", err)
	}
	if strings.Contains(strings.ToLower(err.Error()), "gzip") {
		t.Fatalf("updater.run() error = %v, extraction ran before checksum verification", err)
	}
	fixture.assertCurrentUnchanged(t)
}

func TestUpdaterLeavesExecutableUntouchedOnCandidateVersionMismatch(t *testing.T) {
	fixture := newUpdaterFixture(t, "1.0.0", "9.9.9")

	err := fixture.updater().run(context.Background(), io.Discard)
	if err == nil || !strings.Contains(err.Error(), "candidate version 9.9.9 does not match release 1.2.3") {
		t.Fatalf("updater.run() error = %v, want candidate version mismatch", err)
	}
	fixture.assertCurrentUnchanged(t)
}

func TestUpdaterPropagatesCallerCancellation(t *testing.T) {
	fixture := newUpdaterFixture(t, "1.0.0", "1.2.3")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := fixture.updater().run(ctx, io.Discard)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("updater.run() error = %v, want context.Canceled", err)
	}
	fixture.assertCurrentUnchanged(t)
}

func TestUpdaterCancellationStopsCurrentVersionProbe(t *testing.T) {
	fixture := newUpdaterFixture(t, "1.0.0", "1.2.3")
	startedPath := filepath.Join(t.TempDir(), "started")
	blocking := []byte("#!/bin/sh\nprintf started > \"$UPDATE_CURRENT_VERSION_STARTED\"\nexec sleep 30\n")
	if err := os.WriteFile(fixture.currentPath, blocking, 0o755); err != nil {
		t.Fatal(err)
	}
	fixture.snapshotCurrent(t)
	t.Setenv("UPDATE_CURRENT_VERSION_STARTED", startedPath)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errCh := make(chan error, 1)
	go func() {
		errCh <- fixture.updater().run(ctx, io.Discard)
	}()
	if err := waitForCandidateHelper(startedPath, candidateHelperSyncLimit); err != nil {
		cancel()
		t.Fatalf("wait for current version probe: %v", err)
	}
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("updater.run() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("updater.run() did not return after cancellation")
	}
	if got := fixture.assetRequestCount(); got != 0 {
		t.Fatalf("asset request count = %d, want 0", got)
	}
	fixture.assertCurrentUnchanged(t)
}

func TestUpdaterAlreadyLatestMakesNoAssetRequests(t *testing.T) {
	fixture := newUpdaterFixture(t, "1.2.3", "1.2.3")
	var output bytes.Buffer

	if err := fixture.updater().run(context.Background(), &output); err != nil {
		t.Fatalf("updater.run() error = %v", err)
	}
	if got, want := output.String(), "already latest version: v1.2.3\n"; got != want {
		t.Fatalf("updater output = %q, want %q", got, want)
	}
	if got := fixture.assetRequestCount(); got != 0 {
		t.Fatalf("asset request count = %d, want 0", got)
	}
	fixture.assertCurrentUnchanged(t)
}

func TestUpdaterReturnsOutputError(t *testing.T) {
	fixture := newUpdaterFixture(t, "1.2.3", "1.2.3")
	wantErr := errors.New("write failed")

	err := fixture.updater().run(context.Background(), updateFailWriter{err: wantErr})
	if !errors.Is(err, wantErr) {
		t.Fatalf("updater.run() error = %v, want %v", err, wantErr)
	}
}

func TestUpdaterAssetRedirectsUseOriginalClientPolicy(t *testing.T) {
	fixture := newUpdaterFixture(t, "1.0.0", "1.2.3")
	fixture.redirectAssets = true
	redirects := 0
	client := fixture.server.Client()
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		redirects++
		return nil
	}
	updater := fixture.updater()
	updater.client = client

	if err := updater.run(context.Background(), io.Discard); err != nil {
		t.Fatalf("updater.run() error = %v", err)
	}
	if got, want := redirects, 2; got != want {
		t.Fatalf("asset redirect policy calls = %d, want %d", got, want)
	}
}

func TestUpdaterWrapsExecutableResolutionError(t *testing.T) {
	wantErr := errors.New("executable unavailable")
	updater := updater{executablePath: func() (string, error) { return "", wantErr }}

	err := updater.run(context.Background(), io.Discard)
	if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), "resolve executable") {
		t.Fatalf("updater.run() error = %v, want wrapped resolution error", err)
	}
}

func TestNewUpdateCommandUsesContextAndOutput(t *testing.T) {
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "value")
	var output bytes.Buffer
	command := newUpdateCommand(func(gotCtx context.Context, out io.Writer) error {
		if got, want := gotCtx.Value(contextKey{}), "value"; got != want {
			t.Fatalf("command context value = %v, want %v", got, want)
		}
		_, err := io.WriteString(out, "updated\n")
		return err
	})
	command.SetOut(&output)

	if err := command.ExecuteContext(ctx); err != nil {
		t.Fatalf("update command error = %v", err)
	}
	if got, want := output.String(), "updated\n"; got != want {
		t.Fatalf("update command output = %q, want %q", got, want)
	}
}

func TestNewUpdateCommandReturnsRunnerError(t *testing.T) {
	wantErr := errors.New("update failed")
	command := newUpdateCommand(func(context.Context, io.Writer) error { return wantErr })

	if err := command.ExecuteContext(context.Background()); !errors.Is(err, wantErr) {
		t.Fatalf("update command error = %v, want %v", err, wantErr)
	}
}

type updaterFixture struct {
	currentPath      string
	candidate        []byte
	archive          []byte
	manifest         []byte
	originalContents []byte
	originalMode     os.FileMode
	server           *httptest.Server
	redirectAssets   bool

	mu            sync.Mutex
	requests      []string
	assetRequests int
}

func newUpdaterFixture(t *testing.T, currentVersion, candidateVersion string) *updaterFixture {
	t.Helper()
	fixture := &updaterFixture{
		currentPath: filepath.Join(t.TempDir(), binaryName),
		candidate:   versionScript(candidateVersion),
	}
	if err := os.WriteFile(fixture.currentPath, versionScript(currentVersion), 0o755); err != nil {
		t.Fatal(err)
	}
	fixture.snapshotCurrent(t)
	fixture.archive = archiveWithBinary(t, fixture.candidate)
	fixture.manifest = checksumManifest(testAssetName, fixture.archive)
	fixture.server = httptest.NewServer(http.HandlerFunc(fixture.serveHTTP))
	t.Cleanup(fixture.server.Close)
	return fixture
}

func (f *updaterFixture) updater() updater {
	return updater{
		client:         f.server.Client(),
		latestURL:      f.server.URL + "/releases/latest",
		goos:           "testos",
		goarch:         "testarch",
		executablePath: func() (string, error) { return f.currentPath, nil },
	}
}

func (f *updaterFixture) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if got := r.Header.Get("Authorization"); got != "" {
		http.Error(w, "authorization must be empty", http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.requests = append(f.requests, r.URL.Path)
	f.mu.Unlock()

	archivePath := "/releases/download/v1.2.3/" + testAssetName
	manifestPath := "/releases/download/v1.2.3/checksums.txt"
	switch r.URL.Path {
	case "/releases/latest":
		w.Header().Set("Location", f.server.URL+"/releases/tag/"+testReleaseTag)
		w.WriteHeader(http.StatusFound)
	case archivePath:
		f.incrementAssetRequests()
		if f.redirectAssets {
			http.Redirect(w, r, "/objects/archive", http.StatusFound)
			return
		}
		_, _ = w.Write(f.archive)
	case manifestPath:
		f.incrementAssetRequests()
		if f.redirectAssets {
			http.Redirect(w, r, "/objects/checksums", http.StatusFound)
			return
		}
		_, _ = w.Write(f.manifest)
	case "/objects/archive":
		_, _ = w.Write(f.archive)
	case "/objects/checksums":
		_, _ = w.Write(f.manifest)
	default:
		http.NotFound(w, r)
	}
}

func (f *updaterFixture) incrementAssetRequests() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.assetRequests++
}

func (f *updaterFixture) assetRequestCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.assetRequests
}

func (f *updaterFixture) requestPaths() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.requests...)
}

func (f *updaterFixture) snapshotCurrent(t *testing.T) {
	t.Helper()
	contents, err := os.ReadFile(f.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(f.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	f.originalContents = append([]byte(nil), contents...)
	f.originalMode = info.Mode().Perm()
}

func (f *updaterFixture) assertCurrentUnchanged(t *testing.T) {
	t.Helper()
	contents, err := os.ReadFile(f.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents, f.originalContents) {
		t.Fatalf("current executable contents changed to %q, want %q", contents, f.originalContents)
	}
	info, err := os.Stat(f.currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != f.originalMode {
		t.Fatalf("current executable mode changed to %o, want %o", got, f.originalMode)
	}
}

func versionScript(version string) []byte {
	return []byte("#!/bin/sh\nif [ \"$1\" = version ]; then\n  printf '%s\\n' '" + version + "'\n  exit 0\nfi\nexit 64\n")
}

func archiveWithBinary(t *testing.T, binary []byte) []byte {
	t.Helper()
	var archive bytes.Buffer
	gzipWriter := gzip.NewWriter(&archive)
	tarWriter := tar.NewWriter(gzipWriter)
	header := &tar.Header{Name: binaryName, Mode: 0o755, Size: int64(len(binary))}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(binary); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}

func checksumManifest(assetName string, archive []byte) []byte {
	digest := sha256.Sum256(archive)
	return []byte(fmt.Sprintf("%x  %s\n", digest, assetName))
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

type updateFailWriter struct {
	err error
}

func (w updateFailWriter) Write([]byte) (int, error) {
	return 0, w.err
}
