package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/cmd"
	"github.com/wim-web/tnnl/internal/buildinfo"
)

const (
	latestReleaseURL = "https://github.com/wim-web/tnnl/releases/latest"
	binaryName       = "tnnl"
)

var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update tnnl to the latest release",
	Run: func(cmd *cobra.Command, args []string) {
		if err := updateCLI(); err != nil {
			log.Fatalln(err)
		}
	},
}

type release struct {
	TagName         string
	DownloadBaseURL string
}

func init() {
	cmd.RootCmd.AddCommand(UpdateCmd)
}

func updateCLI() error {
	exePath, err := resolveTargetExecutablePath()
	if err != nil {
		return err
	}

	latest, err := fetchLatestRelease(context.Background(), http.DefaultClient, latestReleaseURL)
	if err != nil {
		return err
	}

	current, err := currentVersion(context.Background(), exePath)
	if err != nil {
		return err
	}
	latestVersion := normalizeVersion(latest.TagName)
	if current != "" && current == latestVersion {
		fmt.Printf("already latest version: %s\n", latest.TagName)
		return nil
	}

	assetName := fmt.Sprintf("%s_%s_%s.tar.gz", binaryName, runtime.GOOS, runtime.GOARCH)
	assetURL, err := latest.assetURL(assetName)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "tnnl-update-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, assetName)
	if err := downloadFile(context.Background(), http.DefaultClient, assetURL, archivePath); err != nil {
		return err
	}

	extractedPath := filepath.Join(tmpDir, binaryName)
	if err := extractBinaryFromArchive(archivePath, extractedPath); err != nil {
		return err
	}

	if err := replaceExecutable(exePath, extractedPath); err != nil {
		return err
	}

	if current == "" {
		fmt.Printf("updated to %s\n", latest.TagName)
		return nil
	}

	fmt.Printf("updated: v%s -> %s\n", current, latest.TagName)
	return nil
}

func fetchLatestRelease(ctx context.Context, client *http.Client, latestURL string) (release, error) {
	if client == nil {
		return release{}, fmt.Errorf("fetch latest release: HTTP client is nil")
	}
	if _, err := parsePublicHTTPURL(latestURL, "latest release URL"); err != nil {
		return release{}, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, latestURL, nil)
	if err != nil {
		return release{}, fmt.Errorf("create latest release request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", binaryName, buildinfo.Current()))

	redirectClient := *client
	redirectClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	res, err := redirectClient.Do(req)
	if err != nil {
		requestErr := fmt.Errorf("fetch latest release: %w", err)
		if contextErr := reqCtx.Err(); contextErr != nil {
			return release{}, errors.Join(requestErr, contextErr)
		}
		return release{}, requestErr
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusMultipleChoices || res.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4*1024))
		return release{}, fmt.Errorf("failed to fetch latest release redirect: status=%d body=%s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	latest, err := releaseFromLatestLocation(res.Header.Get("Location"))
	if err != nil {
		return release{}, fmt.Errorf("parse latest release redirect: %w", err)
	}

	return latest, nil
}

func releaseFromLatestLocation(location string) (release, error) {
	if location == "" {
		return release{}, fmt.Errorf("latest release redirect location is empty")
	}

	redirectURL, err := parsePublicHTTPURL(location, "latest release redirect location")
	if err != nil {
		return release{}, err
	}

	const marker = "/releases/tag/"
	escapedPath := redirectURL.EscapedPath()
	idx := strings.Index(escapedPath, marker)
	if idx < 0 {
		return release{}, fmt.Errorf("latest release redirect location does not contain release tag: %s", location)
	}

	escapedTagName := escapedPath[idx+len(marker):]
	if escapedTagName == "" {
		return release{}, fmt.Errorf("latest release tag is empty")
	}
	if strings.Contains(escapedTagName, "/") {
		return release{}, fmt.Errorf("latest release tag must be one escaped path segment: %s", location)
	}
	tagName, err := url.PathUnescape(escapedTagName)
	if err != nil {
		return release{}, fmt.Errorf("unescape latest release tag: %w", err)
	}
	if tagName == "" {
		return release{}, fmt.Errorf("latest release tag is empty")
	}

	escapedDownloadPath := escapedPath[:idx] + "/releases/download/" + url.PathEscape(tagName)
	downloadPath, err := url.PathUnescape(escapedDownloadPath)
	if err != nil {
		return release{}, fmt.Errorf("unescape release download path: %w", err)
	}
	downloadURL := &url.URL{
		Scheme:  strings.ToLower(redirectURL.Scheme),
		Host:    redirectURL.Host,
		Path:    downloadPath,
		RawPath: escapedDownloadPath,
	}

	return release{TagName: tagName, DownloadBaseURL: downloadURL.String()}, nil
}

