package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/cmd"
)

const (
	releaseAPIURL = "https://api.github.com/repos/wim-web/tnnl/releases/latest"
	binaryName    = "tnnl"
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
	TagName string      `json:"tag_name"`
	Assets  []assetInfo `json:"assets"`
}

type assetInfo struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

func init() {
	cmd.RootCmd.AddCommand(UpdateCmd)
}

func updateCLI() error {
	exePath, err := resolveTargetExecutablePath()
	if err != nil {
		return err
	}

	latest, err := fetchLatestRelease()
	if err != nil {
		return err
	}

	current := currentVersion(exePath)
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
	if err := downloadFile(assetURL, archivePath); err != nil {
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

func fetchLatestRelease() (release, error) {
	reqCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, releaseAPIURL, nil)
	if err != nil {
		return release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", binaryName, cmd.Version))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return release{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4*1024))
		return release{}, fmt.Errorf("failed to fetch latest release: status=%d body=%s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var rel release
	if err := json.NewDecoder(res.Body).Decode(&rel); err != nil {
		return release{}, err
	}
	if rel.TagName == "" {
		return release{}, fmt.Errorf("latest release tag is empty")
	}

	return rel, nil
}

func (r release) assetURL(assetName string) (string, error) {
	for _, a := range r.Assets {
		if a.Name == assetName {
			return a.DownloadURL, nil
		}
	}

	return "", fmt.Errorf("release asset not found: %s", assetName)
}

func downloadFile(url string, destPath string) error {
	reqCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", binaryName, cmd.Version))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4*1024))
		return fmt.Errorf("failed to download release asset: status=%d body=%s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, res.Body); err != nil {
		return err
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

func replaceExecutable(currentPath string, newBinaryPath string) error {
	tmpPath := filepath.Join(filepath.Dir(currentPath), fmt.Sprintf(".%s.new", filepath.Base(currentPath)))

	src, err := os.Open(newBinaryPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}

	if err := os.Rename(tmpPath, currentPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to replace executable (%s). check write permission: %w", currentPath, err)
	}

	return nil
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func currentVersion(exePath string) string {
	if v, err := readBinaryVersion(exePath); err == nil && v != "" {
		return v
	}

	return normalizeVersion(cmd.Version)
}

func readBinaryVersion(exePath string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, exePath, "version").Output()
	if err != nil {
		return "", err
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
