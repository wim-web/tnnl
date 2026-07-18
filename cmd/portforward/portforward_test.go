package portforward

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/internal/input"
)

func TestPortforwardCommandOverridesOnlyExplicitFlags(t *testing.T) {
	path := writePortforwardFixture(t, `{
		"cluster":"cluster",
		"service":"service",
		"target_port_number":"80",
		"local_port_number":"8080"
	}`)

	tests := []struct {
		name string
		args []string
		want input.PortForwardInput
	}{
		{
			name: "no overrides",
			want: input.PortForwardInput{
				EcsParameter:     input.EcsParameter{Cluster: "cluster", Service: "service"},
				TargetPortNumber: "80",
				LocalPortNumber:  "8080",
			},
		},
		{
			name: "target only",
			args: []string{"--target-port", "443"},
			want: input.PortForwardInput{
				EcsParameter:     input.EcsParameter{Cluster: "cluster", Service: "service"},
				TargetPortNumber: "443",
				LocalPortNumber:  "8080",
			},
		},
		{
			name: "local only",
			args: []string{"--local-port", "9000"},
			want: input.PortForwardInput{
				EcsParameter:     input.EcsParameter{Cluster: "cluster", Service: "service"},
				TargetPortNumber: "80",
				LocalPortNumber:  "9000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got input.PortForwardInput
			calls := 0
			command := newPortforwardCommand(func(_ context.Context, in input.PortForwardInput) error {
				calls++
				got = in
				return nil
			})
			command.SetArgs(append([]string{"--input-file", path}, tt.args...))

			if err := command.ExecuteContext(context.Background()); err != nil {
				t.Fatalf("ExecuteContext() error = %v", err)
			}
			if calls != 1 {
				t.Fatalf("runner calls = %d, want 1", calls)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("runner input = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestPortforwardCommandPreservesExplicitEmptyLocalPort(t *testing.T) {
	path := writePortforwardFixture(t, `{"target_port_number":"80","local_port_number":"8080"}`)
	var got input.PortForwardInput
	command := newPortforwardCommand(func(_ context.Context, in input.PortForwardInput) error {
		got = in
		return nil
	})
	command.SetArgs([]string{"--input-file", path, "--local-port", ""})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if got.TargetPortNumber != "80" || got.LocalPortNumber != "" {
		t.Fatalf("runner input = %#v, want target 80 and empty local port", got)
	}
}

func TestPortforwardCommandPassesExecuteContextToRunner(t *testing.T) {
	type contextKey struct{}
	want := "portforward context value"
	ctx := context.WithValue(context.Background(), contextKey{}, want)
	var got any
	command := newPortforwardCommand(func(ctx context.Context, _ input.PortForwardInput) error {
		got = ctx.Value(contextKey{})
		return nil
	})
	command.SetArgs([]string{"--target-port", "80"})

	if err := command.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if got != want {
		t.Fatalf("runner context value = %#v, want %#v", got, want)
	}
}

func TestPortforwardCommandInvalidExplicitFlagDoesNotInvokeRunner(t *testing.T) {
	path := writePortforwardFixture(t, `{"target_port_number":"80","local_port_number":"8080"}`)
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "empty target", args: []string{"--target-port", ""}, wantErr: "target port is required"},
		{name: "invalid target", args: []string{"--target-port", "not-a-port"}, wantErr: "target port must be a decimal integer"},
		{name: "invalid local", args: []string{"--local-port", "0"}, wantErr: "local port must be between 1 and 65535"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			command := newPortforwardCommand(func(_ context.Context, _ input.PortForwardInput) error {
				calls++
				return nil
			})
			command.SetArgs(append([]string{"--input-file", path}, tt.args...))

			err := command.ExecuteContext(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ExecuteContext() error = %v, want substring %q", err, tt.wantErr)
			}
			if calls != 0 {
				t.Fatalf("runner calls = %d, want 0", calls)
			}
		})
	}
}

func TestPortforwardCommandInputFileHelpNamesParent(t *testing.T) {
	command := newPortforwardCommand(func(context.Context, input.PortForwardInput) error { return nil })
	flag := command.Flags().Lookup(inputFileName)
	if flag == nil {
		t.Fatal("input-file flag = nil")
	}
	if !strings.Contains(flag.Usage, "tnnl portforward make-input-file") {
		t.Fatalf("input-file usage = %q, want parent-specific generator", flag.Usage)
	}
}

func TestPortforwardHelpDocumentsInputAndAutomaticLocalPort(t *testing.T) {
	command := newPortforwardCommand(func(context.Context, input.PortForwardInput) error { return nil })

	assertHelpContains(t, command,
		"tnnl portforward make-input-file",
		"automatic local-port selection",
		"omitted or the zero value",
		"explicit flag > input JSON > default",
	)
}

func assertHelpContains(t *testing.T, command *cobra.Command, values ...string) {
	t.Helper()

	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	if err := command.Help(); err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		if !strings.Contains(output.String(), value) {
			t.Errorf("help does not contain %q:\n%s", value, output.String())
		}
	}
}

func writePortforwardFixture(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "portforward.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write input fixture: %v", err)
	}
	return path
}
