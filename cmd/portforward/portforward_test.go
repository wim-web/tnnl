package portforward

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

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

func writePortforwardFixture(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "portforward.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write input fixture: %v", err)
	}
	return path
}
