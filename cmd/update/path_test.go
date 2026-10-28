package update

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestResolveTargetExecutablePath(t *testing.T) {
	const helperResultEnv = "TNNL_UPDATE_PATH_HELPER_RESULT"
	if resultPath := os.Getenv(helperResultEnv); resultPath != "" {
		resolved, err := resolveTargetExecutablePath()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(resultPath, []byte(resolved), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("executable and symlink fixtures require a supported release platform")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name          string
		executableRel string
		launchSymlink bool
		pathSymlink   bool
		pathMissing   bool
		wantPath      bool
	}{
		{name: "installed binary keeps itself", executableRel: "installed/tnnl"},
		{name: "launched symlink resolves to installed binary", executableRel: "installed/tnnl", launchSymlink: true},
		{name: "other executable name uses PATH", executableRel: "installed/main", wantPath: true},
		{name: "custom go run directory uses PATH", executableRel: "custom-tmp/go-build123/b001/exe/tnnl", wantPath: true},
		{name: "system temporary directory uses PATH", executableRel: "temporary/tnnl", wantPath: true},
		{name: "PATH symlink resolves to target", executableRel: "installed/main", pathSymlink: true, wantPath: true},
		{name: "missing PATH binary falls back to running executable", executableRel: "installed/main", pathMissing: true},
		{name: "similar go builder directory keeps itself", executableRel: "go-builder/tnnl"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Canonicalize the fixture root independently: macOS's TempDir often
			// contains /var while os.Executable returns its /private/var target.
			base, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			self := filepath.Join(base, filepath.FromSlash(tt.executableRel))
			if err := os.MkdirAll(filepath.Dir(self), 0o755); err != nil {
				t.Fatal(err)
			}
			copyResolverExecutable(t, executable, self)

			pathDir := filepath.Join(base, "path-bin")
			tempDir := filepath.Join(base, "temporary")
			for _, dir := range []string{pathDir, tempDir} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			pathBinary := filepath.Join(pathDir, "tnnl")
			pathTarget := pathBinary
			if !tt.pathMissing {
				if tt.pathSymlink {
					pathTarget = filepath.Join(base, "versioned-tnnl")
				}
				// LookPath must only inspect this file, never execute it.
				if err := os.WriteFile(pathTarget, []byte("not an executable program\n"), 0o755); err != nil {
					t.Fatal(err)
				}
				if tt.pathSymlink {
					if err := os.Symlink(pathTarget, pathBinary); err != nil {
						t.Fatal(err)
					}
				}
			}
			launch := self
			if tt.launchSymlink {
				launch = filepath.Join(base, "launch-link")
				if err := os.Symlink(self, launch); err != nil {
					t.Fatal(err)
				}
			}

			resultPath := filepath.Join(base, "resolved-path")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, launch, "-test.run=^TestResolveTargetExecutablePath$")
			command.WaitDelay = time.Second
			command.Env = append(os.Environ(),
				helperResultEnv+"="+resultPath,
				"PATH="+pathDir,
				"TMPDIR="+tempDir, "TMP="+tempDir, "TEMP="+tempDir)
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("resolve from %s: %v\n%s", launch, err, output)
			}
			result, err := os.ReadFile(resultPath)
			if err != nil {
				t.Fatal(err)
			}
			want := self
			if tt.wantPath {
				want = pathTarget
			}
			if got := string(result); got != want {
				t.Fatalf("resolved update target = %q, want %q", got, want)
			}
		})
	}
}

func copyResolverExecutable(t *testing.T, source, target string) {
	t.Helper()
	if err := os.Link(source, target); err == nil {
		return
	}
	// Go's test cache and TempDir need not be on the same filesystem.
	in, err := os.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
