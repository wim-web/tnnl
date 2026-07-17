package session_manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	helperModeEnv       = "GO_WANT_SESSION_MANAGER_HELPER"
	helperArgumentsEnv  = "SESSION_MANAGER_HELPER_ARGUMENTS"
	helperStartedEnv    = "SESSION_MANAGER_HELPER_STARTED"
	helperModeSuccess   = "success"
	helperModeFailure   = "failure"
	helperModeBlock     = "block"
	helperModeMarkStart = "mark-start"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(helperModeEnv); mode != "" {
		runHelperProcess(mode)
		return
	}

	os.Exit(m.Run())
}

func runHelperProcess(mode string) {
	if startedFile := os.Getenv(helperStartedEnv); startedFile != "" {
		if err := os.WriteFile(startedFile, []byte("started"), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}

	switch mode {
	case helperModeSuccess:
		arguments, err := json.Marshal(os.Args[1:])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := os.WriteFile(os.Getenv(helperArgumentsEnv), arguments, 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Fprintf(os.Stdout, "stdout:%s", input)
		fmt.Fprint(os.Stderr, "stderr")
		os.Exit(0)
	case helperModeFailure:
		fmt.Fprint(os.Stderr, "plugin failed")
		os.Exit(17)
	case helperModeBlock:
		for {
			time.Sleep(time.Hour)
		}
	case helperModeMarkStart:
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown helper mode %q", mode)
		os.Exit(2)
	}
}

func TestInvocationArgumentsMatchAWSCLIContract(t *testing.T) {
	invocation := Invocation{
		Response: SessionResponse{
			SessionID:  "s-1",
			StreamURL:  "wss://example",
			TokenValue: "token",
		},
		Region: "ap-northeast-1",
		Target: "ecs:cluster_task_runtime",
	}

	got, err := invocation.arguments("team-profile", "https://ssm.example")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 {
		t.Fatalf("len(arguments) = %d, want 6: %#v", len(got), got)
	}
	if got[1] != invocation.Region || got[2] != "StartSession" || got[3] != "team-profile" || got[5] != "https://ssm.example" {
		t.Fatalf("arguments = %#v", got)
	}

	var response map[string]string
	if err := json.Unmarshal([]byte(got[0]), &response); err != nil {
		t.Fatalf("decode response argument: %v", err)
	}
	wantResponse := map[string]string{
		"SessionId":  invocation.Response.SessionID,
		"StreamUrl":  invocation.Response.StreamURL,
		"TokenValue": invocation.Response.TokenValue,
	}
	if !reflect.DeepEqual(response, wantResponse) {
		t.Fatalf("response = %#v, want %#v", response, wantResponse)
	}

	var request map[string]string
	if err := json.Unmarshal([]byte(got[4]), &request); err != nil {
		t.Fatalf("decode request argument: %v", err)
	}
	wantRequest := map[string]string{"Target": invocation.Target}
	if !reflect.DeepEqual(request, wantRequest) {
		t.Fatalf("request = %#v, want %#v", request, wantRequest)
	}
}

func TestInvocationArgumentsAllowEmptyProfileAndEndpoint(t *testing.T) {
	got, err := validInvocation().arguments("", "")
	if err != nil {
		t.Fatal(err)
	}
	if got[3] != "" || got[5] != "" {
		t.Fatalf("arguments = %#v", got)
	}
}

func TestInvocationArgumentsRejectBlankRequiredFields(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		mutate func(*Invocation, string)
	}{
		{name: "empty session ID", value: "", mutate: func(i *Invocation, value string) { i.Response.SessionID = value }},
		{name: "whitespace session ID", value: " \t", mutate: func(i *Invocation, value string) { i.Response.SessionID = value }},
		{name: "empty stream URL", value: "", mutate: func(i *Invocation, value string) { i.Response.StreamURL = value }},
		{name: "whitespace stream URL", value: " \t", mutate: func(i *Invocation, value string) { i.Response.StreamURL = value }},
		{name: "empty token", value: "", mutate: func(i *Invocation, value string) { i.Response.TokenValue = value }},
		{name: "whitespace token", value: " \t", mutate: func(i *Invocation, value string) { i.Response.TokenValue = value }},
		{name: "empty region", value: "", mutate: func(i *Invocation, value string) { i.Region = value }},
		{name: "whitespace region", value: " \t", mutate: func(i *Invocation, value string) { i.Region = value }},
		{name: "empty target", value: "", mutate: func(i *Invocation, value string) { i.Target = value }},
		{name: "whitespace target", value: " \t", mutate: func(i *Invocation, value string) { i.Target = value }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			invocation := validInvocation()
			tt.mutate(&invocation, tt.value)

			arguments, err := invocation.arguments("profile", "endpoint")
			if err == nil {
				t.Fatalf("arguments() = %#v, want error", arguments)
			}
			if arguments != nil {
				t.Fatalf("arguments() = %#v, want nil on validation error", arguments)
			}
		})
	}
}

