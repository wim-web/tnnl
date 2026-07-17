package input

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestResolveExecPrecedence(t *testing.T) {
	path := writeResolveFixture(t, "exec.json", `{"cluster":" c ","service":"s","command":"bash","wait":20}`)
	command := "zsh"
	wait := 0

	got, err := ResolveExec(path, ExecOverrides{Command: &command, Wait: &wait})
	if err != nil {
		t.Fatalf("ResolveExec() error = %v", err)
	}

	want := ExecInput{
		EcsParameter: EcsParameter{Cluster: "c", Service: "s"},
		Cmd:          "zsh",
		Wait:         0,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveExec() = %#v, want %#v", got, want)
	}
}

func TestResolveExecUsesDefaultWithoutFileOrOverride(t *testing.T) {
	got, err := ResolveExec("", ExecOverrides{})
	if err != nil {
		t.Fatalf("ResolveExec() error = %v", err)
	}

	want := ExecInput{Cmd: "sh", Wait: 0}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveExec() = %#v, want %#v", got, want)
	}
}

func TestResolveExecUsesNormalizedFileValuesWithoutOverrides(t *testing.T) {
	path := writeResolveFixture(t, "exec.json", `{"cluster":" cluster ","service":" service ","command":" bash -l ","wait":20}`)

	got, err := ResolveExec(path, ExecOverrides{})
	if err != nil {
		t.Fatalf("ResolveExec() error = %v", err)
	}

	want := ExecInput{
		EcsParameter: EcsParameter{Cluster: "cluster", Service: "service"},
		Cmd:          "bash -l",
		Wait:         20,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveExec() = %#v, want %#v", got, want)
	}
}

