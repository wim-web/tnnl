package remoteportforward

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

func TestRemotePortforwardCommandOverridesOnlyExplicitFlags(t *testing.T) {
	path := writeRemotePortforwardFixture(t, `{
		"cluster":"cluster",
		"service":"service",
		"remote_port_number":"22",
		"local_port_number":"2222",
		"host":"old.example.com"
	}`)

	tests := []struct {
		name string
		args []string
		want input.RemotePortForwardInput
	}{
		{
			name: "no overrides",
			want: input.RemotePortForwardInput{
				EcsParameter:     input.EcsParameter{Cluster: "cluster", Service: "service"},
				RemotePortNumber: "22",
				LocalPortNumber:  "2222",
				Host:             "old.example.com",
			},
		},
		{
			name: "remote only",
			args: []string{"--remote-port", "443"},
			want: input.RemotePortForwardInput{
				EcsParameter:     input.EcsParameter{Cluster: "cluster", Service: "service"},
				RemotePortNumber: "443",
				LocalPortNumber:  "2222",
				Host:             "old.example.com",
			},
		},
		{
			name: "local only and empty",
			args: []string{"--local-port", ""},
			want: input.RemotePortForwardInput{
				EcsParameter:     input.EcsParameter{Cluster: "cluster", Service: "service"},
				RemotePortNumber: "22",
				LocalPortNumber:  "",
				Host:             "old.example.com",
			},
		},
		{
			name: "host only",
			args: []string{"--host", "new.example.com"},
			want: input.RemotePortForwardInput{
				EcsParameter:     input.EcsParameter{Cluster: "cluster", Service: "service"},
				RemotePortNumber: "22",
				LocalPortNumber:  "2222",
				Host:             "new.example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got input.RemotePortForwardInput
			calls := 0
			command := newRemotePortforwardCommand(func(_ context.Context, in input.RemotePortForwardInput) error {
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

func TestRemotePortforwardCommandAllExplicitFlagsOverrideFile(t *testing.T) {
	path := writeRemotePortforwardFixture(t, `{
		"remote_port_number":"22",
		"local_port_number":"2222",
		"host":"old.example.com"
	}`)
	var got input.RemotePortForwardInput
	command := newRemotePortforwardCommand(func(_ context.Context, in input.RemotePortForwardInput) error {
		got = in
		return nil
	})
	command.SetArgs([]string{
		"--input-file", path,
		"--remote-port", "443",
		"--local-port", "4443",
		"--host", "new.example.com",
	})

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	want := input.RemotePortForwardInput{
		RemotePortNumber: "443",
		LocalPortNumber:  "4443",
		Host:             "new.example.com",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("runner input = %#v, want %#v", got, want)
	}
}

func TestRemotePortforwardCommandPassesExecuteContextToRunner(t *testing.T) {
	type contextKey struct{}
	want := "remote portforward context value"
	ctx := context.WithValue(context.Background(), contextKey{}, want)
	var got any
	command := newRemotePortforwardCommand(func(ctx context.Context, _ input.RemotePortForwardInput) error {
		got = ctx.Value(contextKey{})
		return nil
	})
	command.SetArgs([]string{"--remote-port", "22", "--host", "example.com"})

	if err := command.ExecuteContext(ctx); err != nil {
		t.Fatalf("ExecuteContext() error = %v", err)
	}
	if got != want {
		t.Fatalf("runner context value = %#v, want %#v", got, want)
	}
}

func TestRemotePortforwardCommandInvalidRequiredValueDoesNotInvokeRunner(t *testing.T) {
	path := writeRemotePortforwardFixture(t, `{
		"remote_port_number":"22",
		"local_port_number":"2222",
		"host":"old.example.com"
	}`)
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "empty remote", args: []string{"--remote-port", ""}, wantErr: "remote port is required"},
		{name: "invalid remote", args: []string{"--remote-port", "ssh"}, wantErr: "remote port must be a decimal integer"},
		{name: "empty host", args: []string{"--host", ""}, wantErr: "host is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			command := newRemotePortforwardCommand(func(_ context.Context, _ input.RemotePortForwardInput) error {
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

func TestRemotePortforwardCommandInputFileHelpNamesParent(t *testing.T) {
	command := newRemotePortforwardCommand(func(context.Context, input.RemotePortForwardInput) error { return nil })
	flag := command.Flags().Lookup(inputFileName)
	if flag == nil {
		t.Fatal("input-file flag = nil")
	}
	if !strings.Contains(flag.Usage, "tnnl remoteportforward make-input-file") {
		t.Fatalf("input-file usage = %q, want parent-specific generator", flag.Usage)
	}
}

func TestRemotePortforwardHelpDocumentsRemoteHostAndAutomaticLocalPort(t *testing.T) {
	command := newRemotePortforwardCommand(func(context.Context, input.RemotePortForwardInput) error { return nil })

	assertHelpContains(t, command,
		"tnnl remoteportforward make-input-file",
		"remote host",
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

func writeRemotePortforwardFixture(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "remoteportforward.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write input fixture: %v", err)
	}
	return path
}
