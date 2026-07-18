package update

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/internal/buildinfo"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestUpdateHelpDocumentsVerificationAndReplacement(t *testing.T) {
	command := newUpdateCommand(func(context.Context, io.Writer) error { return nil })

	assertHelpContains(t, command,
		"SHA-256 checksum",
		"candidate version",
		"same-directory atomic replacement",
		"write permission",
	)
}

func assertHelpContains(t *testing.T, command *cobra.Command, values ...string) {
	t.Helper()

	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	if err := command.Help(); err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		if !strings.Contains(output.String(), value) {
			t.Errorf("help does not contain %q:\n%s", value, output.String())
		}
	}
}

func TestFetchLatestReleaseUsesInjectedClientWithoutAuthorization(t *testing.T) {
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if got, want := req.URL.String(), "https://example.test/proxy/releases/latest"; got != want {
				t.Fatalf("request URL = %q, want %q", got, want)
			}
			if got := req.Header.Get("Authorization"); got != "" {
				t.Fatalf("Authorization header = %q, want empty", got)
			}

			return &http.Response{
				StatusCode: http.StatusFound,
				Body:       http.NoBody,
				Header: http.Header{
					"Location": []string{"https://downloads.example.test/proxy/releases/tag/v1.2.3?ignored=1#ignored"},
				},
			}, nil
		}),
	}

	got, err := fetchLatestRelease(context.Background(), client, "https://example.test/proxy/releases/latest")
	if err != nil {
		t.Fatalf("fetchLatestRelease() error = %v", err)
	}
	if want := "v1.2.3"; got.TagName != want {
		t.Fatalf("fetchLatestRelease().TagName = %q, want %q", got.TagName, want)
	}
	if got, want := got.DownloadBaseURL, "https://downloads.example.test/proxy/releases/download/v1.2.3"; got != want {
		t.Fatalf("fetchLatestRelease().DownloadBaseURL = %q, want %q", got, want)
	}
}

func TestFetchLatestReleaseDoesNotMutateInjectedClient(t *testing.T) {
	sentinel := errors.New("original redirect policy")
	policyCalled := false
	client := &http.Client{
		Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusFound,
				Body:       http.NoBody,
				Header:     http.Header{"Location": []string{"https://example.test/releases/tag/v1.2.3"}},
			}, nil
		}),
		CheckRedirect: func(*http.Request, []*http.Request) error {
			policyCalled = true
			return sentinel
		},
	}

	if _, err := fetchLatestRelease(context.Background(), client, "https://example.test/releases/latest"); err != nil {
		t.Fatalf("fetchLatestRelease() error = %v", err)
	}
	if policyCalled {
		t.Fatal("injected redirect policy was called for the latest-release probe")
	}
	if err := client.CheckRedirect(&http.Request{}, nil); !errors.Is(err, sentinel) {
		t.Fatalf("injected client CheckRedirect error = %v, want sentinel", err)
	}
	if !policyCalled {
		t.Fatal("injected client redirect policy was replaced")
	}
}

func TestFetchLatestReleaseRejectsUserinfoBeforeRequest(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("transport called for URL containing userinfo")
		return nil, nil
	})}

	_, err := fetchLatestRelease(context.Background(), client, "https://user:password@example.test/releases/latest")
	if err == nil || !strings.Contains(err.Error(), "userinfo") {
		t.Fatalf("fetchLatestRelease() error = %v, want userinfo rejection", err)
	}
}

func TestFetchLatestReleasePreservesCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fetchLatestRelease(ctx, &http.Client{}, "https://example.test/releases/latest")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("fetchLatestRelease() error = %v, want context.Canceled", err)
	}
}

func TestReleaseFromLatestLocation(t *testing.T) {
	got, err := releaseFromLatestLocation("https://example.test/proxy%2Ftenant/releases/tag/v1%2Frc%252F1?ignored=1#ignored")
	if err != nil {
		t.Fatalf("releaseFromLatestLocation() error = %v", err)
	}
	if got, want := got.TagName, "v1/rc%2F1"; got != want {
		t.Fatalf("TagName = %q, want %q", got, want)
	}
	if got, want := got.DownloadBaseURL, "https://example.test/proxy%2Ftenant/releases/download/v1%2Frc%252F1"; got != want {
		t.Fatalf("DownloadBaseURL = %q, want %q", got, want)
	}

	got, err = releaseFromLatestLocation("HTTPS://example.test/releases/tag/v1.2.3")
	if err != nil {
		t.Fatalf("releaseFromLatestLocation() uppercase scheme error = %v", err)
	}
	if got, want := got.DownloadBaseURL, "https://example.test/releases/download/v1.2.3"; got != want {
		t.Fatalf("uppercase scheme DownloadBaseURL = %q, want %q", got, want)
	}
}

