package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCLIBinaryContracts(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "tnnl")
	buildCtx, cancelBuild := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelBuild()
	build := exec.CommandContext(buildCtx, "go", "build", "-trimpath",
		"-ldflags=-X=github.com/wim-web/tnnl/internal/buildinfo.linkerVersion=v1.2.3",
		"-o", binary, ".")
	build.WaitDelay = time.Second
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build release-version CLI: %v\n%s", err, output)
	}

	t.Run("linker version", func(t *testing.T) {
		for _, tt := range []struct {
			name string
			args []string
		}{
			{name: "subcommand", args: []string{"version"}},
			{name: "long flag", args: []string{"--version"}},
			{name: "short flag", args: []string{"-v"}},
		} {
			t.Run(tt.name, func(t *testing.T) {
				if got := runCLIBinary(t, binary, t.TempDir(), tt.args...); got != "1.2.3\n" {
					t.Fatalf("version output = %q, want linker version 1.2.3", got)
				}
			})
		}
	})

	t.Run("registered input templates", func(t *testing.T) {
		for _, tt := range []struct {
			parent   string
			filename string
			want     map[string]any
		}{
			{
				parent: "exec", filename: "exec-input.json",
				want: map[string]any{"cluster": "", "service": "", "command": "sh", "wait": float64(0)},
			},
			{
				parent: "portforward", filename: "portforward-input.json",
				want: map[string]any{"cluster": "", "service": "", "target_port_number": "", "local_port_number": ""},
			},
			{
				parent: "remoteportforward", filename: "remoteportforward-input.json",
				want: map[string]any{"cluster": "", "service": "", "remote_port_number": "", "local_port_number": "", "host": ""},
			},
		} {
			t.Run(tt.parent, func(t *testing.T) {
				dir := t.TempDir()
				help := runCLIBinary(t, binary, dir, tt.parent, "make-input-file", "--help")
				if want := "Create an input file template for " + tt.parent; !strings.Contains(help, want) {
					t.Fatalf("template help = %q, want %q", help, want)
				}
				if got, want := runCLIBinary(t, binary, dir, tt.parent, "make-input-file"), "made "+tt.filename+"\n"; got != want {
					t.Fatalf("template command output = %q, want %q", got, want)
				}
				contents, err := os.ReadFile(filepath.Join(dir, tt.filename))
				if err != nil {
					t.Fatal(err)
				}
				var got map[string]any
				if err := json.Unmarshal(contents, &got); err != nil {
					t.Fatalf("decode generated template: %v", err)
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Fatalf("template = %#v, want %#v", got, tt.want)
				}
			})
		}
	})
}

func runCLIBinary(t *testing.T, binary, dir string, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, binary, args...)
	command.WaitDelay = time.Second
	command.Dir = dir
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("tnnl %v: %v\nstdout: %s\nstderr: %s", args, err, &stdout, &stderr)
	}
	if stderr.Len() != 0 {
		t.Fatalf("tnnl %v wrote unexpected stderr: %s", args, &stderr)
	}
	return stdout.String()
}
