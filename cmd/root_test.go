package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/internal/buildinfo"
)

func TestExecuteContextPropagatesCancellation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	prepareRootCommandTest(t, []string{"wait-for-cancellation"}, &stdout, &stderr)

	started := make(chan struct{})
	waitForCancellation := &cobra.Command{
		Use: "wait-for-cancellation",
		RunE: func(cmd *cobra.Command, args []string) error {
			close(started)
			<-cmd.Context().Done()

			return cmd.Context().Err()
		},
	}
	RootCmd.AddCommand(waitForCancellation)
	t.Cleanup(func() {
		RootCmd.RemoveCommand(waitForCancellation)
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	result := make(chan error, 1)
	go func() {
		result <- ExecuteContext(ctx)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		cancel()
		select {
		case <-result:
		case <-time.After(time.Second):
		}
		t.Fatal("child command did not start before the deadline")
	}

	cancel()

	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ExecuteContext() error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("ExecuteContext() did not return after cancellation")
	}
}

func TestExecuteContextReturnsChildErrorSilently(t *testing.T) {
	var stdout, stderr bytes.Buffer
	prepareRootCommandTest(t, []string{"fail"}, &stdout, &stderr)

	wantErr := errors.New("child failure")
	failingChild := &cobra.Command{
		Use: "fail",
		RunE: func(cmd *cobra.Command, args []string) error {
			return wantErr
		},
	}
	RootCmd.AddCommand(failingChild)
	t.Cleanup(func() {
		RootCmd.RemoveCommand(failingChild)
	})

	err := ExecuteContext(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("ExecuteContext() error = %v, want %v", err, wantErr)
	}
	if got := stderr.String(); got != "" {
		t.Errorf("RootCmd error output = %q, want empty output", got)
	}
	if got := stdout.String() + stderr.String(); strings.Contains(got, "Usage:") {
		t.Errorf("RootCmd output contains usage for an operational error: %q", got)
	}
}

func TestRootCommandMetadataAndFlags(t *testing.T) {
	prepareRootCommandTest(t, []string{}, io.Discard, io.Discard)

	if !RootCmd.SilenceErrors {
		t.Error("RootCmd.SilenceErrors = false, want true")
	}
	if !RootCmd.SilenceUsage {
		t.Error("RootCmd.SilenceUsage = false, want true")
	}
	if strings.TrimSpace(RootCmd.Short) == "" {
		t.Error("RootCmd.Short is empty")
	}
	if strings.TrimSpace(RootCmd.Long) == "" {
		t.Error("RootCmd.Long is empty")
	}

	description := RootCmd.Short + "\n" + RootCmd.Long
	for _, want := range []string{
		"AWS Systems Manager Session Manager",
		"ECS Exec",
		"port forwarding",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("RootCmd description = %q, want it to contain %q", description, want)
		}
	}

	versionFlag := RootCmd.Flags().Lookup("version")
	if versionFlag == nil {
		t.Fatal("RootCmd --version flag is missing")
	}
	if versionFlag.Shorthand != "v" {
		t.Errorf("RootCmd --version shorthand = %q, want %q", versionFlag.Shorthand, "v")
	}

	for _, name := range []string{"profile", "region", "cluster", "service"} {
		if flag := RootCmd.Flags().Lookup(name); flag != nil {
			t.Errorf("RootCmd unexpectedly defines --%s", name)
		}
		if flag := RootCmd.PersistentFlags().Lookup(name); flag != nil {
			t.Errorf("RootCmd unexpectedly defines persistent --%s", name)
		}
	}
}

func TestRootHelpDocumentsAWSSetup(t *testing.T) {
	prepareRootCommandTest(t, []string{}, io.Discard, io.Discard)

	assertHelpContains(t, RootCmd,
		"AWS SDK default configuration chain",
		"AWS_PROFILE",
		"AWS_REGION",
		"`aws-vault exec NAME -- tnnl ...`",
		"session-manager-plugin",
		"PATH",
	)
}

func TestVersionDefaultsToBuildInfo(t *testing.T) {
	if got, want := Version, buildinfo.Current(); got != want {
		t.Fatalf("Version = %q, want buildinfo.Current() %q", got, want)
	}
}

func TestVersionCommandsWriteIdenticalOutput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "subcommand", args: []string{"version"}},
		{name: "flag", args: []string{"--version"}},
	}
	outputs := make(map[string]string, len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			prepareRootCommandTest(t, tt.args, &stdout, &stderr)
			Version = "1.2.3"

			if err := ExecuteContext(context.Background()); err != nil {
				t.Fatalf("ExecuteContext() error = %v", err)
			}
			if got, want := stdout.String(), Version+"\n"; got != want {
				t.Errorf("version output = %q, want %q", got, want)
			}
			if got := stderr.String(); got != "" {
				t.Errorf("version error output = %q, want empty output", got)
			}
			outputs[tt.name] = stdout.String()
		})
	}

	if got, want := outputs["flag"], outputs["subcommand"]; got != want {
		t.Errorf("--version output = %q, want version output %q", got, want)
	}
}

func TestVersionCommandsReturnWriteErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "subcommand", args: []string{"version"}},
		{name: "flag", args: []string{"--version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErr := errors.New("write failure")
			prepareRootCommandTest(t, tt.args, failingWriter{err: wantErr}, io.Discard)
			Version = "1.2.3"

			err := ExecuteContext(context.Background())
			if !errors.Is(err, wantErr) {
				t.Fatalf("ExecuteContext() error = %v, want %v", err, wantErr)
			}
			if !strings.Contains(err.Error(), "write version") {
				t.Errorf("ExecuteContext() error = %q, want version write context", err)
			}
		})
	}
}

func prepareRootCommandTest(t *testing.T, args []string, stdout, stderr io.Writer) {
	t.Helper()

	originalRoot := RootCmd
	originalContext := RootCmd.Context()
	originalVersion := Version
	originalShortVersion := shortVersion
	versionFlag := RootCmd.Flags().Lookup("version")
	versionFlagChanged := false
	if versionFlag != nil {
		versionFlagChanged = versionFlag.Changed
	}

	RootCmd.SetArgs(args)
	RootCmd.SetOut(stdout)
	RootCmd.SetErr(stderr)
	shortVersion = false
	if versionFlag != nil {
		versionFlag.Changed = false
	}

	t.Cleanup(func() {
		RootCmd = originalRoot
		RootCmd.SetArgs(nil)
		RootCmd.SetOut(nil)
		RootCmd.SetErr(nil)
		RootCmd.SetContext(originalContext)
		Version = originalVersion
		shortVersion = originalShortVersion
		if versionFlag != nil {
			versionFlag.Changed = versionFlagChanged
		}
	})
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

type failingWriter struct {
	err error
}

func (w failingWriter) Write([]byte) (int, error) {
	return 0, w.err
}