func parsePublicHTTPURL(rawURL, description string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", description, err)
	}
	if !parsed.IsAbs() || (!strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https")) {
		return nil, fmt.Errorf("%s must be an absolute HTTP(S) URL", description)
	}
	if parsed.Host == "" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("%s must include a host", description)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("%s must not contain userinfo", description)
	}
	return parsed, nil
}

func (r release) assetURL(assetName string) (string, error) {
	if r.DownloadBaseURL != "" {
		return strings.TrimRight(r.DownloadBaseURL, "/") + "/" + url.PathEscape(assetName), nil
	}

	return "", fmt.Errorf("release asset not found: %s", assetName)
}

func downloadFile(ctx context.Context, client *http.Client, assetURL, destPath string) error {
	if client == nil {
		return fmt.Errorf("download release asset: HTTP client is nil")
	}
	if _, err := parsePublicHTTPURL(assetURL, "release asset URL"); err != nil {
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, assetURL, nil)
	if err != nil {
		return fmt.Errorf("create release asset request: %w", err)
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", binaryName, buildinfo.Current()))

	res, err := client.Do(req)
	if err != nil {
		downloadErr := fmt.Errorf("download release asset: %w", err)
		if contextErr := reqCtx.Err(); contextErr != nil {
			return errors.Join(downloadErr, contextErr)
		}
		return downloadErr
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4*1024))
		return fmt.Errorf("failed to download release asset: status=%d body=%s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("create downloaded release asset %q: %w", destPath, err)
	}

	if _, err := io.Copy(f, res.Body); err != nil {
		_ = f.Close()
		return fmt.Errorf("write downloaded release asset %q: %w", destPath, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close downloaded release asset %q: %w", destPath, err)
	}

	return nil
}

func extractBinaryFromArchive(archivePath string, destPath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != binaryName {
			continue
		}

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}

		_, copyErr := io.Copy(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}

		return nil
	}

	return fmt.Errorf("binary not found in archive: %s", binaryName)
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func currentVersion(ctx context.Context, exePath string) (string, error) {
	if version, err := readBinaryVersion(ctx, exePath); err == nil {
		return version, nil
	} else if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "", err
	}

	return normalizeVersion(buildinfo.Current()), nil
}

func readBinaryVersion(ctx context.Context, exePath string) (string, error) {
	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	probe := exec.CommandContext(probeCtx, exePath, "version")
	probe.WaitDelay = candidateVersionWaitDelay
	output, err := probe.Output()
	if err != nil {
		probeErr := fmt.Errorf("run installed binary version: %w", err)
		if contextErr := probeCtx.Err(); contextErr != nil {
			return "", errors.Join(probeErr, contextErr)
		}
		return "", probeErr
	}

	version := normalizeVersion(string(output))
	if version == "" {
		return "", fmt.Errorf("version output is empty")
	}

	return version, nil
}

func resolveTargetExecutablePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	exePath = resolveSymlinkPath(exePath)

	// `go run` 実行時は一時バイナリになるため、PATH 上の tnnl を更新対象にする。
	if shouldUsePathBinary(exePath) {
		lookedUpPath, err := exec.LookPath(binaryName)
		if err == nil {
			return resolveSymlinkPath(lookedUpPath), nil
		}
	}

	return exePath, nil
}

func resolveSymlinkPath(path string) string {
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return path
	}
	return resolvedPath
}

func shouldUsePathBinary(exePath string) bool {
	if filepath.Base(exePath) != binaryName {
		return true
	}

	tempDir := filepath.Clean(os.TempDir()) + string(filepath.Separator)
	cleanPath := filepath.Clean(exePath)
	if strings.HasPrefix(cleanPath, tempDir) {
		return true
	}

	sep := string(filepath.Separator)
	if strings.Contains(cleanPath, sep+"go-build") || strings.Contains(cleanPath, sep+"go-build"+sep) {
		return true
	}

	return false
}
