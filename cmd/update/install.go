package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const candidateVersionWaitDelay = 200 * time.Millisecond

func verifyCandidateVersion(ctx context.Context, candidatePath, releaseTag string) error {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	candidate := exec.CommandContext(checkCtx, candidatePath, "version")
	candidate.WaitDelay = candidateVersionWaitDelay
	output, err := candidate.Output()
	if err != nil {
		runErr := fmt.Errorf("run candidate version: %w", err)
		if contextErr := checkCtx.Err(); contextErr != nil {
			return errors.Join(runErr, contextErr)
		}
		return runErr
	}

	got := normalizeVersion(string(output))
	if got == "" {
		return fmt.Errorf("candidate version output is empty")
	}

	want := normalizeVersion(releaseTag)
	if got != want {
		return fmt.Errorf("candidate version %s does not match release %s", got, want)
	}

	return nil
}

func replaceExecutable(currentPath string, newBinaryPath string) error {
	candidate, err := os.Open(newBinaryPath)
	if err != nil {
		return fmt.Errorf("open candidate executable: %w", err)
	}
	defer candidate.Close()

	replacement, err := os.CreateTemp(filepath.Dir(currentPath), ".tnnl.new-*")
	if err != nil {
		return fmt.Errorf("create replacement temp: %w", err)
	}
	replacementPath := replacement.Name()
	committed := false
	defer func() {
		if committed {
			return
		}
		_ = replacement.Close()
		_ = os.Remove(replacementPath)
	}()

	if _, err := io.Copy(replacement, candidate); err != nil {
		return fmt.Errorf("copy candidate executable: %w", err)
	}
	if err := replacement.Chmod(0o755); err != nil {
		return fmt.Errorf("set replacement executable mode: %w", err)
	}
	if err := replacement.Sync(); err != nil {
		return fmt.Errorf("sync replacement executable: %w", err)
	}
	if err := replacement.Close(); err != nil {
		return fmt.Errorf("close replacement executable: %w", err)
	}
	if err := os.Rename(replacementPath, currentPath); err != nil {
		return fmt.Errorf("replace executable (%s): %w", currentPath, err)
	}
	committed = true

	return nil
}
