package handler

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/wim-web/tnnl/internal/input"
	"github.com/wim-web/tnnl/internal/listview"
	"github.com/wim-web/tnnl/internal/session_manager"
	"github.com/wim-web/tnnl/pkg/command"
)

func TestPortForwardHandlerPreflightFailureStopsBeforeAWS(t *testing.T) {
	preflightErr := errors.New("preflight sentinel")
	var events []string
	deps := dependencies{
		preflight: func(context.Context) (session_manager.Plugin, error) {
			appendEvent(&events, "preflight")
			return nil, preflightErr
		},
		loadConfig: func(context.Context) (aws.Config, error) {
			t.Fatal("loadConfig called after preflight failure")
			return aws.Config{}, nil
		},
		newECS: func(aws.Config) ecsAPI { t.Fatal("newECS called"); return nil },
		newSSM: func(aws.Config) ssmAPI { t.Fatal("newSSM called"); return nil },
		choose: func(string, []listview.Option) (string, bool, error) {
			t.Fatal("chooser called")
			return "", false, nil
		},
		availablePort: func() (int, error) { t.Fatal("availablePort called"); return 0, nil },
	}

	err := portForwardHandler(context.Background(), validPortHandlerInput(""), deps)
	if !errors.Is(err, preflightErr) {
		t.Fatalf("portForwardHandler() error = %v, want errors.Is(preflightErr)", err)
	}
	if !reflect.DeepEqual(events, []string{"preflight"}) {
		t.Fatalf("events = %#v, want preflight only", events)
	}
}

func TestPortForwardHandlerSelectsSecondReplicaAndAllocatesLocalPort(t *testing.T) {
	ctx := context.WithValue(context.Background(), handlerContextKey{}, "port-context")
	var events []string
	ecsClient := newHandlerECS(&events)
	ssmClient := &handlerSSM{events: &events, startOutput: validHandlerStartOutput()}
	plugin := &handlerPlugin{events: &events}
	deps := handlerDependencies(t, &events, ecsClient, ssmClient, plugin)
	availableCalls := 0
	deps.availablePort = func() (int, error) {
		availableCalls++
		appendEvent(&events, "available-port")
		return 49152, nil
	}

	err := portForwardHandler(ctx, validPortHandlerInput(""), deps)
	if err != nil {
		t.Fatalf("portForwardHandler() error = %v", err)
	}
	wantEvents := []string{
		"preflight", "load-config", "list-tasks", "describe-targets", "choose-task",
		"available-port", "start-session", "plugin-run",
	}
	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("events = %#v, want %#v", events, wantEvents)
	}
	if availableCalls != 1 {
		t.Fatalf("availablePort calls = %d, want 1", availableCalls)
	}
	if ssmClient.startCalls != 1 || ssmClient.startCtx != ctx {
		t.Fatalf("StartSession calls/context = %d/%v, want 1/original %v", ssmClient.startCalls, ssmClient.startCtx, ctx)
	}
	wantTarget := "ecs:production_task-second_runtime-second"
	if got := aws.ToString(ssmClient.startInput.Target); got != wantTarget {
		t.Fatalf("StartSession target = %q, want second replica %q", got, wantTarget)
	}
	if got := aws.ToString(ssmClient.startInput.DocumentName); got != string(command.PORT_FORWARD_DOCUMENT_NAME) {
		t.Fatalf("StartSession document = %q", got)
	}
	wantParams := map[string][]string{
		"portNumber":      {"5432"},
		"localPortNumber": {"49152"},
	}
	if !reflect.DeepEqual(ssmClient.startInput.Parameters, wantParams) {
		t.Fatalf("StartSession parameters = %#v, want %#v", ssmClient.startInput.Parameters, wantParams)
	}
	if plugin.invocation.Target != wantTarget || plugin.invocation.Region != handlerRegion {
		t.Fatalf("plugin invocation = %#v", plugin.invocation)
	}
}

func TestPortForwardHandlerExplicitPortSkipsAllocation(t *testing.T) {
	var events []string
	ecsClient := newHandlerECS(&events)
	ssmClient := &handlerSSM{events: &events, startOutput: validHandlerStartOutput()}
	plugin := &handlerPlugin{events: &events}
	deps := handlerDependencies(t, &events, ecsClient, ssmClient, plugin)
	deps.availablePort = func() (int, error) {
		t.Fatal("availablePort called for explicit local port")
		return 0, nil
	}

	if err := portForwardHandler(context.Background(), validPortHandlerInput("6000"), deps); err != nil {
		t.Fatalf("portForwardHandler() error = %v", err)
	}
	if got := ssmClient.startInput.Parameters["localPortNumber"]; !reflect.DeepEqual(got, []string{"6000"}) {
		t.Fatalf("localPortNumber = %#v, want [6000]", got)
	}
}

