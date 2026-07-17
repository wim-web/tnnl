package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wim-web/tnnl/internal/session_manager"
)

type terminateFunc func(context.Context, string) error

type RemoteSession struct {
	ID             string
	Invocation     session_manager.Invocation
	terminate      terminateFunc
	cleanupTimeout time.Duration
}

func (s RemoteSession) Run(ctx context.Context, plugin session_manager.Plugin) error {
	if err := plugin.Run(ctx, s.Invocation); err != nil {
		pluginErr := fmt.Errorf("session-manager-plugin handoff failed: %w", err)
		return cleanupCreatedSession(ctx, s.ID, s.cleanupTimeout, s.terminate, pluginErr)
	}
	return nil
}

func cleanupCreatedSession(
	ctx context.Context,
	sessionID string,
	timeout time.Duration,
	terminate terminateFunc,
	primary error,
) error {
	if sessionID == "" {
		return primary
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
	defer cancel()

	if cleanupErr := terminate(cleanupCtx, sessionID); cleanupErr != nil {
		return errors.Join(
			primary,
			fmt.Errorf("terminate remote session %s: %w", sessionID, cleanupErr),
		)
	}
	return primary
}