func TestPreflightReportsMissingPlugin(t *testing.T) {
	notFound := errors.New("executable file not found")
	deps := dependencies{
		lookPath: func(name string) (string, error) {
			if name != CommandName {
				t.Fatalf("lookPath name = %q, want %q", name, CommandName)
			}
			return "", notFound
		},
		commandContext: unusedProcessFactory(t),
		preflightLimit: time.Second,
	}

	_, err := preflight(context.Background(), deps)
	if !errors.Is(err, notFound) {
		t.Fatalf("preflight() error = %v, want errors.Is(notFound)", err)
	}
	if !strings.Contains(err.Error(), "install") || !strings.Contains(err.Error(), CommandName) {
		t.Fatalf("preflight() error = %q, want installation guidance", err)
	}
}

func TestPreflightReportsVersionFailureWithOutput(t *testing.T) {
	versionErr := errors.New("exit status 1")
	deps := dependencies{
		lookPath: func(string) (string, error) { return "/plugin", nil },
		commandContext: func(context.Context, string, ...string) command {
			return commandFunc(func() ([]byte, error) {
				return []byte("unsupported version\n"), versionErr
			})
		},
		preflightLimit: time.Second,
	}

	_, err := preflight(context.Background(), deps)
	if !errors.Is(err, versionErr) {
		t.Fatalf("preflight() error = %v, want errors.Is(versionErr)", err)
	}
	if !strings.Contains(err.Error(), "unsupported version") || !strings.Contains(err.Error(), "--version") {
		t.Fatalf("preflight() error = %q, want command output and operation", err)
	}
}

func TestPreflightRejectsEmptyVersionOutput(t *testing.T) {
	deps := dependencies{
		lookPath: func(string) (string, error) { return "/plugin", nil },
		commandContext: func(context.Context, string, ...string) command {
			return commandFunc(func() ([]byte, error) { return []byte(" \n\t"), nil })
		},
		preflightLimit: time.Second,
	}

	_, err := preflight(context.Background(), deps)
	if err == nil || !strings.Contains(err.Error(), "empty output") {
		t.Fatalf("preflight() error = %v, want empty output error", err)
	}
}

func TestPreflightDeadlineIsDiscoverable(t *testing.T) {
	processErr := errors.New("process killed")
	deps := dependencies{
		lookPath: func(string) (string, error) { return "/plugin", nil },
		commandContext: func(ctx context.Context, _ string, _ ...string) command {
			return commandFunc(func() ([]byte, error) {
				<-ctx.Done()
				return nil, processErr
			})
		},
		preflightLimit: 20 * time.Millisecond,
	}

	_, err := preflight(context.Background(), deps)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("preflight() error = %v, want errors.Is(DeadlineExceeded)", err)
	}
	if !errors.Is(err, processErr) {
		t.Fatalf("preflight() error = %v, want errors.Is(processErr)", err)
	}
}

