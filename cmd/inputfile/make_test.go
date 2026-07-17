package inputfile

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/wim-web/tnnl/internal/input"
)

func TestNewCreatesTemplateAtOutputAndReportsPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "custom.json")
	skeleton := input.ExecInput{Cmd: "sh", Wait: 3}
	var stdout bytes.Buffer
	command := New("exec", "unused.json", skeleton)
	command.SetOut(&stdout)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"--output", path})

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := stdout.String(), "made "+path+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated template: %v", err)
	}
	var got input.ExecInput
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("decode generated template: %v", err)
	}
	if got != skeleton {
		t.Fatalf("generated template = %#v, want %#v", got, skeleton)
	}
}

func TestNewUsesParentSpecificDescription(t *testing.T) {
	for _, parent := range []string{"exec", "portforward", "remoteportforward"} {
		t.Run(parent, func(t *testing.T) {
			command := New(parent, parent+"-input.json", struct{}{})
			if got, want := command.Use, "make-input-file"; got != want {
				t.Fatalf("Use = %q, want %q", got, want)
			}
			if got, want := command.Short, "Create an input file template for "+parent; got != want {
				t.Fatalf("Short = %q, want %q", got, want)
			}
		})
	}
}

func TestNewRefusesExistingFileAndPreservesContents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.json")
	original := []byte("keep these bytes exactly")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("write existing file: %v", err)
	}
	var stdout bytes.Buffer
	command := New("exec", "exec-input.json", input.ExecInput{Cmd: "sh"})
	command.SilenceUsage = true
	command.SetOut(&stdout)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"--output", path})

	err := command.Execute()
	if err == nil || !errors.Is(err, fs.ErrExist) {
		t.Fatalf("Execute() error = %v, want fs.ErrExist", err)
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read existing file: %v", readErr)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("existing bytes = %q, want %q", got, original)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty after refusal", stdout.String())
	}
}

func TestNewForceReplacesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.json")
	if err := os.WriteFile(path, []byte("stale trailing bytes that must disappear"), 0o600); err != nil {
		t.Fatalf("write existing file: %v", err)
	}
	skeleton := input.PortForwardInput{TargetPortNumber: "443", LocalPortNumber: "8443"}
	var stdout bytes.Buffer
	command := New("portforward", "portforward-input.json", skeleton)
	command.SetOut(&stdout)
	command.SetErr(io.Discard)
	command.SetArgs([]string{"--output", path, "--force"})

	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read replaced file: %v", err)
	}
	var got input.PortForwardInput
	if err := json.Unmarshal(content, &got); err != nil {
		t.Fatalf("decode replaced file: %v; content = %q", err, content)
	}
	if got != skeleton {
		t.Fatalf("replaced template = %#v, want %#v", got, skeleton)
	}
	if got, want := stdout.String(), "made "+path+"\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
}

func TestNewReturnsOutputWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "template.json")
	wantErr := errors.New("output failed")
	command := New("exec", "exec-input.json", input.ExecInput{})
	command.SetOut(failingWriter{err: wantErr})
	command.SetErr(io.Discard)
	command.SetArgs([]string{"--output", path})

	if err := command.Execute(); !errors.Is(err, wantErr) {
		t.Fatalf("Execute() error = %v, want %v", err, wantErr)
	}
}

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}
