package exec

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/wim-web/tnnl/internal/input"
)

func TestExecCommandExplicitFlagsOverrideFile(t *testing.T) {
	path := writeExecFixture(t, `{"cluster":"cluster","service":"service","command":"bash","wait":10}`)
	var got input.ExecInput
	command := newExecCommand(func(_ context.Context, in input.ExecInput) error {
		got = in
		return nil
	})
	command.SetArgs([]string{"--input-file", path, "--command", "zsh", "--wait", "0"})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	want := input.ExecInput{
		EcsParameter: input.EcsParameter{Cluster: "cluster", Service: "service"},
		Cmd:          "zsh",
		Wait:         0,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runner input = %#v, want %#v", got, want)
	}
}

func TestExecCommandInvalidInputDoesNotInvokeRunner(t *testing.T) {
	path := writeExecFixture(t, `{"command":" ","wait":-1}`)
	calls := 0
	command := newExecCommand(func(_ context.Context, _ input.ExecInput) error {
		calls++
		return nil
	})
	command.SetArgs([]string{"--input-file", path})

	err := command.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("ExecuteContext() error = nil, want validation error")
	}
	for _, want := range []string{"command is required", "wait must be non-negative"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ExecuteContext() error = %q, want substring %q", err, want)
		}
	}
	if calls != 0 {
		t.Fatalf("runner calls = %d, want 0", calls)
	}
}

func TestExecCommandPassesExecuteContextToRunner(t *testing.T) {
	type contextKey struct{}
	want := "context value"
	ctx := context.WithValue(context.Background(), contextKey{}, want)
	var got any
	command := newExecCommand(func(ctx context.Context, _ input.ExecInput) error {
		got = ctx.Value(contextKey{})
		return nil
	})
	command.SetArgs([]string{})

	if err := command.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if got != want {
		t.Fatalf("runner context value = %#v, want %#v", got, want)
	}
}

func TestExecCommandInputFileHelpNamesParent(t *testing.T) {
	command := newExecCommand(func(context.Context, input.ExecInput) error { return nil })
	if command.Short == "" {
		t.Fatal("Short is empty")
	}
	flag := command.Flags().Lookup(inputFileName)
	if flag == nil {
		t.Fatal("input-file flag = nil")
	}
	if !strings.Contains(flag.Usage, "tnnl exec make-input-file") {
		t.Fatalf("input-file usage = %q, want parent-specific generator", flag.Usage)
	}
}

func writeExecFixture(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "exec.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write input fixture: %v", err)
	}
	return path
}