func TestPreflightParentCancellationIsDiscoverable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	processErr := errors.New("process killed")
	deps := dependencies{
		lookPath: func(string) (string, error) { return "/plugin", nil },
		commandContext: func(ctx context.Context, _ string, _ ...string) command {
			return commandFunc(func() ([]byte, error) {
				<-ctx.Done()
				return nil, processErr
			})
		},
		preflightLimit: time.Second,
	}

	_, err := preflight(ctx, deps)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("preflight() error = %v, want errors.Is(Canceled)", err)
	}
	if !errors.Is(err, processErr) {
		t.Fatalf("preflight() error = %v, want errors.Is(processErr)", err)
	}
}

func TestPreflightSuccessUsesExactVersionCommandAndEnvironment(t *testing.T) {
	tests := []struct {
		name           string
		profile        string
		defaultProfile string
		ssmEndpoint    string
		globalEndpoint string
		wantProfile    string
		wantEndpoint   string
	}{
		{
			name:           "service-specific values win",
			profile:        "primary-profile",
			defaultProfile: "fallback-profile",
			ssmEndpoint:    "https://ssm.example",
			globalEndpoint: "https://global.example",
			wantProfile:    "primary-profile",
			wantEndpoint:   "https://ssm.example",
		},
		{
			name:           "fallback values are used",
			defaultProfile: "fallback-profile",
			globalEndpoint: "https://global.example",
			wantProfile:    "fallback-profile",
			wantEndpoint:   "https://global.example",
		},
		{name: "unset values stay empty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AWS_PROFILE", tt.profile)
			t.Setenv("AWS_DEFAULT_PROFILE", tt.defaultProfile)
			t.Setenv("AWS_ENDPOINT_URL_SSM", tt.ssmEndpoint)
			t.Setenv("AWS_ENDPOINT_URL", tt.globalEndpoint)

			var gotName string
			var gotArguments []string
			deps := dependencies{
				lookPath: func(name string) (string, error) {
					if name != CommandName {
						t.Fatalf("lookPath name = %q, want %q", name, CommandName)
					}
					return "/resolved/session-manager-plugin", nil
				},
				commandContext: func(_ context.Context, name string, arguments ...string) command {
					gotName = name
					gotArguments = append([]string(nil), arguments...)
					return commandFunc(func() ([]byte, error) { return []byte("1.2.3\n"), nil })
				},
				preflightLimit: time.Second,
			}

			runner, err := preflight(context.Background(), deps)
			if err != nil {
				t.Fatal(err)
			}
			if gotName != "/resolved/session-manager-plugin" {
				t.Fatalf("command name = %q", gotName)
			}
			if !reflect.DeepEqual(gotArguments, []string{"--version"}) {
				t.Fatalf("command arguments = %#v, want [--version]", gotArguments)
			}
			if runner.path != gotName || runner.profile != tt.wantProfile || runner.endpoint != tt.wantEndpoint {
				t.Fatalf("runner = %#v, want path=%q profile=%q endpoint=%q", runner, gotName, tt.wantProfile, tt.wantEndpoint)
			}
		})
	}
}

func TestRunnerValidationErrorDoesNotStartProcess(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	startedFile := t.TempDir() + "/started"
	t.Setenv(helperModeEnv, helperModeMarkStart)
	t.Setenv(helperStartedEnv, startedFile)
	invocation := validInvocation()
	invocation.Response.TokenValue = " \t"

	err = (&Runner{path: executable}).Run(context.Background(), invocation)
	if err == nil {
		t.Fatal("Run() error = nil, want validation error")
	}
	if _, statErr := os.Stat(startedFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("helper process started: os.Stat() error = %v", statErr)
	}
}