func TestResolveExecPropagatesStrictInputFileError(t *testing.T) {
	path := writeResolveFixture(t, "exec.json", `{"command":"sh","commnad":"bash"}`)

	got, err := ResolveExec(path, ExecOverrides{})
	if err == nil {
		t.Fatal("ResolveExec() error = nil, want strict input-file error")
	}
	for _, want := range []string{"decode input file", `unknown field "commnad"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ResolveExec() error = %q, want substring %q", err, want)
		}
	}
	if got != (ExecInput{}) {
		t.Fatalf("ResolveExec() value = %#v, want zero value on error", got)
	}
}

func TestResolvePortForwardAppliesExplicitOverridesAndNormalizes(t *testing.T) {
	path := writeResolveFixture(t, "port.json", `{
		"cluster":" cluster ",
		"service":" service ",
		"target_port_number":"80",
		"local_port_number":"8080"
	}`)
	targetPort := " 443 "
	localPort := " "

	got, err := ResolvePortForward(path, PortForwardOverrides{
		TargetPort: &targetPort,
		LocalPort:  &localPort,
	})
	if err != nil {
		t.Fatalf("ResolvePortForward() error = %v", err)
	}

	want := PortForwardInput{
		EcsParameter:     EcsParameter{Cluster: "cluster", Service: "service"},
		TargetPortNumber: "443",
		LocalPortNumber:  "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolvePortForward() = %#v, want %#v", got, want)
	}
}

func TestResolvePortForwardUsesFileValuesAndPreservesDefaultLocalPort(t *testing.T) {
	path := writeResolveFixture(t, "port.json", `{
		"cluster":" cluster ",
		"service":" service ",
		"target_port_number":" 80 "
	}`)

	got, err := ResolvePortForward(path, PortForwardOverrides{})
	if err != nil {
		t.Fatalf("ResolvePortForward() error = %v", err)
	}

	want := PortForwardInput{
		EcsParameter:     EcsParameter{Cluster: "cluster", Service: "service"},
		TargetPortNumber: "80",
		LocalPortNumber:  "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolvePortForward() = %#v, want %#v", got, want)
	}
}

func TestResolvePortForwardDefaultFailsRequiredTargetValidation(t *testing.T) {
	got, err := ResolvePortForward("", PortForwardOverrides{})
	if err == nil || !strings.Contains(err.Error(), "target port is required") {
		t.Fatalf("ResolvePortForward() error = %v, want target-port requirement", err)
	}
	if got != (PortForwardInput{}) {
		t.Fatalf("ResolvePortForward() value = %#v, want zero value on error", got)
	}
}

func TestResolvePortForwardReturnsZeroValueOnValidationError(t *testing.T) {
	targetPort := "abc"
	localPort := "0"

	got, err := ResolvePortForward("", PortForwardOverrides{
		TargetPort: &targetPort,
		LocalPort:  &localPort,
	})
	if err == nil {
		t.Fatal("ResolvePortForward() error = nil, want validation errors")
	}
	for _, want := range []string{"target port must be a decimal integer", "local port must be between 1 and 65535"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ResolvePortForward() error = %q, want substring %q", err, want)
		}
	}
	if got != (PortForwardInput{}) {
		t.Fatalf("ResolvePortForward() value = %#v, want zero value on error", got)
	}
}

func TestResolveRemotePortForwardAppliesExplicitOverridesAndNormalizes(t *testing.T) {
	path := writeResolveFixture(t, "remote-port.json", `{
		"cluster":" cluster ",
		"service":" service ",
		"remote_port_number":"22",
		"local_port_number":"2222",
		"host":"old.example.com"
	}`)
	remotePort := " 443 "
	localPort := " "
	host := " new.example.com "

	got, err := ResolveRemotePortForward(path, RemotePortForwardOverrides{
		RemotePort: &remotePort,
		LocalPort:  &localPort,
		Host:       &host,
	})
	if err != nil {
		t.Fatalf("ResolveRemotePortForward() error = %v", err)
	}

	want := RemotePortForwardInput{
		EcsParameter:     EcsParameter{Cluster: "cluster", Service: "service"},
		RemotePortNumber: "443",
		LocalPortNumber:  "",
		Host:             "new.example.com",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveRemotePortForward() = %#v, want %#v", got, want)
	}
}

func TestResolveRemotePortForwardUsesNormalizedFileValues(t *testing.T) {
	path := writeResolveFixture(t, "remote-port.json", `{
		"cluster":" cluster ",
		"service":" service ",
		"remote_port_number":" 22 ",
		"local_port_number":" 2222 ",
		"host":" example.com "
	}`)

	got, err := ResolveRemotePortForward(path, RemotePortForwardOverrides{})
	if err != nil {
		t.Fatalf("ResolveRemotePortForward() error = %v", err)
	}

	want := RemotePortForwardInput{
		EcsParameter:     EcsParameter{Cluster: "cluster", Service: "service"},
		RemotePortNumber: "22",
		LocalPortNumber:  "2222",
		Host:             "example.com",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ResolveRemotePortForward() = %#v, want %#v", got, want)
	}
}

func TestResolveRemotePortForwardDefaultJoinsRequiredValidationErrors(t *testing.T) {
	got, err := ResolveRemotePortForward("", RemotePortForwardOverrides{})
	if err == nil {
		t.Fatal("ResolveRemotePortForward() error = nil, want required-field errors")
	}
	for _, want := range []string{"remote port is required", "host is required"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ResolveRemotePortForward() error = %q, want substring %q", err, want)
		}
	}
	if got != (RemotePortForwardInput{}) {
		t.Fatalf("ResolveRemotePortForward() value = %#v, want zero value on error", got)
	}
}

func TestResolveRemotePortForwardReturnsZeroValueOnValidationError(t *testing.T) {
	remotePort := "abc"
	localPort := "0"
	host := " "

	got, err := ResolveRemotePortForward("", RemotePortForwardOverrides{
		RemotePort: &remotePort,
		LocalPort:  &localPort,
		Host:       &host,
	})
	if err == nil {
		t.Fatal("ResolveRemotePortForward() error = nil, want validation errors")
	}
	for _, want := range []string{
		"remote port must be a decimal integer",
		"local port must be between 1 and 65535",
		"host is required",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ResolveRemotePortForward() error = %q, want substring %q", err, want)
		}
	}
	if got != (RemotePortForwardInput{}) {
		t.Fatalf("ResolveRemotePortForward() value = %#v, want zero value on error", got)
	}
}

func writeResolveFixture(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write input fixture: %v", err)
	}
	return path
}
