package update

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestVerifyCandidateVersion(t *testing.T) {
	t.Run("matches unprefixed candidate to prefixed release", func(t *testing.T) {
		candidatePath := writeCandidateFixture(t, "printf '1.2.3\\n'")

		if err := verifyCandidateVersion(context.Background(), candidatePath, "v1.2.3"); err != nil {
			t.Fatalf("verifyCandidateVersion() error = %v", err)
		}
	})

	t.Run("matches prefixed candidate to unprefixed release", func(t *testing.T) {
		candidatePath := writeCandidateFixture(t, "printf 'v1.2.3\\n'")

		if err := verifyCandidateVersion(context.Background(), candidatePath, "1.2.3"); err != nil {
			t.Fatalf("verifyCandidateVersion() error = %v", err)
		}
	})

	t.Run("normalizes surrounding whitespace", func(t *testing.T) {
		candidatePath := writeCandidateFixture(t, "printf '\\n\\t v1.2.3 \\r\\n'")

		if err := verifyCandidateVersion(context.Background(), candidatePath, "  v1.2.3\n"); err != nil {
			t.Fatalf("verifyCandidateVersion() error = %v", err)
		}
	})

	t.Run("rejects mismatched version with canonical values", func(t *testing.T) {
		candidatePath := writeCandidateFixture(t, "printf 'v9.9.9\\n'")

		err := verifyCandidateVersion(context.Background(), candidatePath, "v1.2.3")
		if err == nil {
			t.Fatal("verifyCandidateVersion() error = nil, want mismatch error")
		}
		if got, want := err.Error(), "candidate version 9.9.9 does not match release 1.2.3"; got != want {
			t.Fatalf("verifyCandidateVersion() error = %q, want %q", got, want)
		}
	})

	for _, tt := range []struct {
		name    string
		command string
	}{
		{name: "empty", command: ":"},
		{name: "whitespace only", command: "printf ' \\t\\n'"},
	} {
		t.Run("rejects "+tt.name+" output", func(t *testing.T) {
			candidatePath := writeCandidateFixture(t, tt.command)

			err := verifyCandidateVersion(context.Background(), candidatePath, "v1.2.3")
			if err == nil {
				t.Fatal("verifyCandidateVersion() error = nil, want empty output error")
			}
			if got, want := err.Error(), "candidate version output is empty"; got != want {
				t.Fatalf("verifyCandidateVersion() error = %q, want %q", got, want)
			}
		})
	}

	t.Run("wraps nonzero process error with operation", func(t *testing.T) {
		candidatePath := writeCandidateFixture(t, "exit 7")

		err := verifyCandidateVersion(context.Background(), candidatePath, "v1.2.3")
		if err == nil {
			t.Fatal("verifyCandidateVersion() error = nil, want process error")
		}
		if !strings.Contains(err.Error(), "run candidate version:") {
			t.Fatalf("verifyCandidateVersion() error = %q, want operation context", err)
		}
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("verifyCandidateVersion() error = %v, want wrapped *exec.ExitError", err)
		}
	})

	t.Run("preserves parent cancellation", func(t *testing.T) {
		candidatePath := writeCandidateFixture(t, "printf 'v1.2.3\\n'")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := verifyCandidateVersion(ctx, candidatePath, "v1.2.3")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("verifyCandidateVersion() error = %v, want context.Canceled", err)
		}
	})

	t.Run("honors short parent deadline promptly", func(t *testing.T) {
		candidatePath := writeCandidateFixture(t, "exec sleep 30")
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		started := time.Now()
		err := verifyCandidateVersion(ctx, candidatePath, "v1.2.3")
		elapsed := time.Since(started)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("verifyCandidateVersion() error = %v, want context.DeadlineExceeded", err)
		}
		if elapsed > 2*time.Second {
			t.Fatalf("verifyCandidateVersion() elapsed = %v, want <= 2s", elapsed)
		}
	})

	t.Run("wraps missing candidate path error", func(t *testing.T) {
		candidatePath := filepath.Join(t.TempDir(), "missing-tnnl")

		err := verifyCandidateVersion(context.Background(), candidatePath, "v1.2.3")
		if err == nil {
			t.Fatal("verifyCandidateVersion() error = nil, want missing path error")
		}
		if !strings.Contains(err.Error(), "run candidate version:") {
			t.Fatalf("verifyCandidateVersion() error = %q, want operation context", err)
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("verifyCandidateVersion() error = %v, want wrapped os.ErrNotExist", err)
		}
	})
}

func writeCandidateFixture(t *testing.T, command string) string {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("shell candidate fixture requires Darwin or Linux")
	}

	candidatePath := filepath.Join(t.TempDir(), "tnnl-candidate")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" != \"version\" ]; then exit 64; fi\n" +
		command + "\n"
	if err := os.WriteFile(candidatePath, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile(%q) error = %v", candidatePath, err)
	}

	return candidatePath
}