func TestReleaseFromLatestLocationRejectsUnsafeOrAmbiguousURL(t *testing.T) {
	for _, location := range []string{
		"",
		"/releases/tag/v1.2.3",
		"//example.test/releases/tag/v1.2.3",
		"ftp://example.test/releases/tag/v1.2.3",
		"https:/releases/tag/v1.2.3",
		"https://user:password@example.test/releases/tag/v1.2.3",
		"https://example.test/releases/latest",
		"https://example.test/releases/tag/",
		"https://example.test/releases/tag/v1.2.3/extra",
		"https://example.test/%2Freleases%2Ftag%2Fv1.2.3",
		"https://example.test/releases/tag/%zz",
	} {
		t.Run(location, func(t *testing.T) {
			if _, err := releaseFromLatestLocation(location); err == nil {
				t.Fatalf("releaseFromLatestLocation(%q) error = nil", location)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "v prefix",
			in:   "v1.2.3",
			want: "1.2.3",
		},
		{
			name: "no prefix",
			in:   "1.2.3",
			want: "1.2.3",
		},
		{
			name: "trim spaces",
			in:   " v0.5.0 ",
			want: "0.5.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeVersion(tt.in)
			if got != tt.want {
				t.Fatalf("normalizeVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAssetURL(t *testing.T) {
	rel := release{
		TagName:         "v0.6.0",
		DownloadBaseURL: "https://github.com/wim-web/tnnl/releases/download/v0.6.0",
	}

	got, err := rel.assetURL("tnnl_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("assetURL() unexpected error: %v", err)
	}

	if want := "https://github.com/wim-web/tnnl/releases/download/v0.6.0/tnnl_darwin_arm64.tar.gz"; got != want {
		t.Fatalf("assetURL() = %q, want %q", got, want)
	}
}

func TestShouldUsePathBinary(t *testing.T) {
	tempPath := filepath.Join(os.TempDir(), "go-build123", "b001", "exe", "tnnl")
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{
			name: "normal installed path",
			in:   filepath.Join(string(filepath.Separator), "usr", "local", "bin", "tnnl"),
			want: false,
		},
		{
			name: "not tnnl binary name",
			in:   filepath.Join(string(filepath.Separator), "usr", "local", "bin", "main"),
			want: true,
		},
		{
			name: "temp go run path",
			in:   tempPath,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldUsePathBinary(tt.in)
			if got != tt.want {
				t.Fatalf("shouldUsePathBinary() = %v, want %v (in=%q)", got, tt.want, tt.in)
			}
		})
	}
}

func TestCurrentVersion_FromBinary(t *testing.T) {
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "tnnl")
	content := "#!/bin/sh\nif [ \"$1\" = \"version\" ]; then\n  echo \"v9.9.9\"\n  exit 0\nfi\nexit 1\n"
	if err := os.WriteFile(scriptPath, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	got, err := currentVersion(context.Background(), scriptPath)
	if err != nil {
		t.Fatalf("currentVersion() error = %v", err)
	}
	if want := "9.9.9"; got != want {
		t.Fatalf("currentVersion() = %q, want %q", got, want)
	}
}

func TestCurrentVersion_FallbackToEmbedded(t *testing.T) {
	got, err := currentVersion(context.Background(), "/path/not/found/tnnl")
	if err != nil {
		t.Fatalf("currentVersion() error = %v", err)
	}
	if want := normalizeVersion(buildinfo.Current()); got != want {
		t.Fatalf("currentVersion() = %q, want %q", got, want)
	}
}

func TestCurrentVersionPreservesCallerCancellation(t *testing.T) {
	scriptPath := filepath.Join(t.TempDir(), "tnnl")
	startedPath := filepath.Join(t.TempDir(), "started")
	script := "#!/bin/sh\nprintf started > \"$UPDATE_CURRENT_VERSION_STARTED\"\nexec sleep 30\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("UPDATE_CURRENT_VERSION_STARTED", startedPath)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	errCh := make(chan error, 1)
	go func() {
		_, err := currentVersion(ctx, scriptPath)
		errCh <- err
	}()
	if err := waitForCandidateHelper(startedPath, candidateHelperSyncLimit); err != nil {
		cancel()
		t.Fatalf("wait for version probe: %v", err)
	}
	cancel()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("currentVersion() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("currentVersion() did not return promptly after cancellation")
	}
}

func TestReadBinaryVersionBoundsInheritedStdoutPipeCleanup(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	startedPath := filepath.Join(t.TempDir(), "started")
	t.Setenv(candidateHelperModeEnv, candidateHelperModeParent)
	t.Setenv(candidateHelperStartedEnv, startedPath)
	ctx := newManualDeadlineContext()
	t.Cleanup(ctx.expire)

	errCh := make(chan error, 1)
	go func() {
		_, err := readBinaryVersion(ctx, executable)
		errCh <- err
	}()
	if err := waitForCandidateHelper(startedPath, candidateHelperSyncLimit); err != nil {
		ctx.expire()
		t.Fatalf("wait for version helper: %v", err)
	}
	started := time.Now()
	ctx.expire()

	select {
	case err := <-errCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("readBinaryVersion() error = %v, want context.DeadlineExceeded", err)
		}
		if elapsed := time.Since(started); elapsed >= 500*time.Millisecond {
			t.Fatalf("readBinaryVersion() elapsed = %v, want < 500ms", elapsed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("readBinaryVersion() did not return promptly")
	}
}

func TestDownloadFileUsesInjectedClientWithoutAuthorization(t *testing.T) {
	destPath := filepath.Join(t.TempDir(), "asset")
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got := req.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization header = %q, want empty", got)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("asset contents")),
			Header:     make(http.Header),
		}, nil
	})}

	if err := downloadFile(context.Background(), client, "https://example.test/asset", destPath); err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}
	contents, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), "asset contents"; got != want {
		t.Fatalf("downloaded contents = %q, want %q", got, want)
	}
}

func TestDownloadFilePreservesCallerCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	destPath := filepath.Join(t.TempDir(), "asset")

	err := downloadFile(ctx, &http.Client{}, "https://example.test/asset", destPath)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("downloadFile() error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(destPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("os.Stat(%q) error = %v, want os.ErrNotExist", destPath, statErr)
	}
}
