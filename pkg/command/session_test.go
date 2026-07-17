package command

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/wim-web/tnnl/internal/session_manager"
)

func TestRemoteSessionTerminatesWhenPluginFails(t *testing.T) {
	pluginErr := errors.New("plugin failed")
	wantInvocation := validInvocation()
	var gotInvocation session_manager.Invocation
	var terminatedID string
	terminateCalls := 0
	session := RemoteSession{
		ID:         "s-1",
		Invocation: wantInvocation,
		terminate: func(ctx context.Context, id string) error {
			terminateCalls++
			if err := ctx.Err(); err != nil {
				t.Fatalf("cleanup context already canceled: %v", err)
			}
			terminatedID = id
			return nil
		},
		cleanupTimeout: time.Second,
	}

	err := session.Run(context.Background(), pluginFunc(func(_ context.Context, invocation session_manager.Invocation) error {
		gotInvocation = invocation
		return pluginErr
	}))

	if !errors.Is(err, pluginErr) {
		t.Fatalf("Run() error = %v, want errors.Is(pluginErr)", err)
	}
	if got := err.Error(); !strings.Contains(got, "session-manager-plugin handoff failed: plugin failed") {
		t.Fatalf("Run() error = %q, want handoff context", got)
	}
	if !reflect.DeepEqual(gotInvocation, wantInvocation) {
		t.Fatalf("plugin invocation = %#v, want %#v", gotInvocation, wantInvocation)
	}
	if terminateCalls != 1 || terminatedID != session.ID {
		t.Fatalf("terminate calls = %d, session ID = %q; want 1, %q", terminateCalls, terminatedID, session.ID)
	}
}

func TestRemoteSessionDoesNotTerminateAfterSuccessfulPlugin(t *testing.T) {
	terminateCalls := 0
	session := validRemoteSession(func(context.Context, string) error {
		terminateCalls++
		return nil
	})

	if err := session.Run(context.Background(), pluginFunc(func(context.Context, session_manager.Invocation) error {
		return nil
	})); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if terminateCalls != 0 {
		t.Fatalf("terminate calls = %d, want 0", terminateCalls)
	}
}

func TestRemoteSessionJoinsCleanupError(t *testing.T) {
	pluginErr := errors.New("plugin failed")
	cleanupErr := errors.New("terminate failed")
	session := validRemoteSession(func(context.Context, string) error {
		return cleanupErr
	})

	err := session.Run(context.Background(), pluginFunc(func(context.Context, session_manager.Invocation) error {
		return pluginErr
	}))

	if !errors.Is(err, pluginErr) {
		t.Fatalf("Run() error = %v, want errors.Is(pluginErr)", err)
	}
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("Run() error = %v, want errors.Is(cleanupErr)", err)
	}
	if got := err.Error(); !strings.Contains(got, "terminate remote session s-1: terminate failed") {
		t.Fatalf("Run() error = %q, want cleanup context", got)
	}
}

func TestRemoteSessionCleanupUsesLiveIndependentContext(t *testing.T) {
	type cleanupContextKey struct{}
	const cleanupValue = "keep-me"
	const cleanupTimeout = 500 * time.Millisecond

	parentCtx, cancel := context.WithCancel(context.WithValue(context.Background(), cleanupContextKey{}, cleanupValue))
	cancel()
	pluginErr := errors.New("plugin failed")
	startedAt := time.Now()
	cleanupCalled := false
	session := RemoteSession{
		ID:         "s-1",
		Invocation: validInvocation(),
		terminate: func(cleanupCtx context.Context, _ string) error {
			cleanupCalled = true
			if err := cleanupCtx.Err(); err != nil {
				t.Fatalf("cleanup context error at invocation = %v, want live context", err)
			}
			if got := cleanupCtx.Value(cleanupContextKey{}); got != cleanupValue {
				t.Fatalf("cleanup context value = %#v, want %q", got, cleanupValue)
			}
			deadline, ok := cleanupCtx.Deadline()
			if !ok {
				t.Fatal("cleanup context has no deadline")
			}
			if !deadline.After(startedAt) {
				t.Fatalf("cleanup deadline = %v, want after cleanup start %v", deadline, startedAt)
			}
			if remaining := time.Until(deadline); remaining <= 0 || remaining > cleanupTimeout {
				t.Fatalf("cleanup deadline remaining = %v, want within (0, %v]", remaining, cleanupTimeout)
			}
			return nil
		},
		cleanupTimeout: cleanupTimeout,
	}

	err := session.Run(parentCtx, pluginFunc(func(context.Context, session_manager.Invocation) error {
		return pluginErr
	}))

	if !errors.Is(err, pluginErr) {
		t.Fatalf("Run() error = %v, want errors.Is(pluginErr)", err)
	}
	if !cleanupCalled {
		t.Fatal("terminate was not called")
	}
}