func TestRunnerRunPassesSixArgumentsAndAssignsStreams(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(helperModeEnv, helperModeSuccess)
	argumentsFile := t.TempDir() + "/arguments.json"
	t.Setenv(helperArgumentsEnv, argumentsFile)

	stdin := openTestFile(t, "stdin", "input")
	stdout := openTestFile(t, "stdout", "")
	stderr := openTestFile(t, "stderr", "")
	originalStdin, originalStdout, originalStderr := os.Stdin, os.Stdout, os.Stderr
	os.Stdin, os.Stdout, os.Stderr = stdin, stdout, stderr
	t.Cleanup(func() {
		os.Stdin, os.Stdout, os.Stderr = originalStdin, originalStdout, originalStderr
	})

	invocation := validInvocation()
	runner := &Runner{path: executable, profile: "team-profile", endpoint: "https://ssm.example"}
	if err := runner.Run(context.Background(), invocation); err != nil {
		t.Fatal(err)
	}

	argumentJSON, err := os.ReadFile(argumentsFile)
	if err != nil {
		t.Fatal(err)
	}
	var gotArguments []string
	if err := json.Unmarshal(argumentJSON, &gotArguments); err != nil {
		t.Fatal(err)
	}
	wantArguments, err := invocation.arguments(runner.profile, runner.endpoint)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotArguments, wantArguments) {
		t.Fatalf("process arguments = %#v, want %#v", gotArguments, wantArguments)
	}
	if len(gotArguments) != 6 {
		t.Fatalf("len(process arguments) = %d, want 6", len(gotArguments))
	}
	if got := readTestFile(t, stdout); got != "stdout:input" {
		t.Fatalf("stdout = %q, want %q", got, "stdout:input")
	}
	if got := readTestFile(t, stderr); got != "stderr" {
		t.Fatalf("stderr = %q, want %q", got, "stderr")
	}
}

func TestRunnerRunWrapsProcessError(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(helperModeEnv, helperModeFailure)

	err = (&Runner{path: executable}).Run(context.Background(), validInvocation())
	if err == nil || !strings.Contains(err.Error(), "run "+CommandName) {
		t.Fatalf("Run() error = %v, want wrapped process error", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("Run() error = %v, want wrapped *exec.ExitError", err)
	}
}

func TestRunnerRunPreservesCancellation(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(helperModeEnv, helperModeBlock)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	timer := time.AfterFunc(20*time.Millisecond, cancel)
	defer timer.Stop()

	err = (&Runner{path: executable}).Run(ctx, validInvocation())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want errors.Is(Canceled)", err)
	}
	assertWrappedProcessError(t, err)
}

func TestRunnerRunPreservesDeadline(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(helperModeEnv, helperModeBlock)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = (&Runner{path: executable}).Run(ctx, validInvocation())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want errors.Is(DeadlineExceeded)", err)
	}
	assertWrappedProcessError(t, err)
}

func assertWrappedProcessError(t *testing.T, err error) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "run "+CommandName) {
		t.Fatalf("error = %v, want %q wrapper", err, "run "+CommandName)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error = %v, want wrapped *exec.ExitError", err)
	}
}

type commandFunc func() ([]byte, error)

func (f commandFunc) CombinedOutput() ([]byte, error) {
	return f()
}

func unusedProcessFactory(t *testing.T) processFactory {
	t.Helper()
	return func(context.Context, string, ...string) command {
		t.Fatal("commandContext called unexpectedly")
		return nil
	}
}

func validInvocation() Invocation {
	return Invocation{
		Response: SessionResponse{
			SessionID:  "s-1",
			StreamURL:  "wss://example",
			TokenValue: "token",
		},
		Region: "ap-northeast-1",
		Target: "ecs:cluster_task_runtime",
	}
}

func openTestFile(t *testing.T, name, contents string) *os.File {
	t.Helper()
	path := t.TempDir() + "/" + name
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = file.Close() })
	if _, err := file.WriteString(contents); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	return file
}

func readTestFile(t *testing.T, file *os.File) string {
	t.Helper()
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	contents, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(contents)
}