func TestPortForwardHandlerAllocationFailureDoesNotStartSession(t *testing.T) {
	allocationErr := errors.New("allocation sentinel")
	tests := []struct {
		name string
		port int
		err  error
	}{
		{name: "allocator error", err: allocationErr},
		{name: "zero port", port: 0},
		{name: "negative port", port: -1},
		{name: "port above maximum", port: 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var events []string
			ecsClient := newHandlerECS(&events)
			ssmClient := &handlerSSM{events: &events, startOutput: validHandlerStartOutput()}
			deps := handlerDependencies(t, &events, ecsClient, ssmClient, &handlerPlugin{events: &events})
			deps.availablePort = func() (int, error) {
				appendEvent(&events, "available-port")
				return tt.port, tt.err
			}

			err := portForwardHandler(context.Background(), validPortHandlerInput(""), deps)
			if err == nil {
				t.Fatal("portForwardHandler() error = nil")
			}
			if tt.err != nil && !errors.Is(err, allocationErr) {
				t.Fatalf("error = %v, want errors.Is(allocationErr)", err)
			}
			if tt.err == nil && !strings.Contains(err.Error(), strconv.Itoa(tt.port)) {
				t.Fatalf("error = %q, want invalid port %d", err, tt.port)
			}
			if ssmClient.startCalls != 0 {
				t.Fatalf("StartSession calls = %d, want 0", ssmClient.startCalls)
			}
		})
	}
}

func TestRemotePortForwardHandlerBuildsExactParameters(t *testing.T) {
	var events []string
	ecsClient := newHandlerECS(&events)
	ssmClient := &handlerSSM{events: &events, startOutput: validHandlerStartOutput()}
	plugin := &handlerPlugin{events: &events}
	deps := handlerDependencies(t, &events, ecsClient, ssmClient, plugin)
	deps.availablePort = func() (int, error) {
		t.Fatal("availablePort called for explicit remote forward port")
		return 0, nil
	}
	in := input.RemotePortForwardInput{
		EcsParameter:     input.EcsParameter{Cluster: handlerClusterARN, Service: "service-web"},
		RemotePortNumber: "3306",
		LocalPortNumber:  "13306",
		Host:             "db.internal",
	}

	if err := remotePortForwardHandler(context.Background(), in, deps); err != nil {
		t.Fatalf("remotePortForwardHandler() error = %v", err)
	}
	if got := aws.ToString(ssmClient.startInput.DocumentName); got != string(command.REMOTE_PORT_FORWARD_DOCUMENT_NAME) {
		t.Fatalf("document = %q", got)
	}
	want := map[string][]string{
		"portNumber":      {"3306"},
		"localPortNumber": {"13306"},
		"host":            {"db.internal"},
	}
	if !reflect.DeepEqual(ssmClient.startInput.Parameters, want) {
		t.Fatalf("parameters = %#v, want %#v", ssmClient.startInput.Parameters, want)
	}
}

func TestPortForwardHandlerPluginFailureTerminatesAfterPlugin(t *testing.T) {
	pluginErr := errors.New("plugin sentinel")
	var events []string
	ecsClient := newHandlerECS(&events)
	ssmClient := &handlerSSM{events: &events, startOutput: validHandlerStartOutput()}
	plugin := &handlerPlugin{events: &events, run: func(context.Context, session_manager.Invocation) error {
		return pluginErr
	}}
	deps := handlerDependencies(t, &events, ecsClient, ssmClient, plugin)
	deps.availablePort = func() (int, error) { return 49152, nil }

	err := portForwardHandler(context.Background(), validPortHandlerInput(""), deps)
	if !errors.Is(err, pluginErr) {
		t.Fatalf("error = %v, want errors.Is(pluginErr)", err)
	}
	if ssmClient.terminateCalls != 1 || aws.ToString(ssmClient.terminateInput.SessionId) != handlerSessionID {
		t.Fatalf("TerminateSession calls/input = %d/%#v", ssmClient.terminateCalls, ssmClient.terminateInput)
	}
	if !reflect.DeepEqual(events[len(events)-2:], []string{"plugin-run", "terminate-session"}) {
		t.Fatalf("last events = %#v", events)
	}
}

func TestPortForwardHandlerViewCancellationDoesNotAllocateOrStart(t *testing.T) {
	var events []string
	ecsClient := newHandlerECS(&events)
	ssmClient := &handlerSSM{events: &events, startOutput: validHandlerStartOutput()}
	plugin := &handlerPlugin{events: &events}
	deps := handlerDependencies(t, &events, ecsClient, ssmClient, plugin)
	deps.choose = func(string, []listview.Option) (string, bool, error) {
		appendEvent(&events, "choose-task")
		return "", true, nil
	}
	deps.availablePort = func() (int, error) {
		t.Fatal("availablePort called after view cancellation")
		return 0, nil
	}

	if err := portForwardHandler(context.Background(), validPortHandlerInput(""), deps); err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if ssmClient.startCalls != 0 || plugin.calls != 0 {
		t.Fatalf("calls after cancellation: start=%d plugin=%d", ssmClient.startCalls, plugin.calls)
	}
}

func validPortHandlerInput(localPort string) input.PortForwardInput {
	return input.PortForwardInput{
		EcsParameter:     input.EcsParameter{Cluster: handlerClusterARN, Service: "service-web"},
		TargetPortNumber: "5432",
		LocalPortNumber:  localPort,
	}
}

func validHandlerStartOutput() *ssm.StartSessionOutput {
	return &ssm.StartSessionOutput{
		SessionId:  aws.String(handlerSessionID),
		StreamUrl:  aws.String("wss://handler.example"),
		TokenValue: aws.String("handler-token"),
	}
}
