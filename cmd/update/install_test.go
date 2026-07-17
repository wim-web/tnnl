package update

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	candidateHelperModeEnv       = "GO_WANT_UPDATE_CANDIDATE_HELPER"
	candidateHelperStartedEnv    = "UPDATE_CANDIDATE_HELPER_STARTED"
	candidateHelperModeParent    = "parent"
	candidateHelperModeChild     = "child"
	candidateHelperChildLifetime = time.Second
	candidateHelperSyncLimit     = 5 * time.Second
	candidateHelperResultLimit   = 2 * time.Second
	candidateHelperPollInterval  = 10 * time.Millisecond
)

func TestMain(m *testing.M) {
	switch os.Getenv(candidateHelperModeEnv) {
	case candidateHelperModeParent:
		os.Exit(runCandidateParentHelper())
	case candidateHelperModeChild:
		time.Sleep(candidateHelperChildLifetime)
		os.Exit(0)
	default:
		os.Exit(m.Run())
	}
}

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

	t.Run("bounds inherited stdout pipe cleanup", func(t *testing.T) {
		candidatePath, err := os.Executable()
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
			errCh <- verifyCandidateVersion(ctx, candidatePath, "v1.2.3")
		}()

		if err := waitForCandidateHelper(startedPath, candidateHelperSyncLimit); err != nil {
			ctx.expire()
			verifyErr := receiveCandidateVerification(t, errCh, candidateHelperResultLimit)
			t.Fatalf("wait for candidate helper: %v; verifyCandidateVersion() error = %v", err, verifyErr)
		}

		started := time.Now()
		ctx.expire()
		err = receiveCandidateVerification(t, errCh, candidateHelperResultLimit)
		elapsed := time.Since(started)

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("verifyCandidateVersion() error = %v, want context.DeadlineExceeded", err)
		}
		if elapsed >= 500*time.Millisecond {
			t.Fatalf("verifyCandidateVersion() elapsed = %v, want < 500ms", elapsed)
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

func TestReplaceExecutablePreservesLegacySentinel(t *testing.T) {
	dir := t.TempDir()
	currentPath := filepath.Join(dir, "tnnl")
	candidatePath := filepath.Join(dir, "candidate")
	sentinelPath := filepath.Join(dir, ".tnnl.new")
	if err := os.WriteFile(currentPath, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidatePath, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sentinelPath, []byte("sentinel"), 0o640); err != nil {
		t.Fatal(err)
	}
	sentinelBefore, err := os.Stat(sentinelPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := replaceExecutable(currentPath, candidatePath); err != nil {
		t.Fatalf("replaceExecutable() error = %v", err)
	}

	sentinelAfter, err := os.Stat(sentinelPath)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v; legacy sentinel must remain", sentinelPath, err)
	}
	if !os.SameFile(sentinelBefore, sentinelAfter) {
		t.Fatal("legacy sentinel was replaced")
	}
	if got, want := sentinelAfter.Mode().Perm(), os.FileMode(0o640); got != want {
		t.Fatalf("legacy sentinel mode = %o, want %o", got, want)
	}
	content, err := os.ReadFile(sentinelPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "sentinel"; got != want {
		t.Fatalf("legacy sentinel content = %q, want %q", got, want)
	}
}

func TestReplaceExecutableSuccess(t *testing.T) {
	dir := t.TempDir()
	currentPath := filepath.Join(dir, "tnnl")
	candidatePath := filepath.Join(dir, "candidate")
	if err := os.WriteFile(currentPath, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidatePath, []byte("new"), 0o640); err != nil {
		t.Fatal(err)
	}

	if err := replaceExecutable(currentPath, candidatePath); err != nil {
		t.Fatalf("replaceExecutable() error = %v", err)
	}

	content, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "new"; got != want {
		t.Fatalf("replacement content = %q, want %q", got, want)
	}
	info, err := os.Stat(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o755); got != want {
		t.Fatalf("replacement mode = %o, want %o", got, want)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".tnnl.new-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("replacement temp remnants = %q, want none", matches)
	}
}

func TestReplaceExecutableFailureCleansUpOnlyOwnTemp(t *testing.T) {
	t.Run("candidate open", func(t *testing.T) {
		fixture := newReplacementFailureFixture(t)
		fixture.candidatePath = filepath.Join(fixture.dir, "missing-candidate")

		assertReplacementFailure(t, fixture, "open candidate executable")
	})

	t.Run("candidate copy", func(t *testing.T) {
		if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
			t.Skip("directory read failure fixture requires Darwin or Linux")
		}
		fixture := newReplacementFailureFixture(t)
		if err := os.Remove(fixture.candidatePath); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(fixture.candidatePath, 0o755); err != nil {
			t.Fatal(err)
		}

		assertReplacementFailure(t, fixture, "copy candidate executable")
	})

	t.Run("create temp", func(t *testing.T) {
		fixture := newReplacementFailureFixture(t)
		notDirectoryPath := filepath.Join(fixture.dir, "not-a-directory")
		if err := os.WriteFile(notDirectoryPath, []byte("file"), 0o600); err != nil {
			t.Fatal(err)
		}
		fixture.currentPath = filepath.Join(notDirectoryPath, "tnnl")

		assertReplacementFailure(t, fixture, "create replacement temp")
	})

	t.Run("rename", func(t *testing.T) {
		fixture := newReplacementFailureFixture(t)
		if err := os.Remove(fixture.currentPath); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(fixture.currentPath, 0o755); err != nil {
			t.Fatal(err)
		}

		assertReplacementFailure(t, fixture, "replace executable")
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

type replacementFileSnapshot struct {
	info    os.FileInfo
	content []byte
}

type replacementFailureFixture struct {
	dir            string
	currentPath    string
	candidatePath  string
	otherTempPath  string
	preservedFiles map[string]replacementFileSnapshot
}

func newReplacementFailureFixture(t *testing.T) *replacementFailureFixture {
	t.Helper()
	dir := t.TempDir()
	currentPath := filepath.Join(dir, "tnnl")
	candidatePath := filepath.Join(dir, "candidate")
	sentinelPath := filepath.Join(dir, ".tnnl.new")
	otherTempPath := filepath.Join(dir, ".tnnl.new-unrelated")
	for path, content := range map[string]string{
		currentPath:   "old",
		candidatePath: "new",
		sentinelPath:  "sentinel",
		otherTempPath: "unrelated",
	} {
		if err := os.WriteFile(path, []byte(content), 0o640); err != nil {
			t.Fatalf("os.WriteFile(%q) error = %v", path, err)
		}
	}

	return &replacementFailureFixture{
		dir:           dir,
		currentPath:   currentPath,
		candidatePath: candidatePath,
		otherTempPath: otherTempPath,
		preservedFiles: map[string]replacementFileSnapshot{
			sentinelPath:  snapshotReplacementFile(t, sentinelPath),
			otherTempPath: snapshotReplacementFile(t, otherTempPath),
		},
	}
}

func snapshotReplacementFile(t *testing.T, path string) replacementFileSnapshot {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v", path, err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	return replacementFileSnapshot{info: info, content: content}
}

func assertReplacementFailure(t *testing.T, fixture *replacementFailureFixture, wantOperation string) {
	t.Helper()
	err := replaceExecutable(fixture.currentPath, fixture.candidatePath)
	if err == nil {
		t.Fatalf("replaceExecutable() error = nil, want %q failure", wantOperation)
	}
	if !strings.Contains(err.Error(), wantOperation) {
		t.Fatalf("replaceExecutable() error = %q, want operation %q", err, wantOperation)
	}

	for path, before := range fixture.preservedFiles {
		after := snapshotReplacementFile(t, path)
		if !os.SameFile(before.info, after.info) {
			t.Fatalf("preserved file %q was replaced", path)
		}
		if after.info.Mode() != before.info.Mode() {
			t.Fatalf("preserved file %q mode = %v, want %v", path, after.info.Mode(), before.info.Mode())
		}
		if got, want := string(after.content), string(before.content); got != want {
			t.Fatalf("preserved file %q content = %q, want %q", path, got, want)
		}
	}

	matches, globErr := filepath.Glob(filepath.Join(fixture.dir, ".tnnl.new-*"))
	if globErr != nil {
		t.Fatal(globErr)
	}
	if len(matches) != 1 || matches[0] != fixture.otherTempPath {
		t.Fatalf("replacement temps = %q, want only %q", matches, fixture.otherTempPath)
	}
}

func runCandidateParentHelper() int {
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if err := os.Setenv(candidateHelperModeEnv, candidateHelperModeChild); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	child := exec.Command(executable)
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if err := os.WriteFile(os.Getenv(candidateHelperStartedEnv), []byte("started"), 0o600); err != nil {
		_ = child.Process.Kill()
		_ = child.Wait()
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if err := child.Wait(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	return 0
}

type manualDeadlineContext struct {
	done chan struct{}
	once sync.Once
}

func newManualDeadlineContext() *manualDeadlineContext {
	return &manualDeadlineContext{done: make(chan struct{})}
}

func (*manualDeadlineContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (ctx *manualDeadlineContext) Done() <-chan struct{} {
	return ctx.done
}

func (ctx *manualDeadlineContext) Err() error {
	select {
	case <-ctx.done:
		return context.DeadlineExceeded
	default:
		return nil
	}
}

func (*manualDeadlineContext) Value(any) any {
	return nil
}

func (ctx *manualDeadlineContext) expire() {
	ctx.once.Do(func() {
		close(ctx.done)
	})
}

func waitForCandidateHelper(path string, limit time.Duration) error {
	timer := time.NewTimer(limit)
	defer timer.Stop()
	ticker := time.NewTicker(candidateHelperPollInterval)
	defer ticker.Stop()

	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect candidate helper start notification: %w", err)
		}

		select {
		case <-ticker.C:
		case <-timer.C:
			return fmt.Errorf("timed out after %s", limit)
		}
	}
}

func receiveCandidateVerification(t *testing.T, errCh <-chan error, limit time.Duration) error {
	t.Helper()
	timer := time.NewTimer(limit)
	defer timer.Stop()

	select {
	case err := <-errCh:
		return err
	case <-timer.C:
		t.Fatalf("timed out after %s waiting for verifyCandidateVersion", limit)
		return nil
	}
}