func TestRemoteSessionDoesNotTerminateWithoutID(t *testing.T) {
	pluginErr := errors.New("plugin failed")
	terminateCalls := 0
	session := validRemoteSession(func(context.Context, string) error {
		terminateCalls++
		return nil
	})
	session.ID = ""

	err := session.Run(context.Background(), pluginFunc(func(context.Context, session_manager.Invocation) error {
		return pluginErr
	}))

	if !errors.Is(err, pluginErr) {
		t.Fatalf("Run() error = %v, want errors.Is(pluginErr)", err)
	}
	if terminateCalls != 0 {
		t.Fatalf("terminate calls = %d, want 0", terminateCalls)
	}
}

func TestRemoteSessionCleanupTimeoutIsDiscoverable(t *testing.T) {
	pluginErr := errors.New("plugin failed")
	session := RemoteSession{
		ID:         "s-1",
		Invocation: validInvocation(),
		terminate: func(ctx context.Context, _ string) error {
			<-ctx.Done()
			return ctx.Err()
		},
		cleanupTimeout: 20 * time.Millisecond,
	}

	err := session.Run(context.Background(), pluginFunc(func(context.Context, session_manager.Invocation) error {
		return pluginErr
	}))

	if !errors.Is(err, pluginErr) {
		t.Fatalf("Run() error = %v, want errors.Is(pluginErr)", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want errors.Is(DeadlineExceeded)", err)
	}
}

func TestRemoteSessionPassesOriginalContextToPlugin(t *testing.T) {
	type pluginContextKey struct{}
	const pluginValue = "plugin-value"
	cancelCause := errors.New("caller canceled")
	ctx, cancel := context.WithCancelCause(context.WithValue(context.Background(), pluginContextKey{}, pluginValue))
	cancel(cancelCause)
	pluginCalled := false
	session := validRemoteSession(func(context.Context, string) error {
		t.Fatal("terminate called for session without ID")
		return nil
	})
	session.ID = ""

	err := session.Run(ctx, pluginFunc(func(pluginCtx context.Context, invocation session_manager.Invocation) error {
		pluginCalled = true
		if pluginCtx != ctx {
			t.Fatalf("plugin context = %v, want original context %v", pluginCtx, ctx)
		}
		if got := pluginCtx.Value(pluginContextKey{}); got != pluginValue {
			t.Fatalf("plugin context value = %#v, want %q", got, pluginValue)
		}
		if got := context.Cause(pluginCtx); !errors.Is(got, cancelCause) {
			t.Fatalf("plugin context cause = %v, want %v", got, cancelCause)
		}
		if !reflect.DeepEqual(invocation, session.Invocation) {
			t.Fatalf("plugin invocation = %#v, want %#v", invocation, session.Invocation)
		}
		return cancelCause
	}))

	if !pluginCalled {
		t.Fatal("plugin was not called")
	}
	if !errors.Is(err, cancelCause) {
		t.Fatalf("Run() error = %v, want errors.Is(cancelCause)", err)
	}
}

type pluginFunc func(context.Context, session_manager.Invocation) error

func (f pluginFunc) Run(ctx context.Context, invocation session_manager.Invocation) error {
	return f(ctx, invocation)
}

func validRemoteSession(terminate terminateFunc) RemoteSession {
	return RemoteSession{
		ID:             "s-1",
		Invocation:     validInvocation(),
		terminate:      terminate,
		cleanupTimeout: time.Second,
	}
}

func validInvocation() session_manager.Invocation {
	return session_manager.Invocation{
		Response: session_manager.SessionResponse{
			SessionID:  "s-1",
			StreamURL:  "wss://example",
			TokenValue: "token",
		},
		Region: "ap-northeast-1",
		Target: "ecs:cluster_task_runtime",
	}
}
