package update

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wim-web/tnnl/cmd"
)

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
		TagName: "v0.6.0",
		Assets: []assetInfo{
			{
				Name:        "tnnl_darwin_arm64.tar.gz",
				DownloadURL: "https://example.com/tnnl_darwin_arm64.tar.gz",
			},
		},
	}

	got, err := rel.assetURL("tnnl_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("assetURL() unexpected error: %v", err)
	}

	if want := "https://example.com/tnnl_darwin_arm64.tar.gz"; got != want {
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

	got := currentVersion(scriptPath)
	if want := "9.9.9"; got != want {
		t.Fatalf("currentVersion() = %q, want %q", got, want)
	}
}

func TestCurrentVersion_FallbackToEmbedded(t *testing.T) {
	before := cmd.Version
	cmd.Version = "v1.2.3"
	t.Cleanup(func() {
		cmd.Version = before
	})

	got := currentVersion("/path/not/found/tnnl")
	if want := "1.2.3"; got != want {
		t.Fatalf("currentVersion() = %q, want %q", got, want)
	}
}
