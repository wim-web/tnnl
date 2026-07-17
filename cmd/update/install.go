package update

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

func verifyCandidateVersion(ctx context.Context, candidatePath, releaseTag string) error {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(checkCtx, candidatePath, "version").Output()
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
