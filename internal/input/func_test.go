package input

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadInputFileStrict(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{
			name:    "unknown field",
			content: `{"command":"sh","commnad":"bash"}`,
			wantErr: `unknown field "commnad"`,
		},
		{
			name:    "two JSON documents",
			content: `{ "command":"sh" } { "command":"bash" }`,
			wantErr: "exactly one JSON document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "exec-input.json")
			writeFixture(t, path, []byte(tt.content))

			var got ExecInput
			err := ReadInputFile(&got, path)
			if err == nil {
				t.Fatalf("ReadInputFile() error = nil, want an error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ReadInputFile() error = %q, want an error containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestMakeInputFileRefusesExistingFileUnlessForced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exec-input.json")
	original := []byte("keep me")
	writeFixture(t, path, original)

	err := MakeInputFile(ExecInput{Cmd: "sh"}, path, false)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("MakeInputFile(force=false) error = %v, want fs.ErrExist", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read existing input file: %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("MakeInputFile(force=false) changed existing bytes: got %q, want %q", got, original)
	}

	if err := MakeInputFile(ExecInput{Cmd: "bash"}, path, true); err != nil {
		t.Fatalf("MakeInputFile(force=true) error = %v", err)
	}

	var decoded ExecInput
	if err := ReadInputFile(&decoded, path); err != nil {
		t.Fatalf("ReadInputFile() after forced write error = %v", err)
	}
	if decoded.Cmd != "bash" {
		t.Fatalf("forced command = %q, want %q", decoded.Cmd, "bash")
	}
}

func TestInputFileUnicodeRoundTrip(t *testing.T) {
	type document struct {
		Text string `json:"text"`
	}

	path := filepath.Join(t.TempDir(), "unicode-input.json")
	want := document{Text: "こんにちは世界 😀"}
	if err := MakeInputFile(want, path, false); err != nil {
		t.Fatalf("MakeInputFile() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat input file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("input file mode = %o, want 600", got)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read input file: %v", err)
	}
	wantContent := "{\n  \"text\": \"こんにちは世界 😀\"\n}\n"
	if string(content) != wantContent {
		t.Fatalf("input file content = %q, want %q", content, wantContent)
	}

	var got document
	if err := ReadInputFile(&got, path); err != nil {
		t.Fatalf("ReadInputFile() error = %v", err)
	}
	if got != want {
		t.Fatalf("round-trip value = %#v, want %#v", got, want)
	}
}

func writeFixture(t *testing.T, path string, content []byte) {
	t.Helper()

	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}
